package jsonlogic

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func Evaluate(expr interface{}, data map[string]interface{}) bool {
	if expr == nil {
		return true
	}

	switch v := expr.(type) {
	case bool:
		return v
	case map[string]interface{}:
		return evaluateMap(v, data)
	default:
		return true
	}
}

func evaluateMap(expr map[string]interface{}, data map[string]interface{}) bool {
	for op, val := range expr {
		switch op {
		case "var":
			_, _ = getVar(val, data)
			return true
		case "==":
			arr, ok := val.([]interface{})
			if !ok || len(arr) < 2 {
				return false
			}
			left := resolveValue(arr[0], data)
			right := resolveValue(arr[1], data)
			return reflect.DeepEqual(left, right)
		case "!=":
			arr, ok := val.([]interface{})
			if !ok || len(arr) < 2 {
				return false
			}
			left := resolveValue(arr[0], data)
			right := resolveValue(arr[1], data)
			return !reflect.DeepEqual(left, right)
		case ">":
			arr, ok := val.([]interface{})
			if !ok || len(arr) < 2 {
				return false
			}
			left := toFloat(resolveValue(arr[0], data))
			right := toFloat(resolveValue(arr[1], data))
			return left > right
		case ">=":
			arr, ok := val.([]interface{})
			if !ok || len(arr) < 2 {
				return false
			}
			left := toFloat(resolveValue(arr[0], data))
			right := toFloat(resolveValue(arr[1], data))
			return left >= right
		case "<":
			arr, ok := val.([]interface{})
			if !ok || len(arr) < 2 {
				return false
			}
			left := toFloat(resolveValue(arr[0], data))
			right := toFloat(resolveValue(arr[1], data))
			return left < right
		case "<=":
			arr, ok := val.([]interface{})
			if !ok || len(arr) < 2 {
				return false
			}
			left := toFloat(resolveValue(arr[0], data))
			right := toFloat(resolveValue(arr[1], data))
			return left <= right
		case "and":
			arr, ok := val.([]interface{})
			if !ok {
				return false
			}
			for _, item := range arr {
				if !evaluateItem(item, data) {
					return false
				}
			}
			return true
		case "or":
			arr, ok := val.([]interface{})
			if !ok {
				return false
			}
			for _, item := range arr {
				if evaluateItem(item, data) {
					return true
				}
			}
			return false
		case "!":
			return !evaluateItem(val, data)
		case "in":
			arr, ok := val.([]interface{})
			if !ok || len(arr) < 2 {
				return false
			}
			needle := resolveValue(arr[0], data)
			haystack := resolveValue(arr[1], data)
			return checkIn(needle, haystack)
		case "missing":
			arr, ok := val.([]interface{})
			if !ok {
				if str, ok := val.(string); ok {
					arr = []interface{}{str}
				} else {
					return false
				}
			}
			for _, key := range arr {
				keyStr := fmt.Sprintf("%v", key)
				if v, ok := data[keyStr]; !ok || v == nil || v == "" {
					return true
				}
			}
			return false
		case "cat":
			return true
		default:
			return true
		}
	}
	return true
}

func evaluateItem(item interface{}, data map[string]interface{}) bool {
	switch v := item.(type) {
	case bool:
		return v
	case map[string]interface{}:
		return evaluateMap(v, data)
	default:
		return toBool(v)
	}
}

func resolveValue(val interface{}, data map[string]interface{}) interface{} {
	switch v := val.(type) {
	case map[string]interface{}:
		if varVal, ok := v["var"]; ok {
			result, _ := getVar(varVal, data)
			return result
		}
		return v
	default:
		return v
	}
}

func getVar(varVal interface{}, data map[string]interface{}) (interface{}, bool) {
	switch v := varVal.(type) {
	case string:
		val, ok := data[v]
		return val, ok
	case int:
		return nil, false
	default:
		return nil, false
	}
}

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0
		}
		return f
	default:
		return 0
	}
}

func toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != "" && val != "0" && val != "false"
	case int:
		return val != 0
	case float64:
		return val != 0
	case nil:
		return false
	default:
		return true
	}
}

func checkIn(needle interface{}, haystack interface{}) bool {
	switch h := haystack.(type) {
	case []interface{}:
		for _, item := range h {
			if reflect.DeepEqual(needle, item) {
				return true
			}
		}
		return false
	case string:
		nStr := fmt.Sprintf("%v", needle)
		return strings.Contains(h, nStr)
	default:
		return false
	}
}

func ValidateExpression(expr interface{}) error {
	if expr == nil {
		return nil
	}

	switch v := expr.(type) {
	case bool, string, float64, int:
		return nil
	case map[string]interface{}:
		for op, val := range v {
			switch op {
			case "var", "==", "!=", ">", ">=", "<", "<=", "and", "or", "!", "in", "missing", "cat":
				return validateValue(val)
			default:
				return fmt.Errorf("unknown operator: %s", op)
			}
		}
		return nil
	case []interface{}:
		for _, item := range v {
			if err := ValidateExpression(item); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported expression type: %T", v)
	}
}

func validateValue(val interface{}) error {
	switch v := val.(type) {
	case []interface{}:
		for _, item := range v {
			if err := ValidateExpression(item); err != nil {
				return err
			}
		}
		return nil
	default:
		return ValidateExpression(v)
	}
}

var (
	phoneRegex   = regexp.MustCompile(`^1[3-9]\d{9}$`)
	emailRegex   = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	idCardRegex  = regexp.MustCompile(`^[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]$`)
	urlRegex     = regexp.MustCompile(`^https?://[^\s]+$`)
)

func IsPhone(s string) bool {
	return phoneRegex.MatchString(s)
}

func IsEmail(s string) bool {
	return emailRegex.MatchString(s)
}

func IsIDCard(s string) bool {
	return idCardRegex.MatchString(s)
}

func IsURL(s string) bool {
	return urlRegex.MatchString(s)
}

func MatchesPattern(s, pattern string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(s)
}

func ValidateRegex(pattern string) error {
	_, err := regexp.Compile(pattern)
	return err
}
