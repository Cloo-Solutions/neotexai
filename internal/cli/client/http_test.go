package client

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgressReader_ReportsProgress(t *testing.T) {
	data := []byte("hello world this is test data")
	reader := bytes.NewReader(data)

	var progressCalls []struct{ current, total int64 }
	pr := &progressReader{
		reader: reader,
		total:  int64(len(data)),
		onProgress: func(current, total int64) {
			progressCalls = append(progressCalls, struct{ current, total int64 }{current, total})
		},
	}

	result, err := io.ReadAll(pr)
	require.NoError(t, err)
	assert.Equal(t, data, result)

	// Progress should have been called at least once
	assert.NotEmpty(t, progressCalls)

	// Final progress should equal total
	lastCall := progressCalls[len(progressCalls)-1]
	assert.Equal(t, int64(len(data)), lastCall.current)
	assert.Equal(t, int64(len(data)), lastCall.total)
}

func TestProgressReader_NilCallback(t *testing.T) {
	data := []byte("hello world")
	reader := bytes.NewReader(data)

	pr := &progressReader{
		reader:     reader,
		total:      int64(len(data)),
		onProgress: nil, // No callback
	}

	result, err := io.ReadAll(pr)
	require.NoError(t, err)
	assert.Equal(t, data, result)
}

func TestProgressReader_SmallReads(t *testing.T) {
	data := []byte("hello world")
	reader := bytes.NewReader(data)

	var progressValues []int64
	pr := &progressReader{
		reader: reader,
		total:  int64(len(data)),
		onProgress: func(current, total int64) {
			progressValues = append(progressValues, current)
		},
	}

	// Read one byte at a time
	buf := make([]byte, 1)
	for {
		n, err := pr.Read(buf)
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		assert.Equal(t, 1, n)
	}

	// Progress should increase monotonically
	for i := 1; i < len(progressValues); i++ {
		assert.GreaterOrEqual(t, progressValues[i], progressValues[i-1])
	}
}
