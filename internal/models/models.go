package models

import "time"

type FieldType string

const (
	FieldTypeText        FieldType = "text"
	FieldTypeTextarea    FieldType = "textarea"
	FieldTypeRichtext    FieldType = "richtext"
	FieldTypeNumber      FieldType = "number"
	FieldTypeDate        FieldType = "date"
	FieldTypeDatetime    FieldType = "datetime"
	FieldTypeRadio       FieldType = "radio"
	FieldTypeCheckbox    FieldType = "checkbox"
	FieldTypeSelect      FieldType = "select"
	FieldTypeFile        FieldType = "file"
	FieldTypeSignature   FieldType = "signature"
	FieldTypeRepeater    FieldType = "repeater"
	FieldTypeGroup       FieldType = "group"
	FieldTypeSwitch      FieldType = "switch"
	FieldTypeSlider      FieldType = "slider"
	FieldTypeCascade     FieldType = "cascade"
)

type OptionItem struct {
	Label string      `json:"label"`
	Value interface{} `json:"value"`
}

type CascadeOption struct {
	Label string         `json:"label"`
	Value string         `json:"value"`
	Children []CascadeOption `json:"children,omitempty"`
}

type ValidationRules struct {
	Required     bool        `json:"required,omitempty"`
	Readonly     bool        `json:"readonly,omitempty"`
	DefaultValue interface{} `json:"default_value,omitempty"`

	Min           *float64 `json:"min,omitempty"`
	Max           *float64 `json:"max,omitempty"`
	Integer       bool     `json:"integer,omitempty"`
	DecimalPlaces *int     `json:"decimal_places,omitempty"`

	MinLength *int `json:"min_length,omitempty"`
	MaxLength *int `json:"max_length,omitempty"`

	Phone   bool   `json:"phone,omitempty"`
	Email   bool   `json:"email,omitempty"`
	IDCard  bool   `json:"idcard,omitempty"`
	URL     bool   `json:"url,omitempty"`
	Pattern string `json:"pattern,omitempty"`

	Whitelist []interface{} `json:"whitelist,omitempty"`
	Blacklist []interface{} `json:"blacklist,omitempty"`
	Enum      []interface{} `json:"enum,omitempty"`

	NotBeforeToday bool      `json:"not_before_today,omitempty"`
	DateMin        *time.Time `json:"date_min,omitempty"`
	DateMax        *time.Time `json:"date_max,omitempty"`

	MaxSize           int64    `json:"max_size,omitempty"`
	AllowedExtensions []string `json:"allowed_extensions,omitempty"`
}

type FieldDef struct {
	Key         string            `json:"key"`
	Type        FieldType         `json:"type"`
	Label       string            `json:"label"`
	Placeholder string            `json:"placeholder,omitempty"`
	Rules       ValidationRules   `json:"rules,omitempty"`

	Options    []OptionItem     `json:"options,omitempty"`
	CascadeOptions []CascadeOption `json:"cascade_options,omitempty"`

	VisibleIf  interface{} `json:"visible_if,omitempty"`
	RequiredIf interface{} `json:"required_if,omitempty"`

	RepeaterFields []FieldDef `json:"repeater_fields,omitempty"`
	GroupFields    []FieldDef `json:"group_fields,omitempty"`

	SliderMin *float64 `json:"slider_min,omitempty"`
	SliderMax *float64 `json:"slider_max,omitempty"`
	SliderStep *float64 `json:"slider_step,omitempty"`
}

type FormSchema struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Version     int       `json:"version"`
	Fields      []FieldDef `json:"fields"`

	Workflow *WorkflowConfig `json:"workflow,omitempty"`

	SubmissionRateLimit *RateLimitConfig `json:"submission_rate_limit,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type RateLimitConfig struct {
	MaxPerMinute int `json:"max_per_minute"`
}

type WorkflowConfig struct {
	Enabled bool             `json:"enabled"`
	Nodes   []WorkflowNode   `json:"nodes"`
}

type WorkflowNodeType string

const (
	NodeTypeDirectManager   WorkflowNodeType = "direct_manager"
	NodeTypeDeptHead        WorkflowNodeType = "dept_head"
	NodeTypeHR              WorkflowNodeType = "hr"
	NodeTypeCustom          WorkflowNodeType = "custom"
)

type WorkflowNode struct {
	ID       string           `json:"id"`
	Type     WorkflowNodeType `json:"type"`
	Name     string           `json:"name"`
	Assignee string           `json:"assignee,omitempty"`
	Order    int              `json:"order"`
}

type SubmissionStatus string

const (
	SubmissionStatusDraft     SubmissionStatus = "draft"
	SubmissionStatusPending   SubmissionStatus = "pending"
	SubmissionStatusApproved  SubmissionStatus = "approved"
	SubmissionStatusRejected  SubmissionStatus = "rejected"
)

type Submission struct {
	ID            string           `json:"id"`
	FormID        string           `json:"form_id"`
	SchemaVersion int              `json:"schema_version"`
	SubmitterID   string           `json:"submitter_id"`
	SubmitterName string           `json:"submitter_name"`

	Data map[string]interface{} `json:"data"`

	Status    SubmissionStatus `json:"status"`
	CurrentNode string         `json:"current_node,omitempty"`

	ApprovalHistory []ApprovalRecord `json:"approval_history,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
)

type ApprovalRecord struct {
	NodeID    string         `json:"node_id"`
	NodeName  string         `json:"node_name"`
	Approver  string         `json:"approver"`
	Status    ApprovalStatus `json:"status"`
	Comment   string         `json:"comment,omitempty"`
	ApprovedAt *time.Time    `json:"approved_at,omitempty"`
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type ValidationResult struct {
	Valid   bool              `json:"valid"`
	Errors  []ValidationError `json:"errors,omitempty"`
}
