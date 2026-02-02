package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsJSONInput(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected bool
	}{
		{"json object", []byte(`{"type":"guideline"}`), true},
		{"json array", []byte(`[{"type":"guideline"}]`), true},
		{"json with whitespace", []byte(`  {"type":"guideline"}`), true},
		{"json array with whitespace", []byte(`  [{"type":"guideline"}]`), true},
		{"markdown", []byte(`# Hello World`), false},
		{"plain text", []byte(`hello world`), false},
		{"empty", []byte(``), false},
		{"only whitespace", []byte(`   `), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isJSONInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
