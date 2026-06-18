package validation

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"dynamic-form-engine/internal/jsonlogic"
	"dynamic-form-engine/internal/models"
)

func ValidateSubmission(schema *models.FormSchema, data map[string]interface{}) models.ValidationResult {
	fillDefaults(schema.Fields, data)

	errors := make([]models.ValidationError, 0)

	for _, field := range schema.Fields {
		fieldErrors := validateField(field, data, "")
		errors = append(errors, fieldErrors...)
	}

	return models.ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

func fillDefaults(fields []models.FieldDef, data map[string]interface{}) {
	for _, field := range fields {
		if field.Rules.DefaultValue == nil {
			continue
		}

		val, exists := data[field.Key]
		if !exists || isNullOrBlank(val, field.Type) {
			data[field.Key] = field.Rules.DefaultValue
		}
	}
}

func isNullOrBlank(val interface{}, fieldType models.FieldType) bool {
	if val == nil {
		return true
	}
	switch fieldType {
	case models.FieldTypeText, models.FieldTypeTextarea, models.FieldTypeRichtext:
		if s, ok := val.(string); ok && strings.TrimSpace(s) == "" {
			return true
		}
	}
	return false
}

func validateField(field models.FieldDef, data map[string]interface{}, prefix string) []models.ValidationError {
	errors := make([]models.ValidationError, 0)

	fullKey := field.Key
	if prefix != "" {
		fullKey = prefix + "." + field.Key
	}

	visible := isFieldVisible(field, data)
	if !visible {
		if field.Type == models.FieldTypeGroup && len(field.GroupFields) > 0 {
			for _, sub := range field.GroupFields {
				subErrors := validateField(sub, data, fullKey)
				errors = append(errors, subErrors...)
			}
		}
		if field.Type == models.FieldTypeRepeater && len(field.RepeaterFields) > 0 {
			val, exists := data[field.Key]
			if exists {
				if arr, ok := val.([]interface{}); ok {
					for i, row := range arr {
						if rowMap, ok := row.(map[string]interface{}); ok {
							for _, sub := range field.RepeaterFields {
								subErrors := validateField(sub, rowMap, fmt.Sprintf("%s[%d]", fullKey, i))
								errors = append(errors, subErrors...)
							}
						}
					}
				}
			}
		}
		return errors
	}

	value := data[field.Key]

	required := isFieldRequired(field, data)
	if required && isEmpty(value) {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: field.Label + " 是必填项",
			Code:    "required",
		})
		return errors
	}

	if isEmpty(value) && !required {
		return errors
	}

	switch field.Type {
	case models.FieldTypeText, models.FieldTypeTextarea, models.FieldTypeRichtext:
		errors = append(errors, validateTextField(field, value, fullKey)...)
	case models.FieldTypeNumber:
		errors = append(errors, validateNumberField(field, value, fullKey)...)
	case models.FieldTypeDate, models.FieldTypeDatetime:
		errors = append(errors, validateDateField(field, value, fullKey)...)
	case models.FieldTypeRadio, models.FieldTypeSelect:
		errors = append(errors, validateSingleSelect(field, value, fullKey)...)
	case models.FieldTypeCheckbox:
		errors = append(errors, validateMultiSelect(field, value, fullKey)...)
	case models.FieldTypeFile:
		errors = append(errors, validateFileField(field, value, fullKey)...)
	case models.FieldTypeSignature:
		errors = append(errors, validateSignature(field, value, fullKey)...)
	case models.FieldTypeSwitch:
		errors = append(errors, validateSwitch(field, value, fullKey)...)
	case models.FieldTypeSlider:
		errors = append(errors, validateSlider(field, value, fullKey)...)
	case models.FieldTypeCascade:
		errors = append(errors, validateCascade(field, value, fullKey)...)
	case models.FieldTypeGroup:
		if len(field.GroupFields) > 0 {
			for _, sub := range field.GroupFields {
				subErrors := validateField(sub, data, fullKey)
				errors = append(errors, subErrors...)
			}
		}
	case models.FieldTypeRepeater:
		if len(field.RepeaterFields) > 0 {
			if arr, ok := value.([]interface{}); ok {
				for i, row := range arr {
					if rowMap, ok := row.(map[string]interface{}); ok {
						for _, sub := range field.RepeaterFields {
							subErrors := validateField(sub, rowMap, fmt.Sprintf("%s[%d]", fullKey, i))
							errors = append(errors, subErrors...)
						}
					}
				}
			}
		}
	}

	return errors
}

func isFieldVisible(field models.FieldDef, data map[string]interface{}) bool {
	if field.VisibleIf == nil {
		return true
	}
	return jsonlogic.Evaluate(field.VisibleIf, data)
}

func isFieldRequired(field models.FieldDef, data map[string]interface{}) bool {
	if field.Rules.Required {
		return true
	}
	if field.RequiredIf != nil {
		return jsonlogic.Evaluate(field.RequiredIf, data)
	}
	return false
}

func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	case bool:
		return false
	case int, int8, int16, int32, int64, float32, float64:
		return false
	default:
		return reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface())
	}
}

func validateTextField(field models.FieldDef, value interface{}, fullKey string) []models.ValidationError {
	errors := make([]models.ValidationError, 0)
	s := fmt.Sprintf("%v", value)

	if field.Rules.MinLength != nil && len(s) < *field.Rules.MinLength {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: fmt.Sprintf("%s 最少需要 %d 个字符", field.Label, *field.Rules.MinLength),
			Code:    "min_length",
		})
	}

	if field.Rules.MaxLength != nil && len(s) > *field.Rules.MaxLength {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: fmt.Sprintf("%s 最多 %d 个字符", field.Label, *field.Rules.MaxLength),
			Code:    "max_length",
		})
	}

	if field.Rules.Phone && !jsonlogic.IsPhone(s) {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: field.Label + " 格式不正确，请输入有效的手机号",
			Code:    "invalid_phone",
		})
	}

	if field.Rules.Email && !jsonlogic.IsEmail(s) {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: field.Label + " 格式不正确，请输入有效的邮箱",
			Code:    "invalid_email",
		})
	}

	if field.Rules.IDCard && !jsonlogic.IsIDCard(s) {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: field.Label + " 格式不正确，请输入有效的身份证号",
			Code:    "invalid_idcard",
		})
	}

	if field.Rules.URL && !jsonlogic.IsURL(s) {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: field.Label + " 格式不正确，请输入有效的 URL",
			Code:    "invalid_url",
		})
	}

	if field.Rules.Pattern != "" && !jsonlogic.MatchesPattern(s, field.Rules.Pattern) {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: field.Label + " 格式不符合要求",
			Code:    "invalid_pattern",
		})
	}

	errors = append(errors, validateCollectionRules(field, value, fullKey)...)

	return errors
}

func validateNumberField(field models.FieldDef, value interface{}, fullKey string) []models.ValidationError {
	errors := make([]models.ValidationError, 0)

	f, err := toFloat(value)
	if err != nil {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: field.Label + " 必须是数字",
			Code:    "invalid_number",
		})
		return errors
	}

	if field.Rules.Min != nil && f < *field.Rules.Min {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: fmt.Sprintf("%s 不能小于 %v", field.Label, *field.Rules.Min),
			Code:    "min_value",
		})
	}

	if field.Rules.Max != nil && f > *field.Rules.Max {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: fmt.Sprintf("%s 不能大于 %v", field.Label, *field.Rules.Max),
			Code:    "max_value",
		})
	}

	if field.Rules.Integer {
		if f != float64(int64(f)) {
			errors = append(errors, models.ValidationError{
				Field:   fullKey,
				Message: field.Label + " 必须是整数",
				Code:    "integer_required",
			})
		}
	}

	if field.Rules.DecimalPlaces != nil {
		str := fmt.Sprintf("%v", value)
		dotIdx := strings.Index(str, ".")
		if dotIdx >= 0 {
			decimals := len(str) - dotIdx - 1
			if decimals > *field.Rules.DecimalPlaces {
				errors = append(errors, models.ValidationError{
					Field:   fullKey,
					Message: fmt.Sprintf("%s 最多保留 %d 位小数", field.Label, *field.Rules.DecimalPlaces),
					Code:    "decimal_places",
				})
			}
		}
	}

	errors = append(errors, validateCollectionRules(field, value, fullKey)...)

	return errors
}

func validateDateField(field models.FieldDef, value interface{}, fullKey string) []models.ValidationError {
	errors := make([]models.ValidationError, 0)

	s := fmt.Sprintf("%v", value)
	layout := "2006-01-02"
	if field.Type == models.FieldTypeDatetime {
		layout = time.RFC3339
	}

	t, err := time.Parse(layout, s)
	if err != nil {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: field.Label + " 日期格式不正确",
			Code:    "invalid_date",
		})
		return errors
	}

	if field.Rules.NotBeforeToday {
		today := time.Now().Truncate(24 * time.Hour)
		dateOnly := t.Truncate(24 * time.Hour)
		if dateOnly.Before(today) {
			errors = append(errors, models.ValidationError{
				Field:   fullKey,
				Message: field.Label + " 不能早于今天",
				Code:    "date_before_today",
			})
		}
	}

	if field.Rules.DateMin != nil && t.Before(*field.Rules.DateMin) {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: fmt.Sprintf("%s 不能早于 %v", field.Label, field.Rules.DateMin.Format(layout)),
			Code:    "date_min",
		})
	}

	if field.Rules.DateMax != nil && t.After(*field.Rules.DateMax) {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: fmt.Sprintf("%s 不能晚于 %v", field.Label, field.Rules.DateMax.Format(layout)),
			Code:    "date_max",
		})
	}

	return errors
}

func validateSingleSelect(field models.FieldDef, value interface{}, fullKey string) []models.ValidationError {
	errors := make([]models.ValidationError, 0)

	if len(field.Options) > 0 {
		found := false
		for _, opt := range field.Options {
			if fmt.Sprintf("%v", opt.Value) == fmt.Sprintf("%v", value) {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, models.ValidationError{
				Field:   fullKey,
				Message: field.Label + " 选项不合法",
				Code:    "invalid_option",
			})
		}
	}

	errors = append(errors, validateCollectionRules(field, value, fullKey)...)

	return errors
}

func validateMultiSelect(field models.FieldDef, value interface{}, fullKey string) []models.ValidationError {
	errors := make([]models.ValidationError, 0)

	arr, ok := value.([]interface{})
	if !ok {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: field.Label + " 数据格式不正确",
			Code:    "invalid_format",
		})
		return errors
	}

	if len(field.Options) > 0 {
		for _, v := range arr {
			found := false
			for _, opt := range field.Options {
				if fmt.Sprintf("%v", opt.Value) == fmt.Sprintf("%v", v) {
					found = true
					break
				}
			}
			if !found {
				errors = append(errors, models.ValidationError{
					Field:   fullKey,
					Message: field.Label + " 包含非法选项",
					Code:    "invalid_option",
				})
				break
			}
		}
	}

	errors = append(errors, validateCollectionRules(field, value, fullKey)...)

	return errors
}

func validateFileField(field models.FieldDef, value interface{}, fullKey string) []models.ValidationError {
	errors := make([]models.ValidationError, 0)

	files, ok := value.([]interface{})
	if !ok {
		if s, ok := value.(string); ok && s != "" {
			files = []interface{}{s}
		}
	}

	for _, f := range files {
		filePath := fmt.Sprintf("%v", f)

		if field.Rules.MaxSize > 0 {
			_ = filePath
		}

		if len(field.Rules.AllowedExtensions) > 0 {
			ext := getFileExt(filePath)
			allowed := false
			for _, e := range field.Rules.AllowedExtensions {
				if strings.EqualFold(ext, e) {
					allowed = true
					break
				}
			}
			if !allowed {
				errors = append(errors, models.ValidationError{
					Field:   fullKey,
					Message: field.Label + " 文件类型不支持",
					Code:    "invalid_file_type",
				})
				break
			}
		}
	}

	return errors
}

func validateSignature(field models.FieldDef, value interface{}, fullKey string) []models.ValidationError {
	errors := make([]models.ValidationError, 0)
	if s, ok := value.(string); ok && s == "" {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: field.Label + " 请签名",
			Code:    "signature_required",
		})
	}
	return errors
}

func validateSwitch(field models.FieldDef, value interface{}, fullKey string) []models.ValidationError {
	errors := make([]models.ValidationError, 0)
	if _, ok := value.(bool); !ok {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: field.Label + " 数据格式不正确",
			Code:    "invalid_format",
		})
	}
	return errors
}

func validateSlider(field models.FieldDef, value interface{}, fullKey string) []models.ValidationError {
	errors := make([]models.ValidationError, 0)

	f, err := toFloat(value)
	if err != nil {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: field.Label + " 必须是数字",
			Code:    "invalid_number",
		})
		return errors
	}

	if field.SliderMin != nil && f < *field.SliderMin {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: fmt.Sprintf("%s 不能小于 %v", field.Label, *field.SliderMin),
			Code:    "min_value",
		})
	}

	if field.SliderMax != nil && f > *field.SliderMax {
		errors = append(errors, models.ValidationError{
			Field:   fullKey,
			Message: fmt.Sprintf("%s 不能大于 %v", field.Label, *field.SliderMax),
			Code:    "max_value",
		})
	}

	return errors
}

func validateCascade(field models.FieldDef, value interface{}, fullKey string) []models.ValidationError {
	errors := make([]models.ValidationError, 0)

	arr, ok := value.([]interface{})
	if !ok {
		if s, ok := value.(string); ok && s != "" {
			arr = []interface{}{s}
		}
	}

	if len(arr) > 0 && len(field.CascadeOptions) > 0 {
		valid := validateCascadeValue(field.CascadeOptions, arr, 0)
		if !valid {
			errors = append(errors, models.ValidationError{
				Field:   fullKey,
				Message: field.Label + " 选项不合法",
				Code:    "invalid_option",
			})
		}
	}

	return errors
}

func validateCascadeValue(options []models.CascadeOption, values []interface{}, level int) bool {
	if level >= len(values) {
		return true
	}

	current := fmt.Sprintf("%v", values[level])
	for _, opt := range options {
		if opt.Value == current {
			if level == len(values)-1 {
				return true
			}
			if len(opt.Children) > 0 {
				return validateCascadeValue(opt.Children, values, level+1)
			}
			return false
		}
	}
	return false
}

func validateCollectionRules(field models.FieldDef, value interface{}, fullKey string) []models.ValidationError {
	errors := make([]models.ValidationError, 0)

	if len(field.Rules.Enum) > 0 {
		found := false
		for _, e := range field.Rules.Enum {
			if fmt.Sprintf("%v", e) == fmt.Sprintf("%v", value) {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, models.ValidationError{
				Field:   fullKey,
				Message: field.Label + " 值不在允许范围内",
				Code:    "enum_violation",
			})
		}
	}

	if len(field.Rules.Whitelist) > 0 {
		found := false
		for _, w := range field.Rules.Whitelist {
			if fmt.Sprintf("%v", w) == fmt.Sprintf("%v", value) {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, models.ValidationError{
				Field:   fullKey,
				Message: field.Label + " 值不在白名单中",
				Code:    "whitelist_violation",
			})
		}
	}

	if len(field.Rules.Blacklist) > 0 {
		for _, b := range field.Rules.Blacklist {
			if fmt.Sprintf("%v", b) == fmt.Sprintf("%v", value) {
				errors = append(errors, models.ValidationError{
					Field:   fullKey,
					Message: field.Label + " 值在黑名单中",
					Code:    "blacklist_violation",
				})
				break
			}
		}
	}

	return errors
}

func toFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert to float")
	}
}

func getFileExt(path string) string {
	idx := strings.LastIndex(path, ".")
	if idx < 0 {
		return ""
	}
	return path[idx:]
}
