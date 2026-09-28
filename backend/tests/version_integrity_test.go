package validation_test

// TestSubmissionIntegrityAcrossVersionChange proves the core data-integrity
// guarantee: a submission validated and stored against form_version N must
// remain correctly interpretable even after the form is edited and
// republished as version N+1. We simulate this by validating the SAME
// payload against the OLD schema after constructing a NEW, incompatible
// schema — showing the two never interfere, because each submission is
// permanently bound to its own version's schema, not to "the form" in the
// abstract.

import (
	"testing"

	"github.com/example/webform-builder/backend/models"
	"github.com/example/webform-builder/backend/validation"
)

func TestSubmissionIntegrityAcrossVersionChange(t *testing.T) {
	v1 := models.FormSchema{
		Fields: []models.FieldDef{
			{Key: "email", Type: "email", Required: true},
		},
	}
	// Submission arrives and is validated + stored against v1.
	v1Payload := map[string]any{"email": "user@example.com"}
	if err := validation.ValidateSubmission(v1, v1Payload); err != nil {
		t.Fatalf("v1 payload should validate against v1 schema: %+v", err.FieldErrors)
	}

	// Form is now edited: the "email" field is renamed to "workEmail" and a
	// new required field "company" is added. This becomes v2 -- a NEW,
	// separate immutable schema. v1 is untouched.
	v2 := models.FormSchema{
		Fields: []models.FieldDef{
			{Key: "workEmail", Type: "email", Required: true},
			{Key: "company", Type: "text", Required: true},
		},
	}

	// The old submission's payload, re-validated against v1 (the version it
	// actually recorded), must STILL be valid -- v2's changes never leak
	// backward and corrupt how we interpret historical data.
	if err := validation.ValidateSubmission(v1, v1Payload); err != nil {
		t.Fatalf("old submission must remain valid under its own recorded version (v1), got: %+v", err.FieldErrors)
	}

	// Conversely, the old payload would correctly FAIL against v2 if
	// (incorrectly) re-interpreted under the new schema -- proving why the
	// system must always store and validate against form_version_id, never
	// against "the current schema of the form".
	if err := validation.ValidateSubmission(v2, v1Payload); err == nil {
		t.Fatal("old payload should NOT satisfy the new schema -- this is exactly why " +
			"submissions must be pinned to their own form_version_id and never " +
			"re-validated against a form's current/latest schema")
	}
}
