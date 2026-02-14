package validation

import (
	"fmt"
	"strings"

	"github.com/proyaai/instantgate/internal/database/mysql"
	"github.com/proyaai/instantgate/internal/query"
)

type SchemaValidator struct {
	schemaCache *mysql.SchemaCache
	strictMode  bool
}

func NewSchemaValidator(schemaCache *mysql.SchemaCache, strictMode bool) *SchemaValidator {
	return &SchemaValidator{
		schemaCache: schemaCache,
		strictMode:  strictMode,
	}
}

func (sv *SchemaValidator) Validate(tableName string, data map[string]interface{}, op Operation) ValidationErrors {
	var errs ValidationErrors

	tableSchema, exists := sv.schemaCache.Get(tableName)
	if !exists {
		return ValidationErrors{NewValidationError("_table", fmt.Sprintf("Table '%s' not found", tableName))}
	}

	for field := range data {
		if _, ok := tableSchema.Columns[strings.ToLower(field)]; !ok {
			if sv.strictMode {
				errs = append(errs, NewValidationError(field, fmt.Sprintf("Unknown column '%s' in table '%s'", field, tableName)))
			}
		}
	}

	if errs.HasErrors() {
		return errs
	}

	for field, value := range data {
		col, ok := tableSchema.Columns[strings.ToLower(field)]
		if !ok {
			continue 
		}

		if op == OperationCreate && col.IsAutoIncrement {
			continue
		}

		if op == OperationUpdate && col.IsPrimaryKey {
			continue
		}

		if err := query.ValidateColumn(col, value); err != nil {
			errs = append(errs, NewValidationError(field, err.Error()))
		}
	}

	if op == OperationCreate {
		for colName, col := range tableSchema.Columns {
			if col.IsAutoIncrement || col.Nullable {
				continue
			}

			found := false
			for field := range data {
				if strings.EqualFold(field, colName) {
					found = true
					break
				}
			}

			if !found {
				errs = append(errs, NewValidationError(col.Name, fmt.Sprintf("Column '%s' is required and does not allow NULL", col.Name)))
			}
		}
	}

	return errs
}