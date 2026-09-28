package validation

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/example/webform-builder/backend/models"
)

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// ValidationError collects all field errors for a single submission so the
// caller can return a complete, useful error response in one round trip.
type ValidationError struct {
	FieldErrors map[string]string
}

func (v *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %d field(s)", len(v.FieldErrors))
}

// isVisible re-evaluates a field's conditional visibility against the actual
// submitted payload. A field hidden by its own showIf rule is not required,
// regardless of what the client sent — this is what makes validation
// authoritative rather than trusting client-side conditional logic.
func isVisible(f models.FieldDef, payload map[string]any) bool {
	if f.ShowIf == nil {
		return true
	}
	val, ok := payload[f.ShowIf.Field]
	if !ok {
		return false
	}
	return fmt.Sprintf("%v", val) == f.ShowIf.Equals
}

// ValidateSubmission is THE server-side validation entry point. It is always
// re-derived from the immutable schema of the exact form_version the client
// says it submitted against — never from client-declared types or rules.
func ValidateSubmission(schema models.FormSchema, payload map[string]any) *ValidationError {
	errs := map[string]string{}

	for _, field := range schema.Fields {
		visible := isVisible(field, payload)
		raw, present := payload[field.Key]

		if !visible {
			// Hidden fields must not be required, and their value (if any) is
			// ignored rather than validated — the client may have sent stale
			// data for a field it hid client-side.
			continue
		}

		if field.Required && (!present || raw == nil || raw == "") {
			errs[field.Key] = "this field is required"
			continue
		}
		if !present || raw == nil {
			continue // optional and absent, nothing further to check
		}

		switch field.Type {
		case "text", "textarea":
			s, ok := raw.(string)
			if !ok {
				errs[field.Key] = "must be a string"
				break
			}
			if field.Validators.MinLength != nil && len(s) < *field.Validators.MinLength {
				errs[field.Key] = fmt.Sprintf("must be at least %d characters", *field.Validators.MinLength)
			}
			if field.Validators.MaxLength != nil && len(s) > *field.Validators.MaxLength {
				errs[field.Key] = fmt.Sprintf("must be at most %d characters", *field.Validators.MaxLength)
			}
			if field.Validators.Pattern != nil {
				re, err := regexp.Compile(*field.Validators.Pattern)
				if err == nil && !re.MatchString(s) {
					errs[field.Key] = "does not match required pattern"
				}
			}

		case "email":
			s, ok := raw.(string)
			if !ok || !emailRe.MatchString(s) {
				errs[field.Key] = "must be a valid email address"
			}

		case "number":
			var num float64
			switch v := raw.(type) {
			case float64:
				num = v
			case string:
				parsed, err := strconv.ParseFloat(v, 64)
				if err != nil {
					errs[field.Key] = "must be a number"
					continue
				}
				num = parsed
			default:
				errs[field.Key] = "must be a number"
				continue
			}
			if field.Validators.Min != nil && num < *field.Validators.Min {
				errs[field.Key] = fmt.Sprintf("must be >= %v", *field.Validators.Min)
			}
			if field.Validators.Max != nil && num > *field.Validators.Max {
				errs[field.Key] = fmt.Sprintf("must be <= %v", *field.Validators.Max)
			}

		case "select", "radio":
			s, ok := raw.(string)
			if !ok || !contains(field.Validators.Options, s) {
				errs[field.Key] = "must be one of the allowed options"
			}

		case "multiselect", "checkbox":
			items, ok := raw.([]any)
			if !ok {
				errs[field.Key] = "must be a list of options"
				break
			}
			for _, it := range items {
				s := fmt.Sprintf("%v", it)
				if !contains(field.Validators.Options, s) {
					errs[field.Key] = "contains an invalid option"
					break
				}
			}

		case "date":
			s, ok := raw.(string)
			if !ok {
				errs[field.Key] = "must be a date string (YYYY-MM-DD)"
				break
			}
			if _, err := time.Parse("2006-01-02", s); err != nil {
				errs[field.Key] = "must be a valid date (YYYY-MM-DD)"
			}

		default:
			errs[field.Key] = fmt.Sprintf("unknown field type %q", field.Type)
		}
	}

	// Reject any submitted key that isn't part of the schema at all — closes
	// off a class of injection attempts that rely on unexpected fields.
	known := map[string]bool{}
	for _, f := range schema.Fields {
		known[f.Key] = true
	}
	for k := range payload {
		if !known[k] {
			errs[k] = "unknown field"
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return &ValidationError{FieldErrors: errs}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}
