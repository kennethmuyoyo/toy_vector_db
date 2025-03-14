package embedding

import (
	"encoding/json"
	"fmt"
	"time"
)

// ContentType represents the type of content stored in a document
type ContentType string

const (
	ContentTypeText ContentType = "text"
	ContentTypeJSON ContentType = "json"
)

// Document represents a document with content and its vector embedding
type Document struct {
	ID          string                 `json:"id"`
	Content     interface{}            `json:"content"`
	ContentType ContentType            `json:"content_type"`
	Vector      []float32              `json:"vector,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// NewDocument creates a new document with the specified content
func NewDocument(id string, content interface{}, contentType ContentType) *Document {
	now := time.Now()
	return &Document{
		ID:          id,
		Content:     content,
		ContentType: contentType,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// NewTextDocument creates a new document with text content
func NewTextDocument(id string, content string) *Document {
	return NewDocument(id, content, ContentTypeText)
}

// NewJSONDocument creates a new document with JSON content
func NewJSONDocument(id string, content map[string]interface{}) *Document {
	return NewDocument(id, content, ContentTypeJSON)
}

// SetMetadata sets a metadata value for the document
func (d *Document) SetMetadata(key string, value interface{}) {
	if d.Metadata == nil {
		d.Metadata = make(map[string]interface{})
	}
	d.Metadata[key] = value
	d.UpdatedAt = time.Now()
}

// GetMetadata gets a metadata value from the document
func (d *Document) GetMetadata(key string) (interface{}, bool) {
	if d.Metadata == nil {
		return nil, false
	}
	value, ok := d.Metadata[key]
	return value, ok
}

// ToJSON converts the document to a JSON string
func (d *Document) ToJSON() (string, error) {
	bytes, err := json.Marshal(d)
	if err != nil {
		return "", fmt.Errorf("failed to marshal document to JSON: %w", err)
	}
	return string(bytes), nil
}

// FromJSON populates a document from a JSON string
func DocumentFromJSON(jsonStr string) (*Document, error) {
	var doc Document
	err := json.Unmarshal([]byte(jsonStr), &doc)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal document from JSON: %w", err)
	}
	return &doc, nil
} 