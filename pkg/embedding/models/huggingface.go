package models

import (
	"crypto/sha256"
	"encoding/binary"
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

	// Generate a deterministic vector using a consistent hash of the text
	vector := make([]float32, m.dimension)
	
	// Create a hash of the text
	hash := sha256.Sum256([]byte(text))
	hashBytes := hash[:]  // Convert to slice to make it easier to work with
	
	// Use the hash to seed a pseudorandom number generator
	// This ensures the same text will always generate the same embedding
	for i := range vector {
		// Use a deterministic seed derived from the hash and the dimension index
		// Safely access 4 bytes from the hash (with wrapping)
		byteIndex := (i * 4) % len(hashBytes)
		
		// Make sure we don't go out of bounds when reading 4 bytes
		var seed int64
		if byteIndex + 4 <= len(hashBytes) {
			seed = int64(binary.LittleEndian.Uint32(hashBytes[byteIndex:byteIndex+4]))
		} else {
			// Handle the wrap-around case
			seed = int64(hashBytes[byteIndex]) + 
				int64(hashBytes[(byteIndex+1)%len(hashBytes)])*256 + 
				int64(hashBytes[(byteIndex+2)%len(hashBytes)])*65536 + 
				int64(hashBytes[(byteIndex+3)%len(hashBytes)])*16777216
		}
		
		// Create a deterministic random number generator for this dimension
		r := rand.New(rand.NewSource(seed + int64(i)))
		
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