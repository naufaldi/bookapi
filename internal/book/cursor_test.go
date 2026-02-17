package book

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeCursor(t *testing.T) {
	t.Run("empty data", func(t *testing.T) {
		result := EncodeCursor(CursorData{})
		assert.Empty(t, result)
	})

	t.Run("with after_id", func(t *testing.T) {
		result := EncodeCursor(CursorData{AfterID: "abc123"})
		assert.NotEmpty(t, result)
		// Should be base64 encoded JSON
		assert.Equal(t, "eyJhZnRlcl9pZCI6ImFiYzEyMyJ9", result)
	})
}

func TestDecodeCursor(t *testing.T) {
	t.Run("empty cursor", func(t *testing.T) {
		data, err := DecodeCursor("")
		assert.NoError(t, err)
		assert.Equal(t, CursorData{}, data)
	})

	t.Run("valid cursor", func(t *testing.T) {
		// "eyJhZnRlcl9pZCI6ImFiYzEyMyJ9" = {"after_id":"abc123"}
		data, err := DecodeCursor("eyJhZnRlcl9pZCI6ImFiYzEyMyJ9")
		assert.NoError(t, err)
		assert.Equal(t, "abc123", data.AfterID)
	})

	t.Run("invalid cursor", func(t *testing.T) {
		data, err := DecodeCursor("invalid-base64!!!")
		assert.Error(t, err)
		assert.Equal(t, CursorData{}, data)
	})
}

func TestRoundTrip(t *testing.T) {
	original := CursorData{AfterID: "test-uuid-123"}
	encoded := EncodeCursor(original)
	decoded, err := DecodeCursor(encoded)
	assert.NoError(t, err)
	assert.Equal(t, original.AfterID, decoded.AfterID)
}
