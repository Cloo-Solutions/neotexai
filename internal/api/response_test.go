package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloo-solutions/neotexai/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSON(t *testing.T) {
	w := httptest.NewRecorder()

	JSON(w, http.StatusOK, map[string]string{"key": "value"})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var result map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "value", result["key"])
}

func TestJSON_NilData(t *testing.T) {
	w := httptest.NewRecorder()

	JSON(w, http.StatusNoContent, nil)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Empty(t, w.Body.String())
}

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()

	Success(w, http.StatusCreated, map[string]string{"id": "123"})

	assert.Equal(t, http.StatusCreated, w.Code)

	var result SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)

	data, ok := result.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "123", data["id"])
}

func TestError(t *testing.T) {
	w := httptest.NewRecorder()

	Error(w, http.StatusBadRequest, "invalid input")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var result ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "invalid input", result.Error)
}

func TestDomainErrorToHTTP(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"nil error", nil, http.StatusOK},
		{"validation error", domain.NewDomainError(domain.ErrCodeValidation, "invalid"), http.StatusBadRequest},
		{"not found error", domain.ErrKnowledgeNotFound, http.StatusNotFound},
		{"already exists error", domain.ErrKnowledgeAlreadyExists, http.StatusConflict},
		{"unauthorized error", domain.ErrInvalidAPIKey, http.StatusUnauthorized},
		{"forbidden error", domain.NewDomainError(domain.ErrCodeForbidden, "forbidden"), http.StatusForbidden},
		{"invalid operation", domain.ErrCannotModifyDeprecated, http.StatusBadRequest},
		{"internal error", domain.NewDomainError(domain.ErrCodeInternalError, "internal"), http.StatusInternalServerError},
		{"unknown domain error", domain.NewDomainError("UNKNOWN", "unknown"), http.StatusInternalServerError},
		{"non-domain error", assert.AnError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DomainErrorToHTTP(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHandleError(t *testing.T) {
	w := httptest.NewRecorder()

	HandleError(w, domain.ErrKnowledgeNotFound)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var result ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Contains(t, result.Error, "not found")
}
