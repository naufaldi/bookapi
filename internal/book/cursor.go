package book

import (
	"encoding/base64"
	"encoding/json"
)

// CursorData represents the data encoded in a cursor
type CursorData struct {
	AfterID   string `json:"after_id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

// EncodeCursor encodes cursor data to a base64 string
func EncodeCursor(data CursorData) string {
	if data.AfterID == "" {
		return ""
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(jsonBytes)
}

// DecodeCursor decodes a base64 cursor string to CursorData
func DecodeCursor(cursor string) (CursorData, error) {
	if cursor == "" {
		return CursorData{}, nil
	}

	decoded, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return CursorData{}, err
	}

	var data CursorData
	err = json.Unmarshal(decoded, &data)
	return data, err
}
