package embedding

import (
	"fmt"

	"github.com/ken/vector_database/pkg/embedding/models"
	"github.com/ken/vector_database/pkg/embedding/pipeline"
)

// Engine is the main embedding engine that provides embedding functionality
type Engine struct {
	model       models.EmbeddingModel
	pipeline    *pipeline.Pipeline
	initialized bool
}

// Config holds configuration for the embedding engine
type Config struct {
	ModelName     string
	ModelMaxLength int
	ModelBatchSize int
}

// DefaultConfig returns a default configuration for the embedding engine
func DefaultConfig() *Config {
	return &Config{
		ModelName:     "sentence-transformers/all-MiniLM-L6-v2",
		ModelMaxLength: 256,
		ModelBatchSize: 32,
	}
}

// NewEngine creates a new embedding engine with the specified configuration
func NewEngine(config *Config) (*Engine, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create model configuration
	modelConfig := &models.ModelConfig{
		ModelName: config.ModelName,
		MaxLength: config.ModelMaxLength,
		BatchSize: config.ModelBatchSize,
	}

	// Create model
	model, err := models.NewHuggingFaceModel(modelConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Hugging Face model: %w", err)
	}

	// Create pipeline
	p := pipeline.NewPipeline(model)
	p.AddProcessor(pipeline.NewTextProcessor())
	p.AddProcessor(pipeline.NewJSONProcessor())

	return &Engine{
		model:    model,
		pipeline: p,
		initialized: true,
	}, nil
}

// EmbedText embeds a text string into a vector
func (e *Engine) EmbedText(text string) ([]float32, error) {
	if !e.initialized {
		return nil, fmt.Errorf("embedding engine not initialized")
	}
	return e.pipeline.ProcessAndEmbed(text, "text")
}

// EmbedJSON embeds a JSON object into a vector
func (e *Engine) EmbedJSON(jsonContent map[string]interface{}) ([]float32, error) {
	if !e.initialized {
		return nil, fmt.Errorf("embedding engine not initialized")
	}
	return e.pipeline.ProcessAndEmbed(jsonContent, "json")
}

// EmbedBatch embeds multiple texts into vectors
func (e *Engine) EmbedBatch(texts []string) ([][]float32, error) {
	if !e.initialized {
		return nil, fmt.Errorf("embedding engine not initialized")
	}
	
	contents := make([]interface{}, len(texts))
	for i, text := range texts {
		contents[i] = text
	}
	
	return e.pipeline.ProcessAndEmbedBatch(contents, "text")
}

// ModelDimension returns the dimension of the vectors produced by the model
func (e *Engine) ModelDimension() int {
	return e.model.Dimension()
}

// ModelName returns the name of the embedding model
func (e *Engine) ModelName() string {
	return e.model.Name()
}

// Close releases resources used by the embedding engine
func (e *Engine) Close() error {
	if !e.initialized {
		return nil
	}
	
	err := e.pipeline.Close()
	e.initialized = false
	return err
} 