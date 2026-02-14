package validation

import (
	"fmt"
)

type ValidationError struct {
	Field   string
	Message string
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ve.Field, ve.Message)
}

type ValidationErrors []*ValidationError

func (ve ValidationErrors) Error() string {
	if len(ve) == 0 {
		return "no validation errors"
	}
	if len(ve) == 1 {
		return ve[0].Error()
	}

	msg := fmt.Sprintf("validation failed (%d errors): ", len(ve))
	for i, err := range ve {
		if i > 0 {
			msg += "; "
		}
		msg += err.Error()
	}
	return msg
}

func (ve ValidationErrors) Add(field, message string) ValidationErrors {
	return append(ve, &ValidationError{Field: field, Message: message})
}

func (ve ValidationErrors) HasErrors() bool {
	return len(ve) > 0
}

func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

var (
	ErrTableNotFound     = fmt.Errorf("table not found")
	ErrColumnNotFound    = fmt.Errorf("column not found")
	ErrInvalidValue      = fmt.Errorf("invalid value")
	ErrRequiredFieldMiss = fmt.Errorf("required field is missing")
	ErrPatternMismatch   = fmt.Errorf("pattern mismatch")
	ErrValueOutOfRange   = fmt.Errorf("value out of range")
)
