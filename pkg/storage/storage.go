package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/ken/vector_database/pkg/core/vector"
)

var (
	// ErrVectorNotFound is returned when a vector with the specified ID is not found
	ErrVectorNotFound = errors.New("vector not found")
	
	// ErrVectorAlreadyExists is returned when attempting to insert a vector with an ID that already exists
	ErrVectorAlreadyExists = errors.New("vector already exists")
)

// VectorStore defines the interface for vector storage operations
type VectorStore interface {
	// Insert adds a new vector to the store
	Insert(v *vector.Vector) error
	
	// Get retrieves a vector by ID
	Get(id string) (*vector.Vector, error)
	
	// Update updates an existing vector
	Update(v *vector.Vector) error
	
	// Delete removes a vector by ID
	Delete(id string) error
	
	// List returns all vector IDs
	List() ([]string, error)
	
	// Count returns the number of vectors in the store
	Count() (int, error)
	
	// Close closes the store
	Close() error
}

// MemoryStore is an in-memory implementation of VectorStore
type MemoryStore struct {
	mu      sync.RWMutex
	vectors map[string]*vector.Vector
}

// NewMemoryStore creates a new in-memory vector store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		vectors: make(map[string]*vector.Vector),
	}
}

func (s *MemoryStore) Insert(v *vector.Vector) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.vectors[v.ID]; exists {
		return ErrVectorAlreadyExists
	}

	// Store a copy to prevent modification of the original
	s.vectors[v.ID] = v.Copy()
	return nil
}

func (s *MemoryStore) Get(id string) (*vector.Vector, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	v, exists := s.vectors[id]
	if !exists {
		return nil, ErrVectorNotFound
	}

	// Return a copy to prevent modification of the stored vector
	return v.Copy(), nil
}

func (s *MemoryStore) Update(v *vector.Vector) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.vectors[v.ID]; !exists {
		return ErrVectorNotFound
	}

	s.vectors[v.ID] = v.Copy()
	return nil
}

func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.vectors[id]; !exists {
		return ErrVectorNotFound
	}

	delete(s.vectors, id)
	return nil
}

func (s *MemoryStore) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.vectors))
	for id := range s.vectors {
		ids = append(ids, id)
	}

	return ids, nil
}

func (s *MemoryStore) Count() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.vectors), nil
}

func (s *MemoryStore) Close() error {
	// Nothing to do for memory store
	return nil
}

// FileStore is a file-based implementation of VectorStore
type FileStore struct {
	baseDir   string
	memStore  *MemoryStore
	mu        sync.RWMutex
	isLoaded  bool
}

// NewFileStore creates a new file-based vector store
func NewFileStore(baseDir string) (*FileStore, error) {
	// Ensure the directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return &FileStore{
		baseDir:  baseDir,
		memStore: NewMemoryStore(),
		isLoaded: false,
	}, nil
}

// ensureLoaded loads all vectors from disk if not already loaded
func (s *FileStore) ensureLoaded() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isLoaded {
		return nil
	}

	// Read vector files from the data directory
	files, err := os.ReadDir(s.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".vec" {
			continue
		}

		// Read the vector file
		path := filepath.Join(s.baseDir, file.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read vector file %s: %w", path, err)
		}

		// Decode the vector
		v, err := vector.Decode(data)
		if err != nil {
			return fmt.Errorf("failed to decode vector from file %s: %w", path, err)
		}

		// Store in memory
		s.memStore.vectors[v.ID] = v
	}

	s.isLoaded = true
	return nil
}

func (s *FileStore) Insert(v *vector.Vector) error {
	if err := s.ensureLoaded(); err != nil {
		return err
	}

	// Insert into memory first
	if err := s.memStore.Insert(v); err != nil {
		return err
	}

	// Write to disk
	return s.saveVector(v)
}

func (s *FileStore) Get(id string) (*vector.Vector, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	return s.memStore.Get(id)
}

func (s *FileStore) Update(v *vector.Vector) error {
	if err := s.ensureLoaded(); err != nil {
		return err
	}

	// Update in memory
	if err := s.memStore.Update(v); err != nil {
		return err
	}

	// Update on disk
	return s.saveVector(v)
}

func (s *FileStore) Delete(id string) error {
	if err := s.ensureLoaded(); err != nil {
		return err
	}

	// Get the vector first to ensure it exists
	_, err := s.memStore.Get(id)
	if err != nil {
		return err
	}

	// Delete from memory
	if err := s.memStore.Delete(id); err != nil {
		return err
	}

	// Delete from disk
	path := filepath.Join(s.baseDir, id+".vec")
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete vector file: %w", err)
	}

	return nil
}

func (s *FileStore) List() ([]string, error) {
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}

	return s.memStore.List()
}

func (s *FileStore) Count() (int, error) {
	if err := s.ensureLoaded(); err != nil {
		return 0, err
	}

	return s.memStore.Count()
}

func (s *FileStore) Close() error {
	// Nothing special to do, as we write vectors to disk on every change
	return nil
}

// saveVector writes a vector to disk
func (s *FileStore) saveVector(v *vector.Vector) error {
	data := v.Encode()
	path := filepath.Join(s.baseDir, v.ID+".vec")
	
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write vector to file: %w", err)
	}
	
	return nil
} 