package pipeline

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ken/vector_database/pkg/embedding/models"
)

// ContentProcessor defines the interface for processing different content types
type ContentProcessor interface {
	// Process converts content into a format suitable for embedding
	Process(content interface{}) (string, error)
	
	// Type returns the content type this processor handles
	Type() string
}

// TextProcessor handles plain text content
type TextProcessor struct{}

func NewTextProcessor() *TextProcessor {
	return &TextProcessor{}
}

func (p *TextProcessor) Process(content interface{}) (string, error) {
	switch v := content.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		return "", fmt.Errorf("unsupported content type for text processor: %T", content)
	}
}

func (p *TextProcessor) Type() string {
	return "text"
}

// JSONProcessor handles JSON content
type JSONProcessor struct{}

func NewJSONProcessor() *JSONProcessor {
	return &JSONProcessor{}
}

func (p *JSONProcessor) Process(content interface{}) (string, error) {
	var jsonMap map[string]interface{}
	
	switch v := content.(type) {
	case string:
		if err := json.Unmarshal([]byte(v), &jsonMap); err != nil {
			return "", fmt.Errorf("failed to parse JSON string: %w", err)
		}
	case []byte:
		if err := json.Unmarshal(v, &jsonMap); err != nil {
			return "", fmt.Errorf("failed to parse JSON bytes: %w", err)
		}
	case map[string]interface{}:
		jsonMap = v
	default:
		return "", fmt.Errorf("unsupported content type for JSON processor: %T", content)
	}

	// Convert JSON to a string representation
	var parts []string
	for key, value := range jsonMap {
		parts = append(parts, fmt.Sprintf("%s: %v", key, value))
	}
	
	return strings.Join(parts, " "), nil
}

func (p *JSONProcessor) Type() string {
	return "json"
}

// Pipeline manages content processors and embedding models
type Pipeline struct {
	processors map[string]ContentProcessor
	model      models.EmbeddingModel
}

// NewPipeline creates a new pipeline with the specified model and processors
func NewPipeline(model models.EmbeddingModel) *Pipeline {
	return &Pipeline{
		processors: make(map[string]ContentProcessor),
		model:      model,
	}
}

// AddProcessor adds a content processor to the pipeline
func (p *Pipeline) AddProcessor(processor ContentProcessor) {
	p.processors[processor.Type()] = processor
}

// ProcessAndEmbed processes content and generates embeddings
func (p *Pipeline) ProcessAndEmbed(content interface{}, contentType string) ([]float32, error) {
	processor, ok := p.processors[contentType]
	if !ok {
		return nil, fmt.Errorf("no processor found for content type: %s", contentType)
	}

	processed, err := processor.Process(content)
	if err != nil {
		return nil, fmt.Errorf("failed to process content: %w", err)
	}

	return p.model.Embed(processed)
}

// ProcessAndEmbedBatch processes multiple contents and generates embeddings
func (p *Pipeline) ProcessAndEmbedBatch(contents []interface{}, contentType string) ([][]float32, error) {
	processor, ok := p.processors[contentType]
	if !ok {
		return nil, fmt.Errorf("no processor found for content type: %s", contentType)
	}

	processed := make([]string, len(contents))
	for i, content := range contents {
		result, err := processor.Process(content)
		if err != nil {
			return nil, fmt.Errorf("failed to process content at index %d: %w", i, err)
		}
		processed[i] = result
	}

	return p.model.EmbedBatch(processed)
}

// Close releases resources used by the pipeline
func (p *Pipeline) Close() error {
	if p.model != nil {
		return p.model.Close()
	}
	return nil
} 