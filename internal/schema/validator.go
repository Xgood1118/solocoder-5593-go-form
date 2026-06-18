package schema

import (
	"errors"
	"fmt"

	"dynamic-form-engine/internal/jsonlogic"
	"dynamic-form-engine/internal/models"
)

func ValidateSchema(schema *models.FormSchema) error {
	if schema.Name == "" {
		return errors.New("表单名称不能为空")
	}

	if len(schema.Fields) == 0 {
		return errors.New("表单至少需要一个字段")
	}

	keys := make(map[string]bool)
	if err := validateFields(schema.Fields, keys, ""); err != nil {
		return err
	}

	if schema.Workflow != nil && schema.Workflow.Enabled {
		if err := validateWorkflow(schema.Workflow); err != nil {
			return err
		}
	}

	if schema.SubmissionRateLimit != nil {
		if schema.SubmissionRateLimit.MaxPerMinute < 0 {
			return errors.New("提交频率限制不能为负数")
		}
	}

	return nil
}

func validateFields(fields []models.FieldDef, keys map[string]bool, prefix string) error {
	for _, field := range fields {
		if field.Key == "" {
			return errors.New("字段 key 不能为空")
		}

		fullKey := field.Key
		if prefix != "" {
			fullKey = prefix + "." + field.Key
		}

		if keys[fullKey] {
			return fmt.Errorf("字段 key 重复: %s", fullKey)
		}
		keys[fullKey] = true

		if !isValidFieldType(field.Type) {
			return fmt.Errorf("不支持的字段类型: %s (字段: %s)", field.Type, field.Key)
		}

		if field.Label == "" {
			return fmt.Errorf("字段标签不能为空: %s", field.Key)
		}

		if field.VisibleIf != nil {
			if err := jsonlogic.ValidateExpression(field.VisibleIf); err != nil {
				return fmt.Errorf("字段 %s 的 visible_if 表达式不合法: %v", field.Key, err)
			}
		}

		if field.RequiredIf != nil {
			if err := jsonlogic.ValidateExpression(field.RequiredIf); err != nil {
				return fmt.Errorf("字段 %s 的 required_if 表达式不合法: %v", field.Key, err)
			}
		}

		if field.Rules.Pattern != "" {
			if err := jsonlogic.ValidateRegex(field.Rules.Pattern); err != nil {
				return fmt.Errorf("字段 %s 的正则表达式不合法: %v", field.Key, err)
			}
		}

		if field.Type == models.FieldTypeRepeater {
			if len(field.RepeaterFields) == 0 {
				return fmt.Errorf("repeater 字段 %s 至少需要一个子字段", field.Key)
			}
			subKeys := make(map[string]bool)
			if err := validateFields(field.RepeaterFields, subKeys, fullKey); err != nil {
				return err
			}
		}

		if field.Type == models.FieldTypeGroup {
			if len(field.GroupFields) == 0 {
				return fmt.Errorf("group 字段 %s 至少需要一个子字段", field.Key)
			}
			subKeys := make(map[string]bool)
			if err := validateFields(field.GroupFields, subKeys, fullKey); err != nil {
				return err
			}
		}

		if (field.Type == models.FieldTypeRadio ||
			field.Type == models.FieldTypeCheckbox ||
			field.Type == models.FieldTypeSelect) && len(field.Options) == 0 {
			return fmt.Errorf("字段 %s 是选择类型但没有配置选项", field.Key)
		}

		if field.Type == models.FieldTypeCascade && len(field.CascadeOptions) == 0 {
			return fmt.Errorf("级联字段 %s 没有配置级联选项", field.Key)
		}

		if field.Type == models.FieldTypeSlider {
			if field.SliderMin == nil || field.SliderMax == nil {
				return fmt.Errorf("滑块字段 %s 必须配置最小值和最大值", field.Key)
			}
			if *field.SliderMin >= *field.SliderMax {
				return fmt.Errorf("滑块字段 %s 最小值必须小于最大值", field.Key)
			}
		}
	}

	return nil
}

func isValidFieldType(t models.FieldType) bool {
	switch t {
	case models.FieldTypeText,
		models.FieldTypeTextarea,
		models.FieldTypeRichtext,
		models.FieldTypeNumber,
		models.FieldTypeDate,
		models.FieldTypeDatetime,
		models.FieldTypeRadio,
		models.FieldTypeCheckbox,
		models.FieldTypeSelect,
		models.FieldTypeFile,
		models.FieldTypeSignature,
		models.FieldTypeRepeater,
		models.FieldTypeGroup,
		models.FieldTypeSwitch,
		models.FieldTypeSlider,
		models.FieldTypeCascade:
		return true
	default:
		return false
	}
}

func validateWorkflow(wf *models.WorkflowConfig) error {
	if len(wf.Nodes) == 0 {
		return errors.New("工作流至少需要一个审批节点")
	}

	nodeIDs := make(map[string]bool)
	for _, node := range wf.Nodes {
		if node.ID == "" {
			return errors.New("审批节点 ID 不能为空")
		}
		if nodeIDs[node.ID] {
			return fmt.Errorf("审批节点 ID 重复: %s", node.ID)
		}
		nodeIDs[node.ID] = true

		if node.Name == "" {
			return fmt.Errorf("审批节点名称不能为空: %s", node.ID)
		}

		if !isValidNodeType(node.Type) {
			return fmt.Errorf("不支持的节点类型: %s", node.Type)
		}
	}

	return nil
}

func isValidNodeType(t models.WorkflowNodeType) bool {
	switch t {
	case models.NodeTypeDirectManager,
		models.NodeTypeDeptHead,
		models.NodeTypeHR,
		models.NodeTypeCustom:
		return true
	default:
		return false
	}
}

func GetAllFieldKeys(schema *models.FormSchema) []string {
	var keys []string
	collectFieldKeys(schema.Fields, "", &keys)
	return keys
}

func collectFieldKeys(fields []models.FieldDef, prefix string, result *[]string) {
	for _, field := range fields {
		fullKey := field.Key
		if prefix != "" {
			fullKey = prefix + "." + field.Key
		}
		*result = append(*result, fullKey)

		if field.Type == models.FieldTypeRepeater && len(field.RepeaterFields) > 0 {
			collectFieldKeys(field.RepeaterFields, fullKey, result)
		}
		if field.Type == models.FieldTypeGroup && len(field.GroupFields) > 0 {
			collectFieldKeys(field.GroupFields, fullKey, result)
		}
	}
}
