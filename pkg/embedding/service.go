package embedding

import (
	"encoding/json"
	"fmt"
	"sync"
)

// Service provides high-level embedding functionality for documents
type Service struct {
	engine      *Engine
	cacheMutex  sync.RWMutex
	modelConfig *Config
}

// NewService creates a new embedding service with the specified configuration
func NewService(config *Config) (*Service, error) {
	if config == nil {
		config = DefaultConfig()
	}

	engine, err := NewEngine(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding engine: %w", err)
	}

	return &Service{
		engine:      engine,
		modelConfig: config,
	}, nil
}

// ProcessDocument generates vector embedding for a document
func (s *Service) ProcessDocument(doc *Document) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	var vector []float32
	var err error

	switch doc.ContentType {
	case ContentTypeText:
		content, ok := doc.Content.(string)
		if !ok {
			return fmt.Errorf("content is not a string for text document")
		}
		vector, err = s.engine.EmbedText(content)
	case ContentTypeJSON:
		content, ok := doc.Content.(map[string]interface{})
		if !ok {
			// Try to convert if it's a different type
			if jsonStr, ok := doc.Content.(string); ok {
				var jsonMap map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &jsonMap); err != nil {
					return fmt.Errorf("failed to parse JSON content: %w", err)
				}
				doc.Content = jsonMap
				content = jsonMap
			} else {
				return fmt.Errorf("content is not a JSON object for JSON document")
			}
		}
		vector, err = s.engine.EmbedJSON(content)
	default:
		return fmt.Errorf("unsupported content type: %s", doc.ContentType)
	}

	if err != nil {
		return fmt.Errorf("failed to embed document content: %w", err)
	}

	doc.Vector = vector
	doc.SetMetadata("embedding_model", s.engine.ModelName())
	doc.SetMetadata("vector_dimension", s.engine.ModelDimension())

	return nil
}

// ProcessDocuments generates vector embeddings for multiple documents
func (s *Service) ProcessDocuments(docs []*Document) error {
	for i, doc := range docs {
		err := s.ProcessDocument(doc)
		if err != nil {
			return fmt.Errorf("failed to process document at index %d: %w", i, err)
		}
	}
	return nil
}

// Close releases resources used by the service
func (s *Service) Close() error {
	if s.engine != nil {
		return s.engine.Close()
	}
	return nil
} 