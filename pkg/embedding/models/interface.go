package models

// EmbeddingModel defines the interface for all embedding models
type EmbeddingModel interface {
	// Embed converts input text into a vector embedding
	Embed(text string) ([]float32, error)
	
	// EmbedBatch converts multiple texts into vector embeddings
	EmbedBatch(texts []string) ([][]float32, error)
	
	// Dimension returns the dimension of the vectors produced by this model
	Dimension() int
	
	// Name returns the name of the model
	Name() string
	
	// Close releases resources used by the model
	Close() error
}

// ModelConfig holds configuration for embedding models
type ModelConfig struct {
	ModelName string
	MaxLength int
	BatchSize int
}

// NewModelConfig creates a new model configuration with default values
func NewModelConfig(modelName string) *ModelConfig {
	return &ModelConfig{
		ModelName: modelName,
		MaxLength: 256,
		BatchSize: 32,
	}
} 