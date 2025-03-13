package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ken/vector_database/pkg/core/vector"
)

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	
	// Test initial state
	count, err := store.Count()
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected empty store, got count %d", count)
	}
	
	// Test Insert
	v1 := vector.NewVector("v1", []float32{1.0, 2.0, 3.0})
	if err := store.Insert(v1); err != nil {
		t.Fatalf("Failed to insert vector: %v", err)
	}
	
	// Test Count after insert
	count, err = store.Count()
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected count 1, got %d", count)
	}
	
	// Test Get
	v1Retrieved, err := store.Get("v1")
	if err != nil {
		t.Fatalf("Failed to get vector: %v", err)
	}
	
	if v1Retrieved.ID != v1.ID {
		t.Errorf("Expected ID %s, got %s", v1.ID, v1Retrieved.ID)
	}
	
	for i, val := range v1Retrieved.Values {
		if val != v1.Values[i] {
			t.Errorf("Expected value at index %d to be %f, got %f", i, v1.Values[i], val)
		}
	}
	
	// Test Get with non-existent ID
	_, err = store.Get("non-existent")
	if err != ErrVectorNotFound {
		t.Errorf("Expected ErrVectorNotFound, got %v", err)
	}
	
	// Test Update
	v1Updated := vector.NewVector("v1", []float32{4.0, 5.0, 6.0})
	if err := store.Update(v1Updated); err != nil {
		t.Fatalf("Failed to update vector: %v", err)
	}
	
	v1Retrieved, err = store.Get("v1")
	if err != nil {
		t.Fatalf("Failed to get vector: %v", err)
	}
	
	for i, val := range v1Retrieved.Values {
		if val != v1Updated.Values[i] {
			t.Errorf("Expected value at index %d to be %f, got %f", i, v1Updated.Values[i], val)
		}
	}
	
	// Test Update with non-existent ID
	vNonExistent := vector.NewVector("non-existent", []float32{1.0, 2.0, 3.0})
	err = store.Update(vNonExistent)
	if err != ErrVectorNotFound {
		t.Errorf("Expected ErrVectorNotFound, got %v", err)
	}
	
	// Test List
	v2 := vector.NewVector("v2", []float32{7.0, 8.0, 9.0})
	if err := store.Insert(v2); err != nil {
		t.Fatalf("Failed to insert vector: %v", err)
	}
	
	ids, err := store.List()
	if err != nil {
		t.Fatalf("Failed to list vectors: %v", err)
	}
	
	if len(ids) != 2 {
		t.Errorf("Expected 2 IDs, got %d", len(ids))
	}
	
	foundV1 := false
	foundV2 := false
	for _, id := range ids {
		if id == "v1" {
			foundV1 = true
		}
		if id == "v2" {
			foundV2 = true
		}
	}
	
	if !foundV1 || !foundV2 {
		t.Errorf("Expected to find IDs v1 and v2, got %v", ids)
	}
	
	// Test Delete
	if err := store.Delete("v1"); err != nil {
		t.Fatalf("Failed to delete vector: %v", err)
	}
	
	_, err = store.Get("v1")
	if err != ErrVectorNotFound {
		t.Errorf("Expected ErrVectorNotFound after delete, got %v", err)
	}
	
	// Test Delete with non-existent ID
	err = store.Delete("non-existent")
	if err != ErrVectorNotFound {
		t.Errorf("Expected ErrVectorNotFound, got %v", err)
	}
	
	// Test Close
	if err := store.Close(); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}
}

func TestFileStore(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "vectodb-test")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a new file store
	store, err := NewFileStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create file store: %v", err)
	}
	
	// Test initial state
	count, err := store.Count()
	if err != nil {
		t.Fatalf("Failed to get count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected empty store, got count %d", count)
	}
	
	// Test Insert
	v1 := vector.NewVector("v1", []float32{1.0, 2.0, 3.0})
	if err := store.Insert(v1); err != nil {
		t.Fatalf("Failed to insert vector: %v", err)
	}
	
	// Verify file was created
	_, err = os.Stat(filepath.Join(tempDir, "v1.vec"))
	if err != nil {
		t.Fatalf("Failed to stat vector file: %v", err)
	}
	
	// Test Get
	v1Retrieved, err := store.Get("v1")
	if err != nil {
		t.Fatalf("Failed to get vector: %v", err)
	}
	
	if v1Retrieved.ID != v1.ID {
		t.Errorf("Expected ID %s, got %s", v1.ID, v1Retrieved.ID)
	}
	
	for i, val := range v1Retrieved.Values {
		if val != v1.Values[i] {
			t.Errorf("Expected value at index %d to be %f, got %f", i, v1.Values[i], val)
		}
	}
	
	// Test Update
	v1Updated := vector.NewVector("v1", []float32{4.0, 5.0, 6.0})
	if err := store.Update(v1Updated); err != nil {
		t.Fatalf("Failed to update vector: %v", err)
	}
	
	v1Retrieved, err = store.Get("v1")
	if err != nil {
		t.Fatalf("Failed to get vector: %v", err)
	}
	
	for i, val := range v1Retrieved.Values {
		if val != v1Updated.Values[i] {
			t.Errorf("Expected value at index %d to be %f, got %f", i, v1Updated.Values[i], val)
		}
	}
	
	// Test Delete
	if err := store.Delete("v1"); err != nil {
		t.Fatalf("Failed to delete vector: %v", err)
	}
	
	// Verify file was deleted
	_, err = os.Stat(filepath.Join(tempDir, "v1.vec"))
	if !os.IsNotExist(err) {
		t.Errorf("Expected vector file to be deleted")
	}
	
	// Test Close
	if err := store.Close(); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}
	
	// Test persistence by creating a new store instance
	store2, err := NewFileStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create second file store: %v", err)
	}
	
	// Insert a vector
	v2 := vector.NewVector("v2", []float32{7.0, 8.0, 9.0})
	if err := store2.Insert(v2); err != nil {
		t.Fatalf("Failed to insert vector: %v", err)
	}
	
	// Close the store
	if err := store2.Close(); err != nil {
		t.Fatalf("Failed to close store: %v", err)
	}
	
	// Create a third store instance to test loading from disk
	store3, err := NewFileStore(tempDir)
	if err != nil {
		t.Fatalf("Failed to create third file store: %v", err)
	}
	
	// Test that the vector is still there
	v2Retrieved, err := store3.Get("v2")
	if err != nil {
		t.Fatalf("Failed to get vector: %v", err)
	}
	
	if v2Retrieved.ID != v2.ID {
		t.Errorf("Expected ID %s, got %s", v2.ID, v2Retrieved.ID)
	}
	
	for i, val := range v2Retrieved.Values {
		if val != v2.Values[i] {
			t.Errorf("Expected value at index %d to be %f, got %f", i, v2.Values[i], val)
		}
	}
} 