package index

import (
	"github.com/ken/vector_database/pkg/core/distance"
	"github.com/ken/vector_database/pkg/core/vector"
)

// SearchResult represents a single search result with ID and distance
type SearchResult struct {
	ID       string  // Vector ID
	Vector   *vector.Vector // The actual vector if requested
	Distance float32 // Distance from query vector
}

// SearchResults is a slice of SearchResult
type SearchResults []SearchResult

// Index is the interface that all index implementations must satisfy
type Index interface {
	// Name returns the name of the index
	Name() string

	// Build constructs the index from a set of vectors
	Build(vectors []*vector.Vector) error

	// Add adds a vector to the index
	Add(vec *vector.Vector) error

	// Delete removes a vector from the index
	Delete(id string) error

	// Search performs a k-nearest neighbor search
	Search(query *vector.Vector, k int) (SearchResults, error)

	// Size returns the number of vectors in the index
	Size() int

	// GetIDs returns all vector IDs in the index
	GetIDs() []string

	// Save persists the index to the specified path
	Save(path string) error

	// Load loads the index from the specified path
	Load(path string) error

	// SetMetric sets the distance metric used by the index
	SetMetric(metric distance.Metric)
}

// SortSearchResults sorts search results by distance (ascending)
func (r SearchResults) Sort() {
	// Simple bubble sort implementation
	n := len(r)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if r[j].Distance > r[j+1].Distance {
				r[j], r[j+1] = r[j+1], r[j]
			}
		}
	}
} 