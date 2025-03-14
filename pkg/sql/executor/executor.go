package executor

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/ken/vector_database/pkg/core/distance"
	"github.com/ken/vector_database/pkg/core/vector"
	"github.com/ken/vector_database/pkg/index"
	"github.com/ken/vector_database/pkg/index/flat"
	"github.com/ken/vector_database/pkg/index/hnsw"
	"github.com/ken/vector_database/pkg/sql/parser"
	"github.com/ken/vector_database/pkg/storage"
)

var (
	// ErrUnsupportedOperation is returned when an unsupported operation is requested
	ErrUnsupportedOperation = errors.New("unsupported operation")

	// ErrInvalidQuery is returned when the query is invalid
	ErrInvalidQuery = errors.New("invalid query")

	// ErrInvalidArgument is returned when an argument is invalid
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrCollectionNotFound is returned when a collection is not found
	ErrCollectionNotFound = errors.New("collection not found")

	// ErrCollectionAlreadyExists is returned when a collection already exists
	ErrCollectionAlreadyExists = errors.New("collection already exists")
)

// IndexType represents the type of index to use
type IndexType string

const (
	// IndexTypeFlat represents a flat index
	IndexTypeFlat IndexType = "flat"

	// IndexTypeHNSW represents an HNSW index
	IndexTypeHNSW IndexType = "hnsw"
)

// QueryExecutor executes SQL queries
type QueryExecutor struct {
	store      storage.VectorStore
	indexType  IndexType
	metric     distance.Metric
}

// NewQueryExecutor creates a new query executor
func NewQueryExecutor(store storage.VectorStore, indexType IndexType, metric distance.Metric) *QueryExecutor {
	return &QueryExecutor{
		store:     store,
		indexType: indexType,
		metric:    metric,
	}
}

// Column represents a column in a result set
type Column struct {
	Name  string
	Type  string
}

// Row represents a row in a result set
type Row []interface{}

// ResultSet represents the result of a query
type ResultSet struct {
	Columns []Column
	Rows    []Row
}

// ExecuteQuery executes a SQL query
func (qe *QueryExecutor) ExecuteQuery(query string) (*ResultSet, error) {
	// Parse the query
	ast, err := parser.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Execute the query based on its type
	switch ast.Type {
	case parser.NodeSelect:
		return qe.executeSelect(ast)
	case parser.NodeInsert:
		return qe.executeInsert(ast)
	case parser.NodeDelete:
		return qe.executeDelete(ast)
	case parser.NodeCreate:
		return qe.executeCreate(ast)
	case parser.NodeDrop:
		return qe.executeDrop(ast)
	default:
		return nil, ErrUnsupportedOperation
	}
}

// executeSelect executes a SELECT query
func (qe *QueryExecutor) executeSelect(node *parser.Node) (*ResultSet, error) {
	// Find the FROM node
	var fromNode *parser.Node
	var nearestNode *parser.Node
	var whereNode *parser.Node
	var limitNode *parser.Node
	
	for _, child := range node.Children {
		switch child.Type {
		case parser.NodeFrom:
			fromNode = child
		case parser.NodeNearestTo:
			nearestNode = child
		case parser.NodeWhere:
			whereNode = child
		case parser.NodeLimit:
			limitNode = child
		}
	}
	
	// We need at least a FROM clause
	if fromNode == nil {
		return nil, fmt.Errorf("%w: missing FROM clause", ErrInvalidQuery)
	}
	
	// Get the collection name
	if len(fromNode.Children) == 0 || fromNode.Children[0].Type != parser.NodeTable {
		return nil, fmt.Errorf("%w: invalid FROM clause", ErrInvalidQuery)
	}
	
	collectionName := fromNode.Children[0].Value
	
	// Prepare result columns
	columns := []Column{}
	for _, child := range node.Children {
		if child.Type == parser.NodeColumn {
			columns = append(columns, Column{
				Name: child.Value,
				Type: "string",
			})
		} else if child.Type == parser.NodeIdentifier {
			columns = append(columns, Column{
				Name: child.Value,
				Type: "string",
			})
		} else if child.Type == parser.NodeAlias {
			columns = append(columns, Column{
				Name: child.Value,
				Type: "string",
			})
		}
	}
	
	// Handle COUNT(*) special case
	isCountQuery := false
	for _, child := range node.Children {
		if child.Type == parser.NodeColumn && child.Value == "COUNT(*)" {
			isCountQuery = true
			columns = []Column{{Name: "COUNT(*)", Type: "int"}}
			break
		}
	}
	
	// Get the limit (default to all results)
	limit := -1
	if limitNode != nil {
		limitVal, err := strconv.Atoi(limitNode.Value)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid LIMIT value", ErrInvalidQuery)
		}
		limit = limitVal
	}
	
	// Handle nearest neighbor search
	if nearestNode != nil {
		return qe.executeNearestSearch(nearestNode, collectionName, columns, limit)
	}
	
	// Handle normal select
	// Get all vectors from the store
	ids, err := qe.store.List()
	if err != nil {
		return nil, err
	}
	
	// Apply WHERE filter if present
	if whereNode != nil {
		filteredIDs := []string{}
		for _, id := range ids {
			vec, err := qe.store.Get(id)
			if err != nil {
				// Skip vectors that can't be retrieved
				continue
			}
			
			matches, err := qe.evaluateWhereCondition(whereNode.Children[0], vec, collectionName)
			if err != nil {
				return nil, err
			}
			
			if matches {
				filteredIDs = append(filteredIDs, id)
			}
		}
		ids = filteredIDs
	}
	
	// Apply limit if needed
	if limit > 0 && limit < len(ids) {
		ids = ids[:limit]
	}
	
	// Create result set
	rows := []Row{}
	
	if isCountQuery {
		// For COUNT(*), just return the count
		rows = append(rows, Row{len(ids)})
	} else {
		// Otherwise, return the requested columns
		for _, id := range ids {
			vec, err := qe.store.Get(id)
			if err != nil {
				continue
			}
			
			row := Row{}
			for _, col := range columns {
				if col.Name == "id" {
					row = append(row, id)
				} else if col.Name == "vector" {
					row = append(row, fmt.Sprintf("%v", vec.Values))
				} else if col.Name == "dimension" {
					row = append(row, vec.Dimension)
				} else {
					// By default, return the ID
					row = append(row, id)
				}
			}
			rows = append(rows, row)
		}
	}
	
	return &ResultSet{Columns: columns, Rows: rows}, nil
}

// executeNearestSearch executes a nearest neighbor search
func (qe *QueryExecutor) executeNearestSearch(nearestNode *parser.Node, collectionName string, columns []Column, limit int) (*ResultSet, error) {
	// Get the query vector
	if len(nearestNode.Children) == 0 {
		return nil, fmt.Errorf("%w: missing query vector", ErrInvalidQuery)
	}
	
	queryNode := nearestNode.Children[0]
	var queryVec *vector.Vector
	
	if queryNode.Type == parser.NodeIdentifier {
		// Get the vector from the store
		vecID := queryNode.Value
		vec, err := qe.store.Get(vecID)
		if err != nil {
			return nil, fmt.Errorf("failed to get query vector: %w", err)
		}
		queryVec = vec
	} else if queryNode.Type == parser.NodeVector || queryNode.Type == parser.NodeLiteral {
		// Parse the vector literal
		vecStr := queryNode.Value
		vecStr = strings.Trim(vecStr, "[]")
		parts := strings.Split(vecStr, ",")
		values := make([]float32, 0, len(parts))
		
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue // Skip empty parts
			}
			val, err := strconv.ParseFloat(part, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid vector value: %s", part)
			}
			values = append(values, float32(val))
		}
		
		queryVec = vector.NewVector("query", values)
	} else {
		return nil, fmt.Errorf("%w: invalid query vector", ErrInvalidQuery)
	}
	
	// Get the metric to use
	metric := qe.metric
	if len(nearestNode.Children) > 1 && nearestNode.Children[1].Type == parser.NodeMetric {
		metricName := nearestNode.Children[1].Value
		// Remove quotes if present
		metricName = strings.Trim(metricName, "'\"")
		
		newMetric, err := distance.GetMetric(distance.MetricType(metricName))
		if err != nil {
			return nil, fmt.Errorf("invalid metric: %w", err)
		}
		metric = newMetric
	}
	
	// Set default limit if not specified
	if limit < 0 {
		limit = 10 // Default to 10 results
	}
	
	// Get all vectors from the store
	ids, err := qe.store.List()
	if err != nil {
		return nil, err
	}
	
	vectors := make([]*vector.Vector, 0, len(ids))
	for _, id := range ids {
		vec, err := qe.store.Get(id)
		if err != nil {
			continue
		}
		vectors = append(vectors, vec)
	}
	
	// Create and build the index
	var idx index.Index
	switch qe.indexType {
	case IndexTypeFlat:
		idx = flat.NewFlatIndex(metric)
	case IndexTypeHNSW:
		idx = hnsw.NewHNSWIndex(metric, nil)
	default:
		return nil, fmt.Errorf("unsupported index type: %s", qe.indexType)
	}
	
	if err := idx.Build(vectors); err != nil {
		return nil, fmt.Errorf("failed to build index: %w", err)
	}
	
	// Perform the search
	results, err := idx.Search(queryVec, limit)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	
	// Add "distance" column if not already present
	hasDistanceColumn := false
	for _, col := range columns {
		if col.Name == "distance" {
			hasDistanceColumn = true
			break
		}
	}
	
	if !hasDistanceColumn {
		columns = append(columns, Column{Name: "distance", Type: "float"})
	}
	
	// Create result set
	rows := []Row{}
	for _, result := range results {
		// Skip the query vector itself if it's in the results
		if result.ID == queryVec.ID {
			continue
		}
		
		row := Row{}
		for _, col := range columns {
			switch col.Name {
			case "id":
				row = append(row, result.ID)
			case "distance":
				row = append(row, result.Distance)
			case "vector":
				row = append(row, fmt.Sprintf("%v", result.Vector.Values))
			case "dimension":
				row = append(row, result.Vector.Dimension)
			default:
				// By default, return the ID
				row = append(row, result.ID)
			}
		}
		rows = append(rows, row)
	}
	
	return &ResultSet{Columns: columns, Rows: rows}, nil
}

// executeInsert executes an INSERT query
func (qe *QueryExecutor) executeInsert(node *parser.Node) (*ResultSet, error) {
	// Get the collection name
	if len(node.Children) == 0 || node.Children[0].Type != parser.NodeTable {
		return nil, fmt.Errorf("%w: missing collection name", ErrInvalidQuery)
	}
	
	// Get the columns and values
	var columnsNode *parser.Node
	var valuesNode *parser.Node
	
	for _, child := range node.Children {
		if child.Type == parser.NodeIdentifier && child.Value == "columns" {
			columnsNode = child
		} else if child.Type == parser.NodeIdentifier && child.Value == "values" {
			valuesNode = child
		}
	}
	
	if valuesNode == nil || len(valuesNode.Children) == 0 {
		return nil, fmt.Errorf("%w: missing values", ErrInvalidQuery)
	}
	
	// Extract column names
	columnNames := []string{}
	if columnsNode != nil {
		for _, child := range columnsNode.Children {
			columnNames = append(columnNames, child.Value)
		}
	}
	
	// Parse values
	values := make(map[string]interface{})
	for i, valueNode := range valuesNode.Children {
		var columnName string
		if i < len(columnNames) {
			columnName = columnNames[i]
		} else {
			// Assign default column names if not specified
			switch i {
			case 0:
				columnName = "id"
			case 1:
				columnName = "vector"
			default:
				columnName = fmt.Sprintf("col%d", i)
			}
		}
		
		switch valueNode.Type {
		case parser.NodeLiteral:
			values[columnName] = valueNode.Value
		case parser.NodeVector:
			vectorStr := valueNode.Value
			vectorStr = strings.Trim(vectorStr, "[]")
			parts := strings.Split(vectorStr, ",")
			vectorValues := make([]float32, 0, len(parts))
			
			for _, part := range parts {
				part = strings.TrimSpace(part)
				val, err := strconv.ParseFloat(part, 32)
				if err != nil {
					return nil, fmt.Errorf("invalid vector value: %s", part)
				}
				vectorValues = append(vectorValues, float32(val))
			}
			
			values[columnName] = vectorValues
		default:
			values[columnName] = valueNode.Value
		}
	}
	
	// Extract ID and vector values
	var id string
	var vectorValues []float32
	
	for key, value := range values {
		if strings.ToLower(key) == "id" {
			id = fmt.Sprintf("%v", value)
		} else if strings.ToLower(key) == "vector" {
			switch v := value.(type) {
			case []float32:
				vectorValues = v
			case string:
				// Parse vector from string
				v = strings.Trim(v, "[]")
				parts := strings.Split(v, ",")
				vectorValues = make([]float32, 0, len(parts))
				
				for _, part := range parts {
					part = strings.TrimSpace(part)
					val, err := strconv.ParseFloat(part, 32)
					if err != nil {
						return nil, fmt.Errorf("invalid vector value: %s", part)
					}
					vectorValues = append(vectorValues, float32(val))
				}
			}
		}
	}
	
	if id == "" {
		return nil, fmt.Errorf("%w: missing ID", ErrInvalidQuery)
	}
	
	if len(vectorValues) == 0 {
		return nil, fmt.Errorf("%w: missing vector values", ErrInvalidQuery)
	}
	
	// Create and store the vector
	vec := vector.NewVector(id, vectorValues)
	err := qe.store.Insert(vec)
	if err != nil {
		return nil, fmt.Errorf("failed to insert vector: %w", err)
	}
	
	// Create result set
	return &ResultSet{
		Columns: []Column{
			{Name: "result", Type: "string"},
		},
		Rows: []Row{
			{fmt.Sprintf("Inserted 1 vector with ID '%s'", id)},
		},
	}, nil
}

// executeDelete executes a DELETE query
func (qe *QueryExecutor) executeDelete(node *parser.Node) (*ResultSet, error) {
	// Get the collection name
	if len(node.Children) == 0 || node.Children[0].Type != parser.NodeTable {
		return nil, fmt.Errorf("%w: missing collection name", ErrInvalidQuery)
	}
	
	// Check for WHERE clause
	var whereNode *parser.Node
	for _, child := range node.Children {
		if child.Type == parser.NodeWhere {
			whereNode = child
			break
		}
	}
	
	// If no WHERE clause, error out (for safety)
	if whereNode == nil {
		return nil, fmt.Errorf("%w: DELETE requires a WHERE clause", ErrInvalidQuery)
	}
	
	// Get all vectors
	ids, err := qe.store.List()
	if err != nil {
		return nil, err
	}
	
	// Filter vectors based on WHERE clause
	deletedCount := 0
	for _, id := range ids {
		vec, err := qe.store.Get(id)
		if err != nil {
			continue
		}
		
		matches, err := qe.evaluateWhereCondition(whereNode.Children[0], vec, "")
		if err != nil {
			return nil, err
		}
		
		if matches {
			err = qe.store.Delete(id)
			if err != nil {
				continue
			}
			deletedCount++
		}
	}
	
	// Create result set
	return &ResultSet{
		Columns: []Column{
			{Name: "result", Type: "string"},
		},
		Rows: []Row{
			{fmt.Sprintf("Deleted %d vectors", deletedCount)},
		},
	}, nil
}

// executeCreate executes a CREATE COLLECTION query
func (qe *QueryExecutor) executeCreate(node *parser.Node) (*ResultSet, error) {
	// Get the collection name
	if len(node.Children) == 0 || node.Children[0].Type != parser.NodeTable {
		return nil, fmt.Errorf("%w: missing collection name", ErrInvalidQuery)
	}
	
	collectionName := node.Children[0].Value
	
	// Parse dimension if specified
	_ = 0 // Placeholder for future dimension validation
	for _, child := range node.Children {
		if child.Type == parser.NodeIdentifier && child.Value == "dimension" {
			if len(child.Children) > 0 && child.Children[0].Type == parser.NodeLiteral {
				_, err := strconv.Atoi(child.Children[0].Value)
				if err != nil {
					return nil, fmt.Errorf("invalid dimension: %w", err)
				}
				// Dimension will be used in future implementation
			}
		}
	}
	
	// For now, we don't actually create a collection since we have a single store
	// This would be implemented when we have a multi-collection architecture
	
	// Create result set
	return &ResultSet{
		Columns: []Column{
			{Name: "result", Type: "string"},
		},
		Rows: []Row{
			{fmt.Sprintf("Created collection '%s'", collectionName)},
		},
	}, nil
}

// executeDrop executes a DROP COLLECTION query
func (qe *QueryExecutor) executeDrop(node *parser.Node) (*ResultSet, error) {
	// Get the collection name
	if len(node.Children) == 0 || node.Children[0].Type != parser.NodeTable {
		return nil, fmt.Errorf("%w: missing collection name", ErrInvalidQuery)
	}
	
	collectionName := node.Children[0].Value
	
	// For now, dropping a collection would mean clearing all vectors
	// This would be implemented differently when we have a multi-collection architecture
	
	// Get all vectors
	ids, err := qe.store.List()
	if err != nil {
		return nil, err
	}
	
	// Delete all vectors
	deletedCount := 0
	for _, id := range ids {
		err = qe.store.Delete(id)
		if err != nil {
			continue
		}
		deletedCount++
	}
	
	// Create result set
	return &ResultSet{
		Columns: []Column{
			{Name: "result", Type: "string"},
		},
		Rows: []Row{
			{fmt.Sprintf("Dropped collection '%s' (%d vectors deleted)", collectionName, deletedCount)},
		},
	}, nil
}

// evaluateWhereCondition evaluates a WHERE condition for a vector
func (qe *QueryExecutor) evaluateWhereCondition(condNode *parser.Node, vec *vector.Vector, collectionName string) (bool, error) {
	switch condNode.Type {
	case parser.NodeBinaryOp:
		switch strings.ToUpper(condNode.Value) {
		case "AND":
			left, err := qe.evaluateWhereCondition(condNode.Children[0], vec, collectionName)
			if err != nil {
				return false, err
			}
			
			// Short-circuit evaluation
			if !left {
				return false, nil
			}
			
			return qe.evaluateWhereCondition(condNode.Children[1], vec, collectionName)
			
		case "OR":
			left, err := qe.evaluateWhereCondition(condNode.Children[0], vec, collectionName)
			if err != nil {
				return false, err
			}
			
			// Short-circuit evaluation
			if left {
				return true, nil
			}
			
			return qe.evaluateWhereCondition(condNode.Children[1], vec, collectionName)
			
		case "=":
			if condNode.Children[0].Type == parser.NodeIdentifier && strings.ToLower(condNode.Children[0].Value) == "id" {
				if condNode.Children[1].Type == parser.NodeLiteral {
					// Compare ID - remove quotes from string literals
					literalValue := strings.Trim(condNode.Children[1].Value, "'\"")
					return vec.ID == literalValue, nil
				}
			} else if condNode.Children[0].Type == parser.NodeIdentifier && strings.HasPrefix(strings.ToLower(condNode.Children[0].Value), "metadata.") {
				// Handle metadata field comparison
				metadataKey := strings.TrimPrefix(condNode.Children[0].Value, "metadata.")
				if condNode.Children[1].Type == parser.NodeLiteral {
					// Compare metadata value - remove quotes from string literals
					literalValue := strings.Trim(condNode.Children[1].Value, "'\"")
					actualValue, exists := vec.Metadata[metadataKey]
					return exists && actualValue == literalValue, nil
				}
			}
			
		case "!=", "<>":
			if condNode.Children[0].Type == parser.NodeIdentifier && strings.ToLower(condNode.Children[0].Value) == "id" {
				if condNode.Children[1].Type == parser.NodeLiteral {
					// Compare ID - remove quotes from string literals
					literalValue := strings.Trim(condNode.Children[1].Value, "'\"")
					return vec.ID != literalValue, nil
				}
			} else if condNode.Children[0].Type == parser.NodeIdentifier && strings.HasPrefix(strings.ToLower(condNode.Children[0].Value), "metadata.") {
				// Handle metadata field comparison
				metadataKey := strings.TrimPrefix(condNode.Children[0].Value, "metadata.")
				if condNode.Children[1].Type == parser.NodeLiteral {
					// Compare metadata value - remove quotes from string literals
					literalValue := strings.Trim(condNode.Children[1].Value, "'\"")
					actualValue, exists := vec.Metadata[metadataKey]
					return !exists || actualValue != literalValue, nil
				}
			}
		
		case "LIKE":
			// Support LIKE operator for pattern matching on vector IDs
			if condNode.Children[0].Type == parser.NodeIdentifier && strings.ToLower(condNode.Children[0].Value) == "id" {
				if condNode.Children[1].Type == parser.NodeLiteral {
					// Get the pattern value (remove quotes)
					pattern := strings.Trim(condNode.Children[1].Value, "'\"")
					
					// Convert SQL LIKE pattern to Go regex pattern
					regexPattern := convertLikeToRegex(pattern)
					
					// Compile and match the regex
					regex, err := regexp.Compile(regexPattern)
					if err != nil {
						return false, fmt.Errorf("invalid LIKE pattern: %w", err)
					}
					
					return regex.MatchString(vec.ID), nil
				}
			} else if condNode.Children[0].Type == parser.NodeIdentifier && strings.HasPrefix(strings.ToLower(condNode.Children[0].Value), "metadata.") {
				// Handle metadata field LIKE comparison
				metadataKey := strings.TrimPrefix(condNode.Children[0].Value, "metadata.")
				if condNode.Children[1].Type == parser.NodeLiteral {
					// Get the pattern value (remove quotes)
					pattern := strings.Trim(condNode.Children[1].Value, "'\"")
					
					// Convert SQL LIKE pattern to Go regex pattern
					regexPattern := convertLikeToRegex(pattern)
					
					// Compile and match the regex
					regex, err := regexp.Compile(regexPattern)
					if err != nil {
						return false, fmt.Errorf("invalid LIKE pattern: %w", err)
					}
					
					actualValue, exists := vec.Metadata[metadataKey]
					return exists && regex.MatchString(actualValue), nil
				}
			}
			return false, fmt.Errorf("LIKE operator currently only supports ID and metadata columns")
		}
		
		return false, fmt.Errorf("unsupported operator: %s", condNode.Value)
		
	default:
		return false, fmt.Errorf("unsupported node type in WHERE clause: %v", condNode.Type)
	}
}

// convertLikeToRegex converts a SQL LIKE pattern to a Go regex pattern
func convertLikeToRegex(pattern string) string {
	// Escape special regex characters
	pattern = regexp.QuoteMeta(pattern)
	
	// Replace SQL LIKE wildcards with regex wildcards
	// % matches any sequence of characters (including none)
	pattern = strings.ReplaceAll(pattern, "%", ".*")
	
	// _ matches any single character
	pattern = strings.ReplaceAll(pattern, "_", ".")
	
	// Add start and end anchors to match the whole string
	return "^" + pattern + "$"
} 