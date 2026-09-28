package validation_test

import (
	"testing"

	"github.com/example/webform-builder/backend/models"
	"github.com/example/webform-builder/backend/validation"
)

func schemaFixture() models.FormSchema {
	maxLen := 50
	minVal, maxVal := 18.0, 99.0
	return models.FormSchema{
		Fields: []models.FieldDef{
			{Key: "name", Type: "text", Required: true, Validators: models.FieldValidators{MaxLength: &maxLen}},
			{Key: "email", Type: "email", Required: true},
			{Key: "age", Type: "number", Required: false, Validators: models.FieldValidators{Min: &minVal, Max: &maxVal}},
			{Key: "role", Type: "select", Required: true, Validators: models.FieldValidators{Options: []string{"buyer", "seller"}}},
			{
				Key: "companyName", Type: "text", Required: true,
				ShowIf: &models.ShowIf{Field: "role", Equals: "seller"},
			},
		},
	}
}

func TestValidSubmissionPasses(t *testing.T) {
	schema := schemaFixture()
	payload := map[string]any{
		"name": "Asha Menon", "email": "asha@example.com", "age": 29.0, "role": "buyer",
	}
	if err := validation.ValidateSubmission(schema, payload); err != nil {
		t.Fatalf("expected valid submission, got errors: %+v", err.FieldErrors)
	}
}

func TestMissingRequiredFieldFails(t *testing.T) {
	schema := schemaFixture()
	payload := map[string]any{"email": "asha@example.com", "role": "buyer"}
	err := validation.ValidateSubmission(schema, payload)
	if err == nil {
		t.Fatal("expected validation error for missing required 'name'")
	}
	if _, ok := err.FieldErrors["name"]; !ok {
		t.Errorf("expected error on 'name', got: %+v", err.FieldErrors)
	}
}

func TestInvalidEmailFails(t *testing.T) {
	schema := schemaFixture()
	payload := map[string]any{"name": "Asha", "email": "not-an-email", "role": "buyer"}
	err := validation.ValidateSubmission(schema, payload)
	if err == nil || err.FieldErrors["email"] == "" {
		t.Fatal("expected email validation error")
	}
}

func TestUnknownFieldRejected(t *testing.T) {
	schema := schemaFixture()
	payload := map[string]any{
		"name": "Asha", "email": "asha@example.com", "role": "buyer",
		"injected_script": "<script>alert(1)</script>",
	}
	err := validation.ValidateSubmission(schema, payload)
	if err == nil || err.FieldErrors["injected_script"] == "" {
		t.Fatal("expected unknown-field rejection to catch unexpected keys")
	}
}

func TestConditionalFieldNotRequiredWhenHidden(t *testing.T) {
	schema := schemaFixture()
	// role=buyer -> companyName's showIf (role=seller) is false -> not required
	payload := map[string]any{"name": "Asha", "email": "asha@example.com", "role": "buyer"}
	if err := validation.ValidateSubmission(schema, payload); err != nil {
		t.Fatalf("companyName should not be required when hidden, got: %+v", err.FieldErrors)
	}
}

func TestConditionalFieldRequiredWhenVisible(t *testing.T) {
	schema := schemaFixture()
	// role=seller -> companyName becomes required and is missing
	payload := map[string]any{"name": "Asha", "email": "asha@example.com", "role": "seller"}
	err := validation.ValidateSubmission(schema, payload)
	if err == nil || err.FieldErrors["companyName"] == "" {
		t.Fatal("expected companyName to be required once role=seller makes it visible")
	}
}

func TestOutOfRangeNumberFails(t *testing.T) {
	schema := schemaFixture()
	payload := map[string]any{"name": "Asha", "email": "a@example.com", "role": "buyer", "age": 150.0}
	err := validation.ValidateSubmission(schema, payload)
	if err == nil || err.FieldErrors["age"] == "" {
		t.Fatal("expected age out-of-range validation error")
	}
}
