package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

func SendError(w http.ResponseWriter, r *http.Request, status int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	resp := ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
		Code:    status,
	}

	if err != nil {
		log.Printf("[ERROR] %s %s - Status: %d - Error: %v - Message: %s",
			r.Method, r.URL.Path, status, err, message)

		if status >= 500 {
			resp.Message = "An internal error occurred"
		}
	}

	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
		log.Printf("[ERROR] Failed to encode error response: %v", encErr)
	}
}

func SendJSON(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			log.Printf("[ERROR] Failed to encode JSON response: %v", err)
		}
	}
}

type ValidationErrorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Code    int               `json:"code"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func SendValidationError(w http.ResponseWriter, r *http.Request, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity) // 422

	resp := ValidationErrorResponse{
		Error:   "Validation Failed",
		Message: "The submitted data is invalid",
		Code:    http.StatusUnprocessableEntity,
		Fields:  fields,
	}

	log.Printf("[VALIDATION] %s %s - Fields: %v", r.Method, r.URL.Path, fields)

	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
		log.Printf("[ERROR] Failed to encode validation error response: %v", encErr)
	}
}

var (
	ErrTableNotFound    = "Table not found"
	ErrRecordNotFound   = "Record not found"
	ErrInvalidRequest   = "Invalid request"
	ErrInvalidFilter    = "Invalid filter"
	ErrUnauthorized     = "Unauthorized"
	ErrForbidden        = "Forbidden"
	ErrInvalidInput     = "Invalid input"
	ErrDatabaseError    = "Database error"
	ErrConflict         = "Conflict"
	ErrValidationFailed = "Validation failed"
)
