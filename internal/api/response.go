package api

import (
	"encoding/json"
	"net/http"

	"github.com/cloo-solutions/neotexai/internal/domain"
)

// SuccessResponse wraps successful API responses
type SuccessResponse struct {
	Data interface{} `json:"data"`
}

// ErrorResponse represents an error API response
type ErrorResponse struct {
	Error string `json:"error"`
}

// JSON writes a JSON response with the given status code
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// Success writes a successful JSON response
func Success(w http.ResponseWriter, status int, data interface{}) {
	JSON(w, status, SuccessResponse{Data: data})
}

// Error writes an error JSON response
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, ErrorResponse{Error: message})
}

// DomainErrorToHTTP maps domain errors to HTTP status codes
func DomainErrorToHTTP(err error) int {
	if err == nil {
		return http.StatusOK
	}

	domainErr, ok := err.(*domain.DomainError)
	if !ok {
		return http.StatusInternalServerError
	}

	switch domainErr.Code {
	case domain.ErrCodeValidation:
		return http.StatusBadRequest
	case domain.ErrCodeNotFound:
		return http.StatusNotFound
	case domain.ErrCodeAlreadyExists:
		return http.StatusConflict
	case domain.ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case domain.ErrCodeForbidden:
		return http.StatusForbidden
	case domain.ErrCodeInvalidOperation:
		return http.StatusBadRequest
	case domain.ErrCodeInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// HandleError writes an appropriate error response based on the error type
func HandleError(w http.ResponseWriter, err error) {
	status := DomainErrorToHTTP(err)
	Error(w, status, err.Error())
}
