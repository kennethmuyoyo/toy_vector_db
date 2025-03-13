package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/ken/vector_database/pkg/core/distance"
	"github.com/ken/vector_database/pkg/sql/executor"
	"github.com/ken/vector_database/pkg/sql/parser"
	"github.com/ken/vector_database/pkg/sql/planner"
	"github.com/ken/vector_database/pkg/storage"
)

// SQLService provides a command-line interface for executing SQL queries
type SQLService struct {
	store      storage.VectorStore
	executor   *executor.QueryExecutor
	planner    *planner.QueryPlanner
	indexType  executor.IndexType
	metric     distance.Metric
	verbose    bool
}

// NewSQLService creates a new SQL service
func NewSQLService(store storage.VectorStore, indexType executor.IndexType, metric distance.Metric) *SQLService {
	return &SQLService{
		store:     store,
		executor:  executor.NewQueryExecutor(store, indexType, metric),
		planner:   planner.NewQueryPlanner(),
		indexType: indexType,
		metric:    metric,
		verbose:   false,
	}
}

// SetVerbose sets the verbose flag
func (s *SQLService) SetVerbose(verbose bool) {
	s.verbose = verbose
}

// SetIndexType sets the index type
func (s *SQLService) SetIndexType(indexType executor.IndexType) {
	s.indexType = indexType
	s.executor = executor.NewQueryExecutor(s.store, indexType, s.metric)
}

// SetMetric sets the distance metric
func (s *SQLService) SetMetric(metric distance.Metric) {
	s.metric = metric
	s.executor = executor.NewQueryExecutor(s.store, s.indexType, metric)
}

// Execute executes a SQL query and returns the formatted result
func (s *SQLService) Execute(query string) (string, error) {
	if s.verbose {
		fmt.Println("Query:", query)
	}

	// Start timing
	startTime := time.Now()

	// Parse the query
	ast, err := parser.Parse(query)
	if err != nil {
		return "", fmt.Errorf("parse error: %w", err)
	}

	// Create execution plan (for debugging)
	if s.verbose {
		plan, err := s.planner.CreatePlan(ast)
		if err != nil {
			fmt.Println("Error creating plan:", err)
		} else {
			fmt.Println("Execution Plan:")
			fmt.Println(s.planner.DisplayPlan(plan))
		}
	}

	// Execute the query
	result, err := s.executor.ExecuteQuery(query)
	if err != nil {
		return "", fmt.Errorf("execution error: %w", err)
	}

	// Format the result
	output := formatResult(result)

	// Calculate execution time
	executionTime := time.Since(startTime)
	
	if s.verbose {
		output += fmt.Sprintf("\nExecution time: %v\n", executionTime)
	}

	return output, nil
}

// formatResult formats a result set as a string table
func formatResult(result *executor.ResultSet) string {
	if result == nil || len(result.Columns) == 0 {
		return "No results."
	}

	// Calculate column widths
	colWidths := make([]int, len(result.Columns))
	for i, col := range result.Columns {
		colWidths[i] = len(col.Name)
		
		// Check row values for wider content
		for _, row := range result.Rows {
			if i < len(row) {
				valStr := fmt.Sprintf("%v", row[i])
				if len(valStr) > colWidths[i] {
					// Limit the width to avoid very long columns
					colWidths[i] = min(len(valStr), 50)
				}
			}
		}
	}

	var sb strings.Builder
	
	// Write header
	for i, col := range result.Columns {
		format := fmt.Sprintf("%%-%ds", colWidths[i])
		sb.WriteString(fmt.Sprintf(format, col.Name))
		if i < len(result.Columns)-1 {
			sb.WriteString(" | ")
		}
	}
	sb.WriteString("\n")
	
	// Write separator
	for i, width := range colWidths {
		sb.WriteString(strings.Repeat("-", width))
		if i < len(colWidths)-1 {
			sb.WriteString("-+-")
		}
	}
	sb.WriteString("\n")
	
	// Write rows
	for _, row := range result.Rows {
		for i := 0; i < len(result.Columns); i++ {
			if i < len(row) {
				val := row[i]
				valStr := fmt.Sprintf("%v", val)
				
				// Truncate long values
				if len(valStr) > colWidths[i] {
					valStr = valStr[:colWidths[i]-3] + "..."
				}
				
				format := fmt.Sprintf("%%-%ds", colWidths[i])
				sb.WriteString(fmt.Sprintf(format, valStr))
			} else {
				// Empty value for missing columns
				format := fmt.Sprintf("%%-%ds", colWidths[i])
				sb.WriteString(fmt.Sprintf(format, "NULL"))
			}
			
			if i < len(result.Columns)-1 {
				sb.WriteString(" | ")
			}
		}
		sb.WriteString("\n")
	}
	
	// Write row count
	sb.WriteString(fmt.Sprintf("\n%d row(s) returned\n", len(result.Rows)))
	
	return sb.String()
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
} 