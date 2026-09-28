package models

import "time"

// FieldValidators holds optional constraints for a field. Extend as needed.
type FieldValidators struct {
	MinLength *int     `json:"minLength,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Pattern   *string  `json:"pattern,omitempty"`
	Options   []string `json:"options,omitempty"` // for select/radio/checkbox
}

// ShowIf expresses a simple conditional-visibility rule: show this field only
// if `Field` equals `Equals`. Kept intentionally simple for the working slice;
// designed extension point for AND/OR groups is noted in ARCHITECTURE.md.
type ShowIf struct {
	Field  string `json:"field"`
	Equals string `json:"equals"`
}

// FieldDef is one user-defined form field. This is the atomic unit of the
// dynamic schema stored as JSON in form_versions.schema_json.
type FieldDef struct {
	Key        string           `json:"key"`
	Type       string           `json:"type"` // text|email|number|select|multiselect|radio|checkbox|date
	Label      string           `json:"label"`
	HelpText   string           `json:"helpText,omitempty"`
	Required   bool             `json:"required"`
	Validators FieldValidators  `json:"validators,omitempty"`
	ShowIf     *ShowIf          `json:"showIf,omitempty"`
}

// FormSchema is the full set of fields for one form version.
type FormSchema struct {
	Fields []FieldDef `json:"fields"`
}

type Form struct {
	ID                int64      `json:"id"`
	TenantID          int64      `json:"tenantId"`
	Name              string     `json:"name"`
	Status            string     `json:"status"` // draft|published|archived
	DraftSchema       FormSchema `json:"draftSchema"`
	CurrentVersionID  *int64     `json:"currentVersionId,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type FormVersion struct {
	ID            int64      `json:"id"`
	FormID        int64      `json:"formId"`
	TenantID      int64      `json:"tenantId"`
	VersionNumber int        `json:"versionNumber"`
	Schema        FormSchema `json:"schema"`
	PublishedAt   time.Time  `json:"publishedAt"`
}
