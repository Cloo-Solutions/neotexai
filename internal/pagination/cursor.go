package pagination

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"
)

// Cursor represents a decoded pagination cursor
type Cursor struct {
	LastID    string
	Timestamp time.Time
}

// PageResult represents a paginated result set
type PageResult[T any] struct {
	Items   []T    `json:"items"`
	Cursor  string `json:"cursor,omitempty"`
	HasMore bool   `json:"has_more"`
}

var (
	ErrInvalidCursor = errors.New("invalid cursor format")
)

// EncodeCursor creates a base64-encoded cursor from the last item ID and timestamp
func EncodeCursor(lastID string, timestamp time.Time) string {
	if lastID == "" {
		return ""
	}
	raw := lastID + "|" + timestamp.UTC().Format(time.RFC3339Nano)
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes a base64-encoded cursor and returns the last ID and timestamp
func DecodeCursor(cursor string) (*Cursor, error) {
	if cursor == "" {
		return nil, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return nil, ErrInvalidCursor
	}

	parts := strings.SplitN(string(decoded), "|", 2)
	if len(parts) != 2 {
		return nil, ErrInvalidCursor
	}

	timestamp, err := time.Parse(time.RFC3339Nano, parts[1])
	if err != nil {
		return nil, ErrInvalidCursor
	}

	return &Cursor{
		LastID:    parts[0],
		Timestamp: timestamp,
	}, nil
}

// CreateNextCursor creates a cursor for the next page based on the last item
// Returns empty string if there are no more items
func CreateNextCursor[T any](items []T, limit int, getID func(T) string, getTimestamp func(T) time.Time) string {
	if len(items) == 0 || len(items) < limit {
		return ""
	}
	lastItem := items[len(items)-1]
	return EncodeCursor(getID(lastItem), getTimestamp(lastItem))
}
