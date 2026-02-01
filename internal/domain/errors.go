package domain

import "fmt"

// DomainError represents a domain-specific error
type DomainError struct {
	Code    string
	Message string
	Err     error
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *DomainError) Unwrap() error {
	return e.Err
}

// NewDomainError creates a new DomainError
func NewDomainError(code, message string) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Err:     nil,
	}
}

// NewDomainErrorWithCause creates a new DomainError with an underlying cause
func NewDomainErrorWithCause(code, message string, err error) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Common domain error codes
const (
	ErrCodeValidation       = "VALIDATION_ERROR"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeAlreadyExists    = "ALREADY_EXISTS"
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeForbidden        = "FORBIDDEN"
	ErrCodeInternalError    = "INTERNAL_ERROR"
	ErrCodeInvalidOperation = "INVALID_OPERATION"
)

// Validation errors
var (
	ErrInvalidKnowledgeType      = NewDomainError(ErrCodeValidation, "invalid knowledge type")
	ErrInvalidKnowledgeStatus    = NewDomainError(ErrCodeValidation, "invalid knowledge status")
	ErrInvalidEmbeddingJobStatus = NewDomainError(ErrCodeValidation, "invalid embedding job status")
	ErrMissingRequiredField      = NewDomainError(ErrCodeValidation, "missing required field")
)

// Not found errors
var (
	ErrKnowledgeNotFound    = NewDomainError(ErrCodeNotFound, "knowledge item not found")
	ErrAssetNotFound        = NewDomainError(ErrCodeNotFound, "asset not found")
	ErrOrganizationNotFound = NewDomainError(ErrCodeNotFound, "organization not found")
	ErrProjectNotFound      = NewDomainError(ErrCodeNotFound, "project not found")
	ErrAPIKeyNotFound       = NewDomainError(ErrCodeNotFound, "api key not found")
)

// Already exists errors
var (
	ErrKnowledgeAlreadyExists    = NewDomainError(ErrCodeAlreadyExists, "knowledge item already exists")
	ErrAssetAlreadyExists        = NewDomainError(ErrCodeAlreadyExists, "asset already exists")
	ErrOrganizationAlreadyExists = NewDomainError(ErrCodeAlreadyExists, "organization already exists")
	ErrProjectAlreadyExists      = NewDomainError(ErrCodeAlreadyExists, "project already exists")
	ErrAPIKeyAlreadyExists       = NewDomainError(ErrCodeAlreadyExists, "api key already exists")
)

// Authorization errors
var (
	ErrAPIKeyRevoked = NewDomainError(ErrCodeUnauthorized, "api key has been revoked")
	ErrInvalidAPIKey = NewDomainError(ErrCodeUnauthorized, "invalid api key")
)

// Operation errors
var (
	ErrCannotModifyDeprecated = NewDomainError(ErrCodeInvalidOperation, "cannot modify deprecated knowledge")
	ErrCannotDeleteKnowledge  = NewDomainError(ErrCodeInvalidOperation, "cannot delete knowledge, use deprecation instead")
)

// Asset-specific errors
var (
	ErrSHA256Mismatch       = NewDomainError(ErrCodeValidation, "SHA256 hash does not match uploaded file")
	ErrAssetUploadNotFound  = NewDomainError(ErrCodeNotFound, "pending asset upload not found")
	ErrStorageOperationFail = NewDomainError(ErrCodeInternalError, "storage operation failed")
)
