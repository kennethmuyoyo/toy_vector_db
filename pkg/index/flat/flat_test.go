package flat

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ken/vector_database/pkg/core/distance"
	"github.com/ken/vector_database/pkg/core/vector"
)

func TestNewFlatIndex(t *testing.T) {
	metric := &distance.EuclideanDistance{}
	idx := NewFlatIndex(metric)

	if idx.Name() != "flat" {
		t.Errorf("Expected index name to be 'flat', got %s", idx.Name())
	}

	if idx.metric != metric {
		t.Errorf("Expected metric to be set correctly")
	}

	if len(idx.vectors) != 0 {
		t.Errorf("Expected empty vectors map, got %d items", len(idx.vectors))
	}
}

func TestBuild(t *testing.T) {
	idx := NewFlatIndex(&distance.EuclideanDistance{})

	// Create test vectors
	vectors := []*vector.Vector{
		vector.NewVector("v1", []float32{1.0, 2.0, 3.0}),
		vector.NewVector("v2", []float32{4.0, 5.0, 6.0}),
		vector.NewVector("v3", []float32{7.0, 8.0, 9.0}),
	}

	// Build the index
	err := idx.Build(vectors)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Check that all vectors were added
	if idx.Size() != len(vectors) {
		t.Errorf("Expected %d vectors, got %d", len(vectors), idx.Size())
	}

	// Check that each vector is retrievable and is a copy
	for _, v := range vectors {
		idxVec, exists := idx.vectors[v.ID]
		if !exists {
			t.Errorf("Vector %s not found in index", v.ID)
			continue
		}

		// Check that it's a different object (a copy)
		if idxVec == v {
			t.Errorf("Vector %s was not copied", v.ID)
		}

		// Check that the values are the same
		for i, val := range v.Values {
			if idxVec.Values[i] != val {
				t.Errorf("Vector %s value at index %d is %f, expected %f", v.ID, i, idxVec.Values[i], val)
			}
		}
	}

	// Test that building replaces existing vectors
	newVectors := []*vector.Vector{
		vector.NewVector("v4", []float32{10.0, 11.0, 12.0}),
	}

	err = idx.Build(newVectors)
	if err != nil {
		t.Fatalf("Second build failed: %v", err)
	}

	if idx.Size() != 1 {
		t.Errorf("Expected 1 vector after rebuild, got %d", idx.Size())
	}

	if _, exists := idx.vectors["v1"]; exists {
		t.Errorf("Vector v1 should have been removed after rebuild")
	}

	if _, exists := idx.vectors["v4"]; !exists {
		t.Errorf("Vector v4 should exist after rebuild")
	}
}

func TestAdd(t *testing.T) {
	idx := NewFlatIndex(&distance.EuclideanDistance{})

	// Add a vector
	v1 := vector.NewVector("v1", []float32{1.0, 2.0, 3.0})
	err := idx.Add(v1)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Check that the vector was added
	if idx.Size() != 1 {
		t.Errorf("Expected 1 vector, got %d", idx.Size())
	}

	// Try to add a vector with the same ID
	v1Duplicate := vector.NewVector("v1", []float32{4.0, 5.0, 6.0})
	err = idx.Add(v1Duplicate)
	if err != ErrVectorAlreadyExists {
		t.Errorf("Expected ErrVectorAlreadyExists, got %v", err)
	}

	// Add another vector
	v2 := vector.NewVector("v2", []float32{4.0, 5.0, 6.0})
	err = idx.Add(v2)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Check that both vectors are in the index
	if idx.Size() != 2 {
		t.Errorf("Expected 2 vectors, got %d", idx.Size())
	}
}

func TestDelete(t *testing.T) {
	idx := NewFlatIndex(&distance.EuclideanDistance{})

	// Add some vectors
	vectors := []*vector.Vector{
		vector.NewVector("v1", []float32{1.0, 2.0, 3.0}),
		vector.NewVector("v2", []float32{4.0, 5.0, 6.0}),
	}

	for _, v := range vectors {
		err := idx.Add(v)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	// Delete a vector
	err := idx.Delete("v1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Check that the vector was deleted
	if idx.Size() != 1 {
		t.Errorf("Expected 1 vector after delete, got %d", idx.Size())
	}

	if _, exists := idx.vectors["v1"]; exists {
		t.Errorf("Vector v1 should have been deleted")
	}

	// Try to delete a non-existent vector
	err = idx.Delete("v3")
	if err != ErrVectorNotFound {
		t.Errorf("Expected ErrVectorNotFound, got %v", err)
	}
}

func TestSearch(t *testing.T) {
	// Create an index with vectors at known positions
	idx := NewFlatIndex(&distance.EuclideanDistance{})

	// Add vectors to the index with known distance from origin
	vectors := []*vector.Vector{
		vector.NewVector("v1", []float32{1.0, 0.0, 0.0}), // Distance to origin: 1.0
		vector.NewVector("v2", []float32{2.0, 0.0, 0.0}), // Distance to origin: 2.0
		vector.NewVector("v3", []float32{3.0, 0.0, 0.0}), // Distance to origin: 3.0
	}

	for _, v := range vectors {
		err := idx.Add(v)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	// Create a query vector at the origin
	query := vector.NewVector("query", []float32{0.0, 0.0, 0.0})

	// Search with k = 2
	results, err := idx.Search(query, 2)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Check that we got 2 results
	if len(results) != 2 {
		t.Errorf("Expected 2 search results, got %d", len(results))
	}

	// Check that the results are sorted by distance
	if results[0].Distance > results[1].Distance {
		t.Errorf("Results not sorted by distance")
	}

	// Check that v1 is closest
	if results[0].ID != "v1" {
		t.Errorf("Expected v1 to be closest, got %s", results[0].ID)
	}

	// Test with k larger than the number of vectors
	results, err = idx.Search(query, 10)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 search results, got %d", len(results))
	}

	// Test with k = 0 (invalid)
	_, err = idx.Search(query, 0)
	if err != ErrInvalidK {
		t.Errorf("Expected ErrInvalidK, got %v", err)
	}

	// Test with empty index
	emptyIdx := NewFlatIndex(&distance.EuclideanDistance{})
	_, err = emptyIdx.Search(query, 1)
	if err != ErrNoVectors {
		t.Errorf("Expected ErrNoVectors, got %v", err)
	}

	// Test with no metric
	noMetricIdx := &FlatIndex{
		vectors: make(map[string]*vector.Vector),
		metric:  nil,
	}
	noMetricIdx.Add(vector.NewVector("v1", []float32{1.0, 2.0, 3.0}))
	_, err = noMetricIdx.Search(query, 1)
	if err != ErrMetricRequired {
		t.Errorf("Expected ErrMetricRequired, got %v", err)
	}
}

func TestSaveLoad(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "vectodb-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	indexPath := filepath.Join(tempDir, "index.gob")

	// Create an index with some vectors
	originalIndex := NewFlatIndex(&distance.EuclideanDistance{})
	vectors := []*vector.Vector{
		vector.NewVector("v1", []float32{1.0, 2.0, 3.0}),
		vector.NewVector("v2", []float32{4.0, 5.0, 6.0}),
	}

	for _, v := range vectors {
		err := originalIndex.Add(v)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	// Save the index
	err = originalIndex.Save(indexPath)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Create a new empty index
	newIndex := NewFlatIndex(&distance.EuclideanDistance{})

	// Load the index
	err = newIndex.Load(indexPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Check that the new index has the same vectors
	if newIndex.Size() != originalIndex.Size() {
		t.Errorf("Expected %d vectors in loaded index, got %d", originalIndex.Size(), newIndex.Size())
	}

	// Check that all vectors from the original index are in the new index
	for id, vec := range originalIndex.vectors {
		newVec, exists := newIndex.vectors[id]
		if !exists {
			t.Errorf("Vector %s not found in loaded index", id)
			continue
		}

		// Check that the values are the same
		for i, val := range vec.Values {
			if newVec.Values[i] != val {
				t.Errorf("Vector %s value at index %d is %f, expected %f", id, i, newVec.Values[i], val)
			}
		}
	}
}

func TestSetMetric(t *testing.T) {
	// Create an index with one metric
	idx := NewFlatIndex(&distance.EuclideanDistance{})

	// Add a vector
	v1 := vector.NewVector("v1", []float32{1.0, 2.0, 3.0})
	err := idx.Add(v1)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Create a query vector
	query := vector.NewVector("query", []float32{4.0, 5.0, 6.0})

	// Get results with Euclidean distance
	euclideanResults, err := idx.Search(query, 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Change to Cosine distance
	idx.SetMetric(&distance.CosineDistance{})

	// Get results with Cosine distance
	cosineResults, err := idx.Search(query, 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// The results should be different because the metrics are different
	if euclideanResults[0].Distance == cosineResults[0].Distance {
		t.Errorf("Expected different distances with different metrics, got %.6f for both", euclideanResults[0].Distance)
	}
} 