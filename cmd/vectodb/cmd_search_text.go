package main

import (
	"fmt"

	"github.com/ken/vector_database/pkg/core/distance"
	"github.com/ken/vector_database/pkg/embedding"
	"github.com/ken/vector_database/pkg/sql/cli"
	"github.com/ken/vector_database/pkg/sql/executor"
	"github.com/ken/vector_database/pkg/storage"
)

// HandleSearchTextCommand processes the search-text command
// This command embeds the provided text and searches for similar vectors
func HandleSearchTextCommand(queryText string, metric distance.Metric, indexType string, verbose bool) error {
	// Create embedding service
	service, err := embedding.NewService(nil)
	if err != nil {
		return fmt.Errorf("failed to create embedding service: %w", err)
	}
	defer service.Close()

	// Create a temporary document to get the embedding
	doc := embedding.NewTextDocument("_query_", queryText)
	if err := service.ProcessDocument(doc); err != nil {
		return fmt.Errorf("failed to embed query text: %w", err)
	}

	// Check vector dimensions
	if len(doc.Vector) == 0 {
		return fmt.Errorf("failed to generate vector embedding: empty vector")
	}
	
	if verbose {
		fmt.Printf("Generated embedding with dimension: %d\n", len(doc.Vector))
	}
	
	// Convert the vector to a string representation for the SQL query
	vectorStr := "["
	for i, val := range doc.Vector {
		if i > 0 {
			vectorStr += ", "
		}
		vectorStr += fmt.Sprintf("%f", val)
	}
	vectorStr += "]"

	// Construct SQL query
	sqlQuery := fmt.Sprintf("SELECT id, distance FROM vectors NEAREST TO %s USING %s LIMIT 10", 
		vectorStr, metric.Name())

	if verbose {
		fmt.Printf("Generated SQL query:\n%s\n\n", sqlQuery)
	}

	// Create storage
	store, err := storage.NewFileStore("data")
	if err != nil {
		return fmt.Errorf("failed to create storage: %w", err)
	}

	// Check if the database has any vectors
	count, err := store.Count()
	if err != nil {
		return fmt.Errorf("failed to count vectors: %w", err)
	}
	
	if count == 0 {
		return fmt.Errorf("no vectors found in the database")
	}
	
	// Get any vector from the database to check dimensions
	ids, err := store.List()
	if err != nil {
		return fmt.Errorf("failed to list vectors: %w", err)
	}
	
	if len(ids) > 0 {
		sampleVec, err := store.Get(ids[0])
		if err == nil && sampleVec.Dimension != len(doc.Vector) {
			return fmt.Errorf("dimension mismatch: query vector has dimension %d, but database vectors have dimension %d", 
				len(doc.Vector), sampleVec.Dimension)
		}
	}

	// Convert index type string to executor.IndexType
	var idxType executor.IndexType
	switch indexType {
	case "flat":
		idxType = executor.IndexTypeFlat
	case "hnsw":
		idxType = executor.IndexTypeHNSW
	default:
		return fmt.Errorf("unsupported index type: %s", indexType)
	}

	// Create SQL service
	sqlService := cli.NewSQLService(store, idxType, metric)
	sqlService.SetVerbose(verbose)
	
	// Execute SQL query
	result, err := sqlService.Execute(sqlQuery)
	if err != nil {
		return err
	}
	
	if result == "0 row(s) returned" && verbose {
		fmt.Println("No similar vectors found. This could be due to:")
		fmt.Println("1. No semantically similar vectors in the database")
		fmt.Println("2. Embedding model mismatch between stored vectors and query")
		fmt.Println("3. Threshold settings filtering out potential matches")
	}
	
	// Print result
	fmt.Println(result)
	
	return nil
} 