package validation

import (
	"github.com/proyaai/instantgate/internal/config"
	"github.com/proyaai/instantgate/internal/database/mysql"
)

type ValidationManager struct {
	config          *config.ValidationConfig
	schemaCache     *mysql.SchemaCache
	schemaValidator *SchemaValidator
	ruleValidator   *RuleValidator
}

func NewValidationManager(cfg *config.ValidationConfig, schemaCache *mysql.SchemaCache) *ValidationManager {
	return &ValidationManager{
		config:          cfg,
		schemaCache:     schemaCache,
		schemaValidator: NewSchemaValidator(schemaCache, cfg.StrictMode),
		ruleValidator:   NewRuleValidator(cfg.Rules),
	}
}

// Validate performs schema-based and rule-based validation
func (vm *ValidationManager) Validate(tableName string, data map[string]interface{}, operation Operation) error {
	// Validation kapalıysa skip et
	if !vm.config.Enabled {
		return nil
	}

	var allErrors ValidationErrors

	// 1. Schema-based validation (otomatik: tip, uzunluk, null kontrolü)
	if schemaErrs := vm.schemaValidator.Validate(tableName, data, operation); schemaErrs.HasErrors() {
		allErrors = append(allErrors, schemaErrs...)
	}

	// 2. Rule-based validation (config.yaml kuralları: regex, min/max, enum)
	if ruleErrs := vm.ruleValidator.Validate(tableName, data); ruleErrs.HasErrors() {
		allErrors = append(allErrors, ruleErrs...)
	}

	if allErrors.HasErrors() {
		// İlk hatayı döndür (ileride tüm hataları döndürmeye genişletilebilir)
		return allErrors[0]
	}

	return nil
}

// ValidateMultiple returns all validation errors instead of just the first one
func (vm *ValidationManager) ValidateMultiple(tableName string, data map[string]interface{}, operation Operation) ValidationErrors {
	if !vm.config.Enabled {
		return nil
	}

	var allErrors ValidationErrors

	if schemaErrs := vm.schemaValidator.Validate(tableName, data, operation); schemaErrs.HasErrors() {
		allErrors = append(allErrors, schemaErrs...)
	}

	if ruleErrs := vm.ruleValidator.Validate(tableName, data); ruleErrs.HasErrors() {
		allErrors = append(allErrors, ruleErrs...)
	}

	return allErrors
}
