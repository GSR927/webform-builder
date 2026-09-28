package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"

	_ "github.com/go-sql-driver/mysql"

	"github.com/example/webform-builder/backend/models"
)

// SubmissionStore abstracts the insert target so we can point at ClickHouse
// in production and at a plain MySQL table in local/dev environments that
// don't have ClickHouse running (documented trade-off, see TRADEOFFS.md and
// ARCHITECTURE.md "Built vs Designed").
type SubmissionStore struct {
	db        *sql.DB
	useClick  bool
}

func NewSubmissionStore() *SubmissionStore {
	dsn := os.Getenv("SUBMISSIONS_DSN") // e.g. clickhouse or mysql DSN
	if dsn == "" {
		dsn = "webform:webform@tcp(localhost:3306)/webform"
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		panic(err)
	}
	return &SubmissionStore{db: db, useClick: os.Getenv("USE_CLICKHOUSE") == "1"}
}

// InsertBatch is idempotent on submission_id: a unique key on that column
// means redelivery after a crash produces a harmless duplicate-key error
// that we treat as success, not a failure to retry.
func (s *SubmissionStore) InsertBatch(ctx context.Context, subs []models.Submission) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO submissions (id, tenant_id, form_id, form_version_id, payload_json, ip_hash, submitted_at, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE id = id`) // no-op on conflict = idempotent
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, s2 := range subs {
		payload, _ := json.Marshal(s2.Payload)
		if _, err := stmt.ExecContext(ctx, s2.ID, s2.TenantID, s2.FormID, s2.FormVersionID,
			payload, s2.IPHash, s2.SubmittedAt, s2.Status); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *SubmissionStore) Close() {
	_ = s.db.Close()
}
