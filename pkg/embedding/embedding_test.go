package embedding

import (
	"testing"

	"github.com/ken/vector_database/pkg/embedding/models"
	"github.com/ken/vector_database/pkg/embedding/pipeline"
	"github.com/stretchr/testify/assert"
)

func TestEmbeddingEngine(t *testing.T) {
	// Create model
	model, err := models.NewHuggingFaceModel(nil)
	assert.NoError(t, err)
	defer model.Close()

	// Create pipeline
	pipe := pipeline.NewPipeline(model)
	pipe.AddProcessor(pipeline.NewTextProcessor())
	pipe.AddProcessor(pipeline.NewJSONProcessor())
	defer pipe.Close()

	// Test text embedding
	text := "This is a test sentence about vector databases."
	vector, err := pipe.ProcessAndEmbed(text, "text")
	assert.NoError(t, err)
	assert.Equal(t, 384, len(vector)) // all-MiniLM-L6-v2 produces 384-dimensional vectors

	// Test JSON embedding
	jsonContent := map[string]interface{}{
		"title": "Vector Database",
		"description": "A database for storing and searching vector embeddings",
		"features": []string{"text search", "similarity search"},
	}
	vector, err = pipe.ProcessAndEmbed(jsonContent, "json")
	assert.NoError(t, err)
	assert.Equal(t, 384, len(vector))

	// Test batch processing
	texts := []interface{}{
		"First test sentence",
		"Second test sentence",
		"Third test sentence",
	}
	vectors, err := pipe.ProcessAndEmbedBatch(texts, "text")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(vectors))
	for _, v := range vectors {
		assert.Equal(t, 384, len(v))
	}
} 