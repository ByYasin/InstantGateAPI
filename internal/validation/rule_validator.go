package validation

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/proyaai/instantgate/internal/config"
)

type RuleValidator struct {
	rules        map[string]map[string][]config.RuleItem // table -> field -> rules
	regexCache   map[string]*regexp.Regexp
	regexCacheMu sync.RWMutex
}

func NewRuleValidator(rules map[string]map[string][]config.RuleItem) *RuleValidator {
	normalizedRules := make(map[string]map[string][]config.RuleItem)
	for table, fields := range rules {
		normalizedRules[strings.ToLower(table)] = fields
	}

	return &RuleValidator{
		rules:      normalizedRules,
		regexCache: make(map[string]*regexp.Regexp),
	}
}

func (rv *RuleValidator) Validate(tableName string, data map[string]interface{}) ValidationErrors {
	var errs ValidationErrors

	tableRules, ok := rv.rules[strings.ToLower(tableName)]
	if !ok {
		return nil
	}

	for field, rules := range tableRules {
		value, exists := data[field]

		for _, rule := range rules {
			if err := rv.validateRule(field, value, exists, rule); err != nil {
				errs = append(errs, err)
			}
		}
	}

	return errs
}

func (rv *RuleValidator) validateRule(field string, value interface{}, exists bool, rule config.RuleItem) *ValidationError {
	switch strings.ToLower(rule.Type) {
	case "required":
		return rv.validateRequired(field, value, exists, rule)
	case "regex":
		return rv.validateRegex(field, value, exists, rule)
	case "min":
		return rv.validateMin(field, value, exists, rule)
	case "max":
		return rv.validateMax(field, value, exists, rule)
	case "enum":
		return rv.validateEnum(field, value, exists, rule)
	case "length":
		return rv.validateLength(field, value, exists, rule)
	default:
		return nil
	}
}

func (rv *RuleValidator) validateRequired(field string, value interface{}, exists bool, rule config.RuleItem) *ValidationError {
	if !exists || value == nil {
		return NewValidationError(field, rv.getMessage(rule, fmt.Sprintf("'%s' is required", field)))
	}

	if str, ok := value.(string); ok && strings.TrimSpace(str) == "" {
		return NewValidationError(field, rv.getMessage(rule, fmt.Sprintf("'%s' cannot be empty", field)))
	}

	return nil
}

func (rv *RuleValidator) validateRegex(field string, value interface{}, exists bool, rule config.RuleItem) *ValidationError {
	if !exists || value == nil {
		return nil 
	}

	str, ok := value.(string)
	if !ok {
		return nil
	}

	pattern := rv.getCompiledRegex(rule.Pattern)
	if pattern == nil {
		return nil 
	}

	if !pattern.MatchString(str) {
		return NewValidationError(field, rv.getMessage(rule, fmt.Sprintf("'%s' does not match the required pattern", field)))
	}

	return nil
}

func (rv *RuleValidator) validateMin(field string, value interface{}, exists bool, rule config.RuleItem) *ValidationError {
	if !exists || value == nil {
		return nil
	}

	numValue, ok := toFloat64(value)
	if !ok {
		return nil
	}

	minValue, ok := toFloat64(rule.Value)
	if !ok {
		return nil
	}

	if numValue < minValue {
		return NewValidationError(field, rv.getMessage(rule, fmt.Sprintf("'%s' must be at least %v", field, rule.Value)))
	}

	return nil
}

func (rv *RuleValidator) validateMax(field string, value interface{}, exists bool, rule config.RuleItem) *ValidationError {
	if !exists || value == nil {
		return nil
	}

	numValue, ok := toFloat64(value)
	if !ok {
		return nil
	}

	maxValue, ok := toFloat64(rule.Value)
	if !ok {
		return nil
	}

	if numValue > maxValue {
		return NewValidationError(field, rv.getMessage(rule, fmt.Sprintf("'%s' must be at most %v", field, rule.Value)))
	}

	return nil
}

func (rv *RuleValidator) validateEnum(field string, value interface{}, exists bool, rule config.RuleItem) *ValidationError {
	if !exists || value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return nil
	}

	for _, allowed := range rule.Values {
		if strings.EqualFold(str, allowed) {
			return nil
		}
	}

	return NewValidationError(field, rv.getMessage(rule, fmt.Sprintf("'%s' must be one of: %s", field, strings.Join(rule.Values, ", "))))
}

func (rv *RuleValidator) validateLength(field string, value interface{}, exists bool, rule config.RuleItem) *ValidationError {
	if !exists || value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return nil
	}

	maxLen, ok := toFloat64(rule.Value)
	if !ok {
		return nil
	}

	if len(str) > int(maxLen) {
		return NewValidationError(field, rv.getMessage(rule, fmt.Sprintf("'%s' must be at most %d characters", field, int(maxLen))))
	}

	return nil
}

func (rv *RuleValidator) getMessage(rule config.RuleItem, defaultMsg string) string {
	if rule.Message != "" {
		return rule.Message
	}
	return defaultMsg
}

func (rv *RuleValidator) getCompiledRegex(pattern string) *regexp.Regexp {
	rv.regexCacheMu.RLock()
	compiled, ok := rv.regexCache[pattern]
	rv.regexCacheMu.RUnlock()

	if ok {
		return compiled
	}

	rv.regexCacheMu.Lock()
	defer rv.regexCacheMu.Unlock()

	if compiled, ok := rv.regexCache[pattern]; ok {
		return compiled
	}

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		rv.regexCache[pattern] = nil
		return nil
	}

	rv.regexCache[pattern] = compiled
	return compiled
}

func toFloat64(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	default:
		return 0, false
	}
}
