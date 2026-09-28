package handlers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/example/webform-builder/backend/models"
	"github.com/example/webform-builder/backend/queue"
	"github.com/example/webform-builder/backend/ratelimit"
	"github.com/example/webform-builder/backend/validation"
)

type SubmissionsHandler struct {
	DB        *sql.DB
	Publisher *queue.Publisher
	Limiter   *ratelimit.TokenBucketLimiter
}

type submitRequest struct {
	VersionID int64          `json:"versionId"`
	Payload   map[string]any `json:"payload"`
	Honeypot  string         `json:"hp,omitempty"` // hidden field; humans leave it empty
}

// Submit is the hot path: public, anonymous, bursty. Every step is ordered
// so we fail cheap requests fast (rate limit, honeypot) before doing the
// more expensive schema fetch + validation + durable publish.
func (h *SubmissionsHandler) Submit(w http.ResponseWriter, r *http.Request) {
	formID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	ip := clientIP(r)
	ipHash := hashIP(ip)

	// 1. Rate limit first — cheapest possible rejection for abusive traffic.
	if !h.Limiter.Allow(formIDKey(formID), ip) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	var req submitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	// 2. Honeypot: a field real users never see or fill. Non-empty = bot.
	if req.Honeypot != "" {
		w.WriteHeader(http.StatusOK) // pretend success so bots don't learn
		return
	}

	// 3. Fetch the EXACT version schema the client claims to have rendered.
	//    This, not the client's own declared field types, is the source of
	//    truth for validation.
	var tenantID int64
	var schemaJSON []byte
	err := h.DB.QueryRowContext(r.Context(),
		`SELECT tenant_id, schema_json FROM form_versions WHERE id = ? AND form_id = ?`,
		req.VersionID, formID,
	).Scan(&tenantID, &schemaJSON)
	if err != nil {
		http.Error(w, "form version not found", http.StatusNotFound)
		return
	}

	var schema models.FormSchema
	if err := json.Unmarshal(schemaJSON, &schema); err != nil {
		http.Error(w, "internal schema error", http.StatusInternalServerError)
		return
	}

	// 4. Server-side validation, re-derived from the schema, never the client.
	if verr := validation.ValidateSubmission(schema, req.Payload); verr != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"errors": verr.FieldErrors})
		return
	}

	submission := models.Submission{
		ID:            uuid.NewString(), // generated here, not by the consumer -> idempotency key
		TenantID:      tenantID,
		FormID:        formID,
		FormVersionID: req.VersionID,
		Payload:       req.Payload,
		IPHash:        ipHash,
		SubmittedAt:   time.Now(),
		Status:        "accepted",
	}

	// 5. Durability commit point: block on RabbitMQ's publish confirm. Only
	//    after this succeeds do we tell the end-user "accepted".
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := h.Publisher.PublishSubmission(ctx, submission); err != nil {
		// 6. Fallback: RabbitMQ unavailable/slow -> write to the MySQL inbox
		//    directly so we still never lose the submission before ACKing.
		if fallbackErr := h.writeInboxFallback(submission); fallbackErr != nil {
			http.Error(w, "failed to durably store submission", http.StatusServiceUnavailable)
			return
		}
	}

	writeJSON(w, http.StatusAccepted, map[string]any{"submissionId": submission.ID})
}

func (h *SubmissionsHandler) writeInboxFallback(s models.Submission) error {
	payload, _ := json.Marshal(s.Payload)
	_, err := h.DB.ExecContext(context.Background(),
		`INSERT INTO submissions_inbox (id, tenant_id, form_id, form_version_id, payload_json, received_at, status)
		 VALUES (?, ?, ?, ?, ?, ?, 'pending_consume')`,
		s.ID, s.TenantID, s.FormID, s.FormVersionID, payload, s.SubmittedAt,
	)
	return err
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	return r.RemoteAddr
}

func hashIP(ip string) string {
	sum := sha256.Sum256([]byte(ip))
	return fmt.Sprintf("%x", sum)
}

func formIDKey(id int64) string {
	return fmt.Sprintf("form:%d", id)
}
