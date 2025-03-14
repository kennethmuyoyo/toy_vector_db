package models

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
)

// HuggingFaceModel implements the EmbeddingModel interface
// Note: This is a mock implementation for demonstration purposes
type HuggingFaceModel struct {
	config     *ModelConfig
	modelMutex sync.RWMutex
	dimension  int
}

// NewHuggingFaceModel creates a new model instance
func NewHuggingFaceModel(config *ModelConfig) (*HuggingFaceModel, error) {
	if config == nil {
		config = NewModelConfig("sentence-transformers/all-MiniLM-L6-v2")
	}

	// In a real implementation, we would load the model here
	// For now, we'll use a mock implementation
	
	return &HuggingFaceModel{
		config:    config,
		dimension: 384, // all-MiniLM-L6-v2 produces 384-dimensional vectors
	}, nil
}

// Embed converts input text into a vector embedding
func (m *HuggingFaceModel) Embed(text string) ([]float32, error) {
	m.modelMutex.RLock()
	defer m.modelMutex.RUnlock()

	// In a real implementation, we would use the model to generate embeddings
	// For now, we'll generate a deterministic vector based on the text
	
	vector := make([]float32, m.dimension)
	
	// Generate a deterministic vector based on the text
	// This is not a real embedding, just a demonstration
	seed := int64(0)
	for _, c := range text {
		seed += int64(c)
	}
	r := rand.New(rand.NewSource(seed))
	
	for i := range vector {
		// Generate values between -1 and 1
		vector[i] = float32(r.Float64()*2 - 1)
	}
	
	// Normalize the vector
	norm := float32(0)
	for _, v := range vector {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))
	
	if norm > 0 {
		for i := range vector {
			vector[i] /= norm
		}
	}
	
	return vector, nil
}

// EmbedBatch converts multiple texts into vector embeddings
func (m *HuggingFaceModel) EmbedBatch(texts []string) ([][]float32, error) {
	m.modelMutex.RLock()
	defer m.modelMutex.RUnlock()

	results := make([][]float32, len(texts))
	for i, text := range texts {
		vector, err := m.Embed(text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text at index %d: %w", i, err)
		}
		results[i] = vector
	}

	return results, nil
}

// Dimension returns the dimension of the vectors produced by this model
func (m *HuggingFaceModel) Dimension() int {
	return m.dimension
}

// Name returns the name of the model
func (m *HuggingFaceModel) Name() string {
	return m.config.ModelName
}

// Close releases resources used by the model
func (m *HuggingFaceModel) Close() error {
	return nil
} 