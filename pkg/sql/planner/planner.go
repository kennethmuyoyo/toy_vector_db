package planner

import (
	"fmt"
	"strings"

	"github.com/ken/vector_database/pkg/sql/parser"
)

// PlanType represents the type of execution plan
type PlanType string

const (
	// PlanTypeFullScan represents a full scan of all vectors
	PlanTypeFullScan PlanType = "FULL_SCAN"

	// PlanTypeIDLookup represents a lookup by ID
	PlanTypeIDLookup PlanType = "ID_LOOKUP"

	// PlanTypeVectorSearch represents a nearest neighbor vector search
	PlanTypeVectorSearch PlanType = "VECTOR_SEARCH"
)

// PlanNode represents a node in the execution plan
type PlanNode struct {
	Type         PlanType
	Cost         float64
	Children     []*PlanNode
	TableName    string
	Condition    *parser.Node
	Projection   []string
	Limit        int
	VectorQuery  string
	DistanceFunc string
}

// QueryPlanner plans the execution of SQL queries
type QueryPlanner struct {}

// NewQueryPlanner creates a new query planner
func NewQueryPlanner() *QueryPlanner {
	return &QueryPlanner{}
}

// CreatePlan creates an execution plan for a SQL query
func (qp *QueryPlanner) CreatePlan(node *parser.Node) (*PlanNode, error) {
	switch node.Type {
	case parser.NodeSelect:
		return qp.createSelectPlan(node)
	case parser.NodeInsert:
		return &PlanNode{
			Type:      PlanTypeFullScan,
			Cost:      1.0,
			TableName: node.Children[0].Value,
		}, nil
	case parser.NodeDelete:
		return qp.createDeletePlan(node)
	case parser.NodeCreate:
		return &PlanNode{
			Type:      PlanTypeFullScan,
			Cost:      1.0,
			TableName: node.Children[0].Value,
		}, nil
	case parser.NodeDrop:
		return &PlanNode{
			Type:      PlanTypeFullScan,
			Cost:      1.0,
			TableName: node.Children[0].Value,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported node type: %v", node.Type)
	}
}

// createSelectPlan creates a plan for a SELECT query
func (qp *QueryPlanner) createSelectPlan(node *parser.Node) (*PlanNode, error) {
	// Find the FROM, WHERE, NEAREST TO, and LIMIT nodes
	var fromNode *parser.Node
	var whereNode *parser.Node
	var nearestNode *parser.Node
	var limitNode *parser.Node
	
	for _, child := range node.Children {
		switch child.Type {
		case parser.NodeFrom:
			fromNode = child
		case parser.NodeWhere:
			whereNode = child
		case parser.NodeNearestTo:
			nearestNode = child
		case parser.NodeLimit:
			limitNode = child
		}
	}
	
	if fromNode == nil || len(fromNode.Children) == 0 {
		return nil, fmt.Errorf("missing FROM clause or table name")
	}
	
	tableName := fromNode.Children[0].Value
	
	// Get projections (columns to return)
	projections := []string{}
	for _, child := range node.Children {
		if child.Type == parser.NodeColumn || child.Type == parser.NodeIdentifier {
			projections = append(projections, child.Value)
		}
	}
	
	// If no columns specified, select all columns
	if len(projections) == 0 {
		projections = append(projections, "*")
	}
	
	// Get limit if present
	limit := -1
	if limitNode != nil {
		fmt.Sscanf(limitNode.Value, "%d", &limit)
	}
	
	// Check if this is a vector search (NEAREST TO clause)
	if nearestNode != nil {
		vectorQuery := ""
		distanceFunc := "euclidean" // Default distance function
		
		// Extract vector query
		if len(nearestNode.Children) > 0 {
			vectorNode := nearestNode.Children[0]
			vectorQuery = vectorNode.Value
		}
		
		// Extract distance function if specified
		if len(nearestNode.Children) > 1 && nearestNode.Children[1].Type == parser.NodeMetric {
			distanceFunc = strings.Trim(nearestNode.Children[1].Value, "'\"")
		}
		
		return &PlanNode{
			Type:         PlanTypeVectorSearch,
			Cost:         10.0, // Vector search is more expensive than simple lookups
			TableName:    tableName,
			Projection:   projections,
			Limit:        limit,
			VectorQuery:  vectorQuery,
			DistanceFunc: distanceFunc,
		}, nil
	}
	
	// Check if this is an ID lookup (WHERE id = 'something')
	if whereNode != nil && len(whereNode.Children) > 0 {
		whereExpr := whereNode.Children[0]
		if whereExpr.Type == parser.NodeBinaryOp && whereExpr.Value == "=" {
			if len(whereExpr.Children) >= 2 && 
			   whereExpr.Children[0].Type == parser.NodeIdentifier && 
			   strings.ToLower(whereExpr.Children[0].Value) == "id" &&
			   whereExpr.Children[1].Type == parser.NodeLiteral {
				// This is an ID lookup
				return &PlanNode{
					Type:       PlanTypeIDLookup,
					Cost:       1.0, // ID lookups are cheap
					TableName:  tableName,
					Condition:  whereExpr,
					Projection: projections,
					Limit:      limit,
				}, nil
			}
		}
	}
	
	// Otherwise, this is a full scan
	var condition *parser.Node
	if whereNode != nil && len(whereNode.Children) > 0 {
		condition = whereNode.Children[0]
	}
	
	return &PlanNode{
		Type:       PlanTypeFullScan,
		Cost:       100.0, // Full scans are expensive
		TableName:  tableName,
		Condition:  condition,
		Projection: projections,
		Limit:      limit,
	}, nil
}

// createDeletePlan creates a plan for a DELETE query
func (qp *QueryPlanner) createDeletePlan(node *parser.Node) (*PlanNode, error) {
	if len(node.Children) == 0 || node.Children[0].Type != parser.NodeTable {
		return nil, fmt.Errorf("missing table name")
	}
	
	tableName := node.Children[0].Value
	
	// Find WHERE clause
	var whereNode *parser.Node
	for _, child := range node.Children {
		if child.Type == parser.NodeWhere {
			whereNode = child
			break
		}
	}
	
	// If no WHERE clause, this will be a full scan delete
	if whereNode == nil || len(whereNode.Children) == 0 {
		return &PlanNode{
			Type:      PlanTypeFullScan,
			Cost:      100.0, // Full scans are expensive
			TableName: tableName,
		}, nil
	}
	
	// Check if this is an ID-based delete
	whereExpr := whereNode.Children[0]
	if whereExpr.Type == parser.NodeBinaryOp && whereExpr.Value == "=" {
		if len(whereExpr.Children) >= 2 && 
		   whereExpr.Children[0].Type == parser.NodeIdentifier && 
		   strings.ToLower(whereExpr.Children[0].Value) == "id" &&
		   whereExpr.Children[1].Type == parser.NodeLiteral {
			// This is an ID lookup
			return &PlanNode{
				Type:      PlanTypeIDLookup,
				Cost:      1.0, // ID lookups are cheap
				TableName: tableName,
				Condition: whereExpr,
			}, nil
		}
	}
	
	// Otherwise, this is a full scan with filter
	return &PlanNode{
		Type:      PlanTypeFullScan,
		Cost:      100.0, // Full scans are expensive
		TableName: tableName,
		Condition: whereExpr,
	}, nil
}

// OptimizePlan optimizes the execution plan
func (qp *QueryPlanner) OptimizePlan(plan *PlanNode) *PlanNode {
	// Currently, we don't do much optimization, but this is where we would
	// reorder operations, choose indexes, etc.
	
	// Make a copy of the plan to avoid modifying the original
	optimizedPlan := *plan
	
	// Some basic optimizations
	if optimizedPlan.Type == PlanTypeFullScan && optimizedPlan.Condition == nil && optimizedPlan.Limit > 0 {
		// If we're doing a full scan with no condition but with a limit,
		// we can reduce the cost estimate
		optimizedPlan.Cost = float64(optimizedPlan.Limit) * 1.0
	}
	
	return &optimizedPlan
}

// DisplayPlan returns a string representation of the plan
func (qp *QueryPlanner) DisplayPlan(plan *PlanNode) string {
	var sb strings.Builder
	
	qp.displayPlanNode(&sb, plan, 0)
	
	return sb.String()
}

// displayPlanNode recursively displays a plan node
func (qp *QueryPlanner) displayPlanNode(sb *strings.Builder, node *PlanNode, indent int) {
	// Add indentation
	for i := 0; i < indent; i++ {
		sb.WriteString("  ")
	}
	
	// Write node type and cost
	sb.WriteString(fmt.Sprintf("%s (cost=%.2f)\n", node.Type, node.Cost))
	
	// Add indentation for details
	for i := 0; i < indent+1; i++ {
		sb.WriteString("  ")
	}
	
	// Write details
	sb.WriteString(fmt.Sprintf("Table: %s\n", node.TableName))
	
	if len(node.Projection) > 0 {
		for i := 0; i < indent+1; i++ {
			sb.WriteString("  ")
		}
		sb.WriteString(fmt.Sprintf("Columns: %s\n", strings.Join(node.Projection, ", ")))
	}
	
	if node.Condition != nil {
		for i := 0; i < indent+1; i++ {
			sb.WriteString("  ")
		}
		sb.WriteString(fmt.Sprintf("Filter: %s\n", qp.displayCondition(node.Condition)))
	}
	
	if node.Limit > 0 {
		for i := 0; i < indent+1; i++ {
			sb.WriteString("  ")
		}
		sb.WriteString(fmt.Sprintf("Limit: %d\n", node.Limit))
	}
	
	if node.Type == PlanTypeVectorSearch {
		for i := 0; i < indent+1; i++ {
			sb.WriteString("  ")
		}
		sb.WriteString(fmt.Sprintf("Vector: %s\n", node.VectorQuery))
		
		for i := 0; i < indent+1; i++ {
			sb.WriteString("  ")
		}
		sb.WriteString(fmt.Sprintf("Distance: %s\n", node.DistanceFunc))
	}
	
	// Display children
	for _, child := range node.Children {
		qp.displayPlanNode(sb, child, indent+1)
	}
}

// displayCondition returns a string representation of a condition
func (qp *QueryPlanner) displayCondition(node *parser.Node) string {
	if node == nil {
		return "NULL"
	}
	
	switch node.Type {
	case parser.NodeBinaryOp:
		if len(node.Children) >= 2 {
			left := qp.displayCondition(node.Children[0])
			right := qp.displayCondition(node.Children[1])
			return fmt.Sprintf("(%s %s %s)", left, node.Value, right)
		}
		return node.Value
		
	case parser.NodeIdentifier:
		return node.Value
		
	case parser.NodeLiteral:
		return fmt.Sprintf("'%s'", node.Value)
		
	default:
		return node.Value
	}
} 