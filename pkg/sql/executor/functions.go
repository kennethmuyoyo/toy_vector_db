package executor

import (
	"fmt"
	"strings"
	
	"github.com/ken/vector_database/pkg/embedding"
)

// SqlFunction represents a SQL function that can be called in queries
type SqlFunction interface {
	// Name returns the function name (for registration)
	Name() string
	
	// Eval evaluates the function with the given arguments
	Eval(args []interface{}) (interface{}, error)
}

// CountFunction implements COUNT(*) aggregate function
type CountFunction struct{}

func (f *CountFunction) Name() string {
	return "COUNT"
}

func (f *CountFunction) Eval(args []interface{}) (interface{}, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("COUNT() requires 1 argument, got %d", len(args))
	}
	
	// For COUNT(*), we use a special case
	if args[0] == "*" {
		// Just return 1 for each row, will be summed by the executor
		return 1, nil
	}
	
	// For non-* arguments, count non-null values
	if args[0] != nil {
		return 1, nil
	}
	
	return 0, nil
}

// EmbeddingFunction implements EMBEDDING() function for text-to-vector conversion
type EmbeddingFunction struct {
	service *embedding.Service
}

func NewEmbeddingFunction() (*EmbeddingFunction, error) {
	service, err := embedding.NewService(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding service: %w", err)
	}
	
	return &EmbeddingFunction{
		service: service,
	}, nil
}

func (f *EmbeddingFunction) Name() string {
	return "EMBEDDING"
}

func (f *EmbeddingFunction) Eval(args []interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("EMBEDDING() requires at least 1 argument, got %d", len(args))
	}
	
	// First argument should be the text to embed
	text, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("EMBEDDING() first argument must be a string, got %T", args[0])
	}
	
	// Optional second argument can be model name (not implemented yet)
	
	// Create a document and embed it
	doc := embedding.NewTextDocument("_query_", text)
	if err := f.service.ProcessDocument(doc); err != nil {
		return nil, fmt.Errorf("failed to embed text: %w", err)
	}
	
	return doc.Vector, nil
}

func (f *EmbeddingFunction) Close() error {
	if f.service != nil {
		return f.service.Close()
	}
	return nil
}

// Function registry
var sqlFunctions = map[string]SqlFunction{
	"COUNT": &CountFunction{},
}

// RegisterFunction adds a function to the global registry
func RegisterFunction(function SqlFunction) {
	sqlFunctions[strings.ToUpper(function.Name())] = function
}

// GetFunction retrieves a registered function
func GetFunction(name string) (SqlFunction, bool) {
	function, ok := sqlFunctions[strings.ToUpper(name)]
	return function, ok
}

// Evaluate evaluates a function call with the given arguments
func EvaluateFunction(name string, args []interface{}) (interface{}, error) {
	function, ok := GetFunction(name)
	if !ok {
		return nil, fmt.Errorf("unknown function: %s", name)
	}
	
	return function.Eval(args)
} 