package hnsw

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ken/vector_database/pkg/core/distance"
	"github.com/ken/vector_database/pkg/core/vector"
)

func TestNewHNSWIndex(t *testing.T) {
	metric := &distance.EuclideanDistance{}
	idx := NewHNSWIndex(metric, nil)

	if idx.Name() != "hnsw" {
		t.Errorf("Expected index name to be 'hnsw', got %s", idx.Name())
	}

	if idx.metric != metric {
		t.Errorf("Expected metric to be set correctly")
	}

	if len(idx.nodes) != 0 {
		t.Errorf("Expected empty nodes map, got %d items", len(idx.nodes))
	}

	// Test with custom config
	config := &HNSWConfig{
		M:              32,
		EfConstruction: 100,
		EfSearch:       30,
		MaxLevel:       5,
		LevelMult:      0.5,
	}
	customIdx := NewHNSWIndex(metric, config)

	if customIdx.config.M != 32 {
		t.Errorf("Expected M to be 32, got %d", customIdx.config.M)
	}
	if customIdx.config.EfConstruction != 100 {
		t.Errorf("Expected EfConstruction to be 100, got %d", customIdx.config.EfConstruction)
	}
	if customIdx.config.EfSearch != 30 {
		t.Errorf("Expected EfSearch to be 30, got %d", customIdx.config.EfSearch)
	}
	if customIdx.config.MaxLevel != 5 {
		t.Errorf("Expected MaxLevel to be 5, got %d", customIdx.config.MaxLevel)
	}
	if customIdx.config.LevelMult != 0.5 {
		t.Errorf("Expected LevelMult to be 0.5, got %f", customIdx.config.LevelMult)
	}
}

func TestBuild(t *testing.T) {
	idx := NewHNSWIndex(&distance.EuclideanDistance{}, nil)

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
		node, exists := idx.nodes[v.ID]
		if !exists {
			t.Errorf("Vector %s not found in index", v.ID)
			continue
		}

		// Check that it's a different object (a copy)
		if node.Vector == v {
			t.Errorf("Vector %s was not copied", v.ID)
		}

		// Check that the values are the same
		for i, val := range v.Values {
			if node.Vector.Values[i] != val {
				t.Errorf("Vector %s value at index %d is %f, expected %f", v.ID, i, node.Vector.Values[i], val)
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

	if _, exists := idx.nodes["v1"]; exists {
		t.Errorf("Vector v1 should have been removed after rebuild")
	}

	if _, exists := idx.nodes["v4"]; !exists {
		t.Errorf("Vector v4 should exist after rebuild")
	}

	// Check that entry point is set
	if idx.entryPoint == "" {
		t.Errorf("Entry point should be set after build")
	}
}

func TestAdd(t *testing.T) {
	idx := NewHNSWIndex(&distance.EuclideanDistance{}, nil)

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

	// Check that the entry point is set
	if idx.entryPoint != "v1" {
		t.Errorf("Expected entry point to be v1, got %s", idx.entryPoint)
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

	// Check that connections exist between nodes
	node1 := idx.nodes["v1"]
	node2 := idx.nodes["v2"]

	// If levels allow, there should be connections at level 0
	if len(node1.Edges) > 0 && len(node1.Edges[0]) == 0 {
		t.Errorf("Node v1 should have connections at level 0")
	}
	if len(node2.Edges) > 0 && len(node2.Edges[0]) == 0 {
		t.Errorf("Node v2 should have connections at level 0")
	}
}

func TestDelete(t *testing.T) {
	idx := NewHNSWIndex(&distance.EuclideanDistance{}, nil)

	// Add some vectors
	vectors := []*vector.Vector{
		vector.NewVector("v1", []float32{1.0, 2.0, 3.0}),
		vector.NewVector("v2", []float32{4.0, 5.0, 6.0}),
		vector.NewVector("v3", []float32{7.0, 8.0, 9.0}),
	}

	for _, v := range vectors {
		err := idx.Add(v)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	// Remember the entry point
	originalEntryPoint := idx.entryPoint

	// Delete a vector that is not the entry point
	nonEntryPointID := ""
	for _, v := range vectors {
		if v.ID != originalEntryPoint {
			nonEntryPointID = v.ID
			break
		}
	}

	err := idx.Delete(nonEntryPointID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Check that the vector was marked as deleted
	if !idx.nodes[nonEntryPointID].Deleted {
		t.Errorf("Vector %s should be marked as deleted", nonEntryPointID)
	}

	// Check that the entry point remains the same
	if idx.entryPoint != originalEntryPoint {
		t.Errorf("Entry point changed after deleting a non-entry point vector")
	}

	// Delete the entry point
	err = idx.Delete(originalEntryPoint)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Check that the entry point was updated
	if idx.entryPoint == originalEntryPoint {
		t.Errorf("Entry point should be updated after deleting the entry point")
	}

	// Check the size
	if idx.Size() != 1 {
		t.Errorf("Expected 1 non-deleted vector, got %d", idx.Size())
	}

	// Try to delete a non-existent vector
	err = idx.Delete("non-existent")
	if err != ErrVectorNotFound {
		t.Errorf("Expected ErrVectorNotFound, got %v", err)
	}
}

func TestSearch(t *testing.T) {
	// Create a simple index with known positions
	idx := NewHNSWIndex(&distance.EuclideanDistance{}, nil)

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
	emptyIdx := NewHNSWIndex(&distance.EuclideanDistance{}, nil)
	_, err = emptyIdx.Search(query, 1)
	if err != ErrNoVectors {
		t.Errorf("Expected ErrNoVectors, got %v", err)
	}

	// Test with no metric
	noMetricIdx := NewHNSWIndex(nil, nil)
	noMetricIdx.Add(vector.NewVector("v1", []float32{1.0, 2.0, 3.0}))
	_, err = noMetricIdx.Search(query, 1)
	if err != ErrMetricRequired {
		t.Errorf("Expected ErrMetricRequired, got %v", err)
	}
}

// Test deletion and search separately
func TestDeleteAndSearch(t *testing.T) {
	// Create an index
	idx := NewHNSWIndex(&distance.EuclideanDistance{}, nil)
	
	// Add multiple vectors
	vectors := []*vector.Vector{
		vector.NewVector("v1", []float32{1.0, 0.0, 0.0}),
		vector.NewVector("v2", []float32{2.0, 0.0, 0.0}),
		vector.NewVector("v3", []float32{3.0, 0.0, 0.0}),
	}
	
	for _, v := range vectors {
		err := idx.Add(v)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}
	
	// Save the entry point
	originalEntryPoint := idx.entryPoint
	
	// Delete the entry point
	err := idx.Delete(originalEntryPoint)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	
	// Check that entry point is updated
	if idx.entryPoint == originalEntryPoint {
		t.Errorf("Entry point should be updated after deletion")
	}
	
	// Create a query vector
	query := vector.NewVector("query", []float32{0.0, 0.0, 0.0})
	
	// Search after deletion
	results, err := idx.Search(query, 10)
	if err != nil {
		t.Fatalf("Search after deletion failed: %v", err)
	}
	
	// Check that we get the right number of results (all vectors minus deleted ones)
	expectedResults := len(vectors) - 1
	if len(results) != expectedResults {
		t.Errorf("Expected %d results after deletion, got %d", expectedResults, len(results))
	}
	
	// Check that the deleted vector is not in the results
	for _, result := range results {
		if result.ID == originalEntryPoint {
			t.Errorf("Deleted vector should not be in search results")
		}
	}
	
	// Delete all remaining vectors
	for _, v := range vectors {
		if v.ID != originalEntryPoint {
			idx.Delete(v.ID)
		}
	}
	
	// Search after all deletions
	_, err = idx.Search(query, 1)
	if err != ErrNoVectors {
		t.Errorf("Expected ErrNoVectors after all deletions, got %v", err)
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
	config := &HNSWConfig{
		M:              8,
		EfConstruction: 50,
		EfSearch:       20,
		MaxLevel:       3,
		LevelMult:      0.4,
	}
	originalIndex := NewHNSWIndex(&distance.EuclideanDistance{}, config)

	vectors := []*vector.Vector{
		vector.NewVector("v1", []float32{1.0, 2.0, 3.0}),
		vector.NewVector("v2", []float32{4.0, 5.0, 6.0}),
		vector.NewVector("v3", []float32{7.0, 8.0, 9.0}),
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
	newIndex := NewHNSWIndex(&distance.EuclideanDistance{}, nil)

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
	for id, node := range originalIndex.nodes {
		if node.Deleted {
			continue
		}

		newNode, exists := newIndex.nodes[id]
		if !exists {
			t.Errorf("Vector %s not found in loaded index", id)
			continue
		}

		if newNode.Deleted {
			t.Errorf("Vector %s incorrectly marked as deleted in loaded index", id)
		}

		// Check that the values are the same
		for i, val := range node.Vector.Values {
			if newNode.Vector.Values[i] != val {
				t.Errorf("Vector %s value at index %d is %f, expected %f", id, i, newNode.Vector.Values[i], val)
			}
		}
	}

	// Check that config parameters were preserved
	if newIndex.config.M != config.M {
		t.Errorf("Expected M to be %d, got %d", config.M, newIndex.config.M)
	}
	if newIndex.config.EfConstruction != config.EfConstruction {
		t.Errorf("Expected EfConstruction to be %d, got %d", config.EfConstruction, newIndex.config.EfConstruction)
	}
	if newIndex.config.EfSearch != config.EfSearch {
		t.Errorf("Expected EfSearch to be %d, got %d", config.EfSearch, newIndex.config.EfSearch)
	}

	// Check that entry point was preserved
	if newIndex.entryPoint != originalIndex.entryPoint {
		t.Errorf("Expected entry point to be %s, got %s", originalIndex.entryPoint, newIndex.entryPoint)
	}
}

func TestSetMetric(t *testing.T) {
	// Create an index with one metric
	idx := NewHNSWIndex(&distance.EuclideanDistance{}, nil)

	// Add vectors
	vectors := []*vector.Vector{
		vector.NewVector("v1", []float32{1.0, 0.0, 0.0}),
		vector.NewVector("v2", []float32{0.0, 1.0, 0.0}),
		vector.NewVector("v3", []float32{0.0, 0.0, 1.0}),
	}

	for _, v := range vectors {
		err := idx.Add(v)
		if err != nil {
			t.Fatalf("Add failed: %v", err)
		}
	}

	// Create a query vector
	query := vector.NewVector("query", []float32{0.9, 0.1, 0.1})

	// Get results with Euclidean distance
	euclideanResults, err := idx.Search(query, 3)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Change to Cosine distance
	idx.SetMetric(&distance.CosineDistance{})

	// Get results with Cosine distance
	cosineResults, err := idx.Search(query, 3)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// The results should be different because the metrics are different
	if len(euclideanResults) > 0 && len(cosineResults) > 0 {
		if euclideanResults[0].ID == cosineResults[0].ID && euclideanResults[0].Distance == cosineResults[0].Distance {
			t.Errorf("Expected different top results with different metrics")
		}
	}
}

func TestEmptyIndex(t *testing.T) {
	// Create an empty index
	idx := NewHNSWIndex(&distance.EuclideanDistance{}, nil)

	// Check size
	if idx.Size() != 0 {
		t.Errorf("Expected size to be 0, got %d", idx.Size())
	}

	// Check get IDs
	ids := idx.GetIDs()
	if len(ids) != 0 {
		t.Errorf("Expected empty ID list, got %d IDs", len(ids))
	}

	// Search should fail
	query := vector.NewVector("query", []float32{1.0, 2.0, 3.0})
	_, err := idx.Search(query, 1)
	if err != ErrNoVectors {
		t.Errorf("Expected ErrNoVectors, got %v", err)
	}
} 