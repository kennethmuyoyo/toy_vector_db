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
	
	// Print result
	fmt.Println(result)
	
	return nil
} 