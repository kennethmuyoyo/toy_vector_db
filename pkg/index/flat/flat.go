package flat

import (
	"encoding/gob"
	"errors"
	"os"
	"sync"

	"github.com/ken/vector_database/pkg/core/distance"
	"github.com/ken/vector_database/pkg/core/vector"
	"github.com/ken/vector_database/pkg/index"
)

var (
	// ErrVectorNotFound is returned when a vector with the specified ID is not found
	ErrVectorNotFound = errors.New("vector not found")

	// ErrVectorAlreadyExists is returned when attempting to add a vector with an ID that already exists
	ErrVectorAlreadyExists = errors.New("vector already exists")

	// ErrInvalidK is returned when k is less than 1
	ErrInvalidK = errors.New("k must be greater than 0")

	// ErrNoVectors is returned when the index is empty
	ErrNoVectors = errors.New("index contains no vectors")

	// ErrMetricRequired is returned when a distance metric is required but not set
	ErrMetricRequired = errors.New("distance metric is required")
)

// FlatIndex implements a brute-force nearest neighbor search index
type FlatIndex struct {
	vectors map[string]*vector.Vector // Map of vector ID to vector
	metric  distance.Metric           // Distance metric to use
	mu      sync.RWMutex              // Mutex for thread safety
}

// NewFlatIndex creates a new flat index with the specified distance metric
func NewFlatIndex(metric distance.Metric) *FlatIndex {
	return &FlatIndex{
		vectors: make(map[string]*vector.Vector),
		metric:  metric,
	}
}

// Name returns the name of the index
func (idx *FlatIndex) Name() string {
	return "flat"
}

// Build constructs the index from a set of vectors
func (idx *FlatIndex) Build(vectors []*vector.Vector) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Reset the index
	idx.vectors = make(map[string]*vector.Vector)

	// Add each vector to the index
	for _, vec := range vectors {
		idx.vectors[vec.ID] = vec.Copy() // Store a copy of the vector
	}

	return nil
}

// Add adds a vector to the index
func (idx *FlatIndex) Add(vec *vector.Vector) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Check if the vector already exists
	if _, exists := idx.vectors[vec.ID]; exists {
		return ErrVectorAlreadyExists
	}

	// Add the vector to the index
	idx.vectors[vec.ID] = vec.Copy() // Store a copy of the vector

	return nil
}

// Delete removes a vector from the index
func (idx *FlatIndex) Delete(id string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Check if the vector exists
	if _, exists := idx.vectors[id]; !exists {
		return ErrVectorNotFound
	}

	// Delete the vector
	delete(idx.vectors, id)

	return nil
}

// Search performs a k-nearest neighbor search
func (idx *FlatIndex) Search(query *vector.Vector, k int) (index.SearchResults, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Check if the index is empty
	if len(idx.vectors) == 0 {
		return nil, ErrNoVectors
	}

	// Check if k is valid
	if k < 1 {
		return nil, ErrInvalidK
	}

	// Check if a metric is set
	if idx.metric == nil {
		return nil, ErrMetricRequired
	}

	// Calculate distances to all vectors
	results := make(index.SearchResults, 0, len(idx.vectors))
	for id, vec := range idx.vectors {
		// Calculate distance
		dist, err := idx.metric.Distance(query, vec)
		if err != nil {
			return nil, err
		}

		// Add to results
		results = append(results, index.SearchResult{
			ID:       id,
			Vector:   vec.Copy(), // Return a copy to prevent modification
			Distance: dist,
		})
	}

	// Sort results by distance
	results.Sort()

	// Return top k results
	if k > len(results) {
		k = len(results)
	}
	return results[:k], nil
}

// Size returns the number of vectors in the index
func (idx *FlatIndex) Size() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return len(idx.vectors)
}

// GetIDs returns all vector IDs in the index
func (idx *FlatIndex) GetIDs() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	ids := make([]string, 0, len(idx.vectors))
	for id := range idx.vectors {
		ids = append(ids, id)
	}

	return ids
}

// Save persists the index to the specified path
func (idx *FlatIndex) Save(path string) error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Create the file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a gob encoder
	encoder := gob.NewEncoder(file)

	// Create a serializable version of the index
	type indexData struct {
		Vectors map[string]*vector.Vector
		Metric  string
	}

	// Get the metric name
	var metricName string
	if idx.metric != nil {
		metricName = string(idx.metric.Name())
	}

	// Encode the index
	data := indexData{
		Vectors: idx.vectors,
		Metric:  metricName,
	}
	if err := encoder.Encode(data); err != nil {
		return err
	}

	return nil
}

// Load loads the index from the specified path
func (idx *FlatIndex) Load(path string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a gob decoder
	decoder := gob.NewDecoder(file)

	// Define the serializable version of the index
	type indexData struct {
		Vectors map[string]*vector.Vector
		Metric  string
	}

	// Decode the index
	var data indexData
	if err := decoder.Decode(&data); err != nil {
		return err
	}

	// Update the index
	idx.vectors = data.Vectors

	// Set the metric if it's not already set
	if idx.metric == nil && data.Metric != "" {
		metric, err := distance.GetMetric(distance.MetricType(data.Metric))
		if err != nil {
			return err
		}
		idx.metric = metric
	}

	return nil
}

// SetMetric sets the distance metric used by the index
func (idx *FlatIndex) SetMetric(metric distance.Metric) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.metric = metric
} 