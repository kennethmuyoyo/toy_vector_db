package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/ken/vector_database/pkg/core/vector"
	"github.com/ken/vector_database/pkg/embedding"
	"github.com/ken/vector_database/pkg/storage"
)

// HandleEmbedCommand processes the embed command
// Usage:
//   ./vectodb embed text <id> <text>
//   ./vectodb embed file <id> <file_path>
//   ./vectodb embed json <id> <json_string_or_file>
func HandleEmbedCommand(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: embed [text|file|json] <id> <content>")
	}

	embedType := args[0]
	id := args[1]
	contentArg := args[2]

	// Create embedding service
	service, err := embedding.NewService(nil)
	if err != nil {
		return fmt.Errorf("failed to create embedding service: %w", err)
	}
	defer service.Close()

	var doc *embedding.Document

	switch embedType {
	case "text":
		// Direct text embedding
		doc = embedding.NewTextDocument(id, contentArg)
	case "file":
		// Read from file
		content, err := ioutil.ReadFile(contentArg)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		doc = embedding.NewTextDocument(id, string(content))
	case "json":
		// Handle JSON content
		var jsonContent map[string]interface{}
		
		// Check if the argument is a file or JSON string
		if strings.HasPrefix(contentArg, "{") {
			// Parse as JSON string
			if err := json.Unmarshal([]byte(contentArg), &jsonContent); err != nil {
				return fmt.Errorf("failed to parse JSON: %w", err)
			}
		} else {
			// Try to read as a file
			content, err := ioutil.ReadFile(contentArg)
			if err != nil {
				return fmt.Errorf("failed to read JSON file: %w", err)
			}
			if err := json.Unmarshal(content, &jsonContent); err != nil {
				return fmt.Errorf("failed to parse JSON file: %w", err)
			}
		}
		
		doc = embedding.NewJSONDocument(id, jsonContent)
	default:
		return fmt.Errorf("unknown embed type: %s (use text, file, or json)", embedType)
	}

	// Process the document to generate embeddings
	if err := service.ProcessDocument(doc); err != nil {
		return fmt.Errorf("failed to process document: %w", err)
	}

	// Make sure we're using the specified ID, not any potential content-as-ID
	if doc.ID != id {
		fmt.Printf("Warning: Document ID (%s) was different from specified ID (%s). Using specified ID.\n", doc.ID, id)
		doc.ID = id
	}

	// Store the document and its vector
	store, err := storage.NewFileStore("data")
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	// Store as a vector - explicitly use the specified ID
	v := vector.NewVector(id, doc.Vector)
	if err := store.Insert(v); err != nil {
		return fmt.Errorf("failed to store vector: %w", err)
	}

	// Store document metadata as a JSON file in the same directory
	docJson, err := doc.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to convert document to JSON: %w", err)
	}

	// Get the data directory from the store
	dataDir := filepath.Dir(store.BaseDir())
	metadataPath := filepath.Join(dataDir, "docs", id+".json")
	os.MkdirAll(filepath.Join(dataDir, "docs"), 0755)
	
	if err := ioutil.WriteFile(metadataPath, []byte(docJson), 0644); err != nil {
		return fmt.Errorf("failed to write document metadata: %w", err)
	}

	fmt.Printf("Document '%s' embedded and stored successfully.\n", id)
	fmt.Printf("Vector dimension: %d\n", len(doc.Vector))
	fmt.Printf("Content type: %s\n", doc.ContentType)
	fmt.Printf("Metadata stored at: %s\n", metadataPath)

	return nil
} 