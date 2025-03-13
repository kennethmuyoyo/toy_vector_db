package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// NodeType represents the type of an AST node
type NodeType int

const (
	// Node types
	NodeSelect NodeType = iota
	NodeInsert
	NodeDelete
	NodeCreate
	NodeDrop
	NodeUpdate
	NodeNearestTo
	NodeFrom
	NodeWhere
	NodeLimit
	NodeColumn
	NodeAlias
	NodeTable
	NodeIdentifier
	NodeBinaryOp
	NodeLiteral
	NodeVector
	NodeMetric
)

// Node represents a node in the abstract syntax tree
type Node struct {
	Type     NodeType
	Value    string
	Children []*Node
}

// Parser converts tokens into an AST
type Parser struct {
	tokens  []Token
	current int
}

// NewParser creates a new parser
func NewParser(tokens []Token) *Parser {
	// Filter out comments and whitespace
	filteredTokens := make([]Token, 0, len(tokens))
	for _, t := range tokens {
		if t.Type != TokenComment && t.Type != TokenWhitespace {
			filteredTokens = append(filteredTokens, t)
		}
	}
	
	return &Parser{
		tokens:  filteredTokens,
		current: 0,
	}
}

// Parse parses the tokens into an AST
func (p *Parser) Parse() (*Node, error) {
	if len(p.tokens) == 0 {
		return nil, fmt.Errorf("no tokens to parse")
	}

	// Check the first token to determine statement type
	if p.peek().Type == TokenKeyword {
		switch strings.ToUpper(p.peek().Value) {
		case "SELECT":
			return p.parseSelect()
		case "INSERT":
			return p.parseInsert()
		case "DELETE":
			return p.parseDelete()
		case "CREATE":
			return p.parseCreate()
		case "DROP":
			return p.parseDrop()
		case "UPDATE":
			return p.parseUpdate()
		default:
			return nil, fmt.Errorf("unexpected keyword: %s", p.peek().Value)
		}
	}

	return nil, fmt.Errorf("unexpected token: %s", p.peek().Value)
}

// advance advances the current token
func (p *Parser) advance() Token {
	if p.current >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	token := p.tokens[p.current]
	p.current++
	return token
}

// peek returns the current token without advancing
func (p *Parser) peek() Token {
	if p.current >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.current]
}

// check checks if the current token is of the expected type
func (p *Parser) check(tokenType TokenType) bool {
	if p.current >= len(p.tokens) {
		return false
	}
	return p.tokens[p.current].Type == tokenType
}

// consume consumes the current token if it's of the expected type
func (p *Parser) consume(tokenType TokenType, errorMsg string) (Token, error) {
	if p.check(tokenType) {
		return p.advance(), nil
	}
	return Token{}, fmt.Errorf("%s, got %s", errorMsg, p.peek().Value)
}

// consumeKeyword consumes the current token if it's a keyword with the expected value
func (p *Parser) consumeKeyword(keyword string, errorMsg string) (Token, error) {
	if p.check(TokenKeyword) && strings.ToUpper(p.peek().Value) == strings.ToUpper(keyword) {
		return p.advance(), nil
	}
	return Token{}, fmt.Errorf("%s, got %s", errorMsg, p.peek().Value)
}

// parseSelect parses a SELECT statement
func (p *Parser) parseSelect() (*Node, error) {
	selectNode := &Node{Type: NodeSelect, Children: []*Node{}}

	// Consume SELECT
	_, err := p.consumeKeyword("SELECT", "expected SELECT")
	if err != nil {
		return nil, err
	}

	// Parse column list
	for {
		// Handle COUNT(*)
		if p.check(TokenKeyword) && strings.ToUpper(p.peek().Value) == "COUNT" {
			p.advance()
			if p.check(TokenPunctuation) && p.peek().Value == "(" {
				p.advance()
				if p.check(TokenOperator) && p.peek().Value == "*" {
					p.advance()
					if p.check(TokenPunctuation) && p.peek().Value == ")" {
						p.advance()
						countNode := &Node{Type: NodeColumn, Value: "COUNT(*)", Children: []*Node{}}
						selectNode.Children = append(selectNode.Children, countNode)
					} else {
						return nil, fmt.Errorf("expected ), got %s", p.peek().Value)
					}
				} else {
					return nil, fmt.Errorf("expected *, got %s", p.peek().Value)
				}
			} else {
				return nil, fmt.Errorf("expected (, got %s", p.peek().Value)
			}
		} else {
			// Normal column
			column, err := p.parseExpression()
			if err != nil {
				return nil, err
			}
			
			// Check for AS alias
			if p.check(TokenKeyword) && strings.ToUpper(p.peek().Value) == "AS" {
				p.advance()
				
				// Consume alias identifier
				alias, err := p.consume(TokenIdentifier, "expected identifier for alias")
				if err != nil {
					return nil, err
				}
				
				aliasNode := &Node{Type: NodeAlias, Value: alias.Value, Children: []*Node{column}}
				selectNode.Children = append(selectNode.Children, aliasNode)
			} else {
				selectNode.Children = append(selectNode.Children, column)
			}
		}
		
		// Check for comma
		if p.check(TokenPunctuation) && p.peek().Value == "," {
			p.advance()
		} else {
			break
		}
	}

	// Parse FROM clause
	if p.check(TokenKeyword) && strings.ToUpper(p.peek().Value) == "FROM" {
		p.advance()
		
		// Parse table name
		table, err := p.consume(TokenIdentifier, "expected table name")
		if err != nil {
			return nil, err
		}
		
		fromNode := &Node{Type: NodeFrom, Children: []*Node{
			{Type: NodeTable, Value: table.Value},
		}}
		selectNode.Children = append(selectNode.Children, fromNode)
	}

	// Parse NEAREST TO clause
	if p.check(TokenKeyword) && strings.ToUpper(p.peek().Value) == "NEAREST" {
		p.advance()
		
		// Consume TO
		_, err := p.consumeKeyword("TO", "expected TO after NEAREST")
		if err != nil {
			return nil, err
		}
		
		// Parse vector expression
		var vectorExpr *Node
		
		// Handle subquery in parentheses
		if p.check(TokenPunctuation) && p.peek().Value == "(" {
			p.advance()
			subquery, err := p.parseSelect()
			if err != nil {
				return nil, err
			}
			_, err = p.consume(TokenPunctuation, "expected )")
			if err != nil {
				return nil, err
			}
			vectorExpr = subquery
		} else {
			// Handle literal vector or identifier
			vectorExpr, err = p.parseExpression()
			if err != nil {
				return nil, err
			}
		}

		nearestNode := &Node{Type: NodeNearestTo, Children: []*Node{vectorExpr}}
		
		// Parse USING METRIC clause
		if p.check(TokenKeyword) && strings.ToUpper(p.peek().Value) == "USING" {
			p.advance()
			
			// Parse metric name (identifier or string)
			var metricNode *Node
			if p.check(TokenString) {
				// String literal metric
				metricToken, err := p.consume(TokenString, "expected metric name")
				if err != nil {
					return nil, err
				}
				metricNode = &Node{Type: NodeMetric, Value: metricToken.Value}
			} else {
				// Identifier metric
				metricToken, err := p.consume(TokenIdentifier, "expected metric name")
				if err != nil {
					return nil, err
				}
				metricNode = &Node{Type: NodeMetric, Value: metricToken.Value}
			}
			
			nearestNode.Children = append(nearestNode.Children, metricNode)
		}
		
		selectNode.Children = append(selectNode.Children, nearestNode)
	}
	
	// Parse WHERE clause
	if p.check(TokenKeyword) && strings.ToUpper(p.peek().Value) == "WHERE" {
		p.advance()
		
		// Parse condition
		condition, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		
		whereNode := &Node{Type: NodeWhere, Children: []*Node{condition}}
		selectNode.Children = append(selectNode.Children, whereNode)
	}

	// Parse LIMIT clause
	if p.check(TokenKeyword) && strings.ToUpper(p.peek().Value) == "LIMIT" {
		p.advance()
		
		// Parse limit value
		limit, err := p.consume(TokenNumber, "expected number for LIMIT")
		if err != nil {
			return nil, err
		}
		
		limitNode := &Node{Type: NodeLimit, Value: limit.Value}
		selectNode.Children = append(selectNode.Children, limitNode)
	}
	
	// Consume optional semicolon
	if p.check(TokenPunctuation) && p.peek().Value == ";" {
		p.advance()
	}

	return selectNode, nil
}

// parseInsert parses an INSERT statement
func (p *Parser) parseInsert() (*Node, error) {
	insertNode := &Node{Type: NodeInsert, Children: []*Node{}}

	// Consume INSERT
	_, err := p.consumeKeyword("INSERT", "expected INSERT")
	if err != nil {
		return nil, err
	}
	
	// Consume INTO
	_, err = p.consumeKeyword("INTO", "expected INTO")
	if err != nil {
		return nil, err
	}
	
	// Parse table name
	table, err := p.consume(TokenIdentifier, "expected table name")
	if err != nil {
		return nil, err
	}
	
	tableNode := &Node{Type: NodeTable, Value: table.Value}
	insertNode.Children = append(insertNode.Children, tableNode)
	
	// Parse column list
	if p.check(TokenPunctuation) && p.peek().Value == "(" {
		p.advance()
		
		columnNodes := []*Node{}
		
		for {
			column, err := p.consume(TokenIdentifier, "expected column name")
			if err != nil {
				return nil, err
			}
			
			columnNode := &Node{Type: NodeColumn, Value: column.Value}
			columnNodes = append(columnNodes, columnNode)
			
			// Check for comma
			if p.check(TokenPunctuation) && p.peek().Value == "," {
				p.advance()
			} else {
				break
			}
		}
		
		_, err = p.consume(TokenPunctuation, "expected )")
		if err != nil {
			return nil, err
		}
		
		// Add all columns as a single node
		columnsNode := &Node{Type: NodeIdentifier, Value: "columns", Children: columnNodes}
		insertNode.Children = append(insertNode.Children, columnsNode)
	}
	
	// Consume VALUES
	_, err = p.consumeKeyword("VALUES", "expected VALUES")
	if err != nil {
		return nil, err
	}
	
	// Parse values list
	_, err = p.consume(TokenPunctuation, "expected (")
	if err != nil {
		return nil, err
	}
	
	valueNodes := []*Node{}
	
	for {
		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		
		valueNodes = append(valueNodes, value)
		
		// Check for comma
		if p.check(TokenPunctuation) && p.peek().Value == "," {
			p.advance()
		} else {
			break
		}
	}
	
	_, err = p.consume(TokenPunctuation, "expected )")
	if err != nil {
		return nil, err
	}
	
	// Add all values as a single node
	valuesNode := &Node{Type: NodeIdentifier, Value: "values", Children: valueNodes}
	insertNode.Children = append(insertNode.Children, valuesNode)
	
	// Consume optional semicolon
	if p.check(TokenPunctuation) && p.peek().Value == ";" {
		p.advance()
	}

	return insertNode, nil
}

// parseDelete parses a DELETE statement
func (p *Parser) parseDelete() (*Node, error) {
	deleteNode := &Node{Type: NodeDelete, Children: []*Node{}}

	// Consume DELETE
	_, err := p.consumeKeyword("DELETE", "expected DELETE")
	if err != nil {
		return nil, err
	}
	
	// Consume FROM
	_, err = p.consumeKeyword("FROM", "expected FROM")
	if err != nil {
		return nil, err
	}
	
	// Parse table name
	table, err := p.consume(TokenIdentifier, "expected table name")
	if err != nil {
		return nil, err
	}
	
	tableNode := &Node{Type: NodeTable, Value: table.Value}
	deleteNode.Children = append(deleteNode.Children, tableNode)
	
	// Parse WHERE clause (optional)
	if p.check(TokenKeyword) && strings.ToUpper(p.peek().Value) == "WHERE" {
		p.advance()
		
		// Parse condition
		condition, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		
		whereNode := &Node{Type: NodeWhere, Children: []*Node{condition}}
		deleteNode.Children = append(deleteNode.Children, whereNode)
	}
	
	// Consume optional semicolon
	if p.check(TokenPunctuation) && p.peek().Value == ";" {
		p.advance()
	}

	return deleteNode, nil
}

// parseCreate parses a CREATE statement
func (p *Parser) parseCreate() (*Node, error) {
	createNode := &Node{Type: NodeCreate, Children: []*Node{}}

	// Consume CREATE
	_, err := p.consumeKeyword("CREATE", "expected CREATE")
	if err != nil {
		return nil, err
	}
	
	// Consume COLLECTION
	_, err = p.consumeKeyword("COLLECTION", "expected COLLECTION")
	if err != nil {
		return nil, err
	}
	
	// Parse collection name
	collection, err := p.consume(TokenIdentifier, "expected collection name")
	if err != nil {
		return nil, err
	}
	
	tableNode := &Node{Type: NodeTable, Value: collection.Value}
	createNode.Children = append(createNode.Children, tableNode)
	
	// Parse dimension specification
	if p.check(TokenPunctuation) && p.peek().Value == "(" {
		p.advance()
		
		// Consume dimension
		dimension, err := p.consume(TokenIdentifier, "expected dimension")
		if err != nil {
			return nil, err
		}
		
		if strings.ToUpper(dimension.Value) != "DIMENSION" {
			return nil, fmt.Errorf("expected DIMENSION, got %s", dimension.Value)
		}
		
		// Consume INT
		intType, err := p.consumeKeyword("INT", "expected INT")
		if err != nil {
			return nil, err
		}
		
		dimensionNode := &Node{Type: NodeIdentifier, Value: "dimension", Children: []*Node{
			{Type: NodeLiteral, Value: intType.Value},
		}}
		createNode.Children = append(createNode.Children, dimensionNode)
		
		_, err = p.consume(TokenPunctuation, "expected )")
		if err != nil {
			return nil, err
		}
	}
	
	// Consume optional semicolon
	if p.check(TokenPunctuation) && p.peek().Value == ";" {
		p.advance()
	}

	return createNode, nil
}

// parseDrop parses a DROP statement
func (p *Parser) parseDrop() (*Node, error) {
	dropNode := &Node{Type: NodeDrop, Children: []*Node{}}

	// Consume DROP
	_, err := p.consumeKeyword("DROP", "expected DROP")
	if err != nil {
		return nil, err
	}
	
	// Consume COLLECTION
	_, err = p.consumeKeyword("COLLECTION", "expected COLLECTION")
	if err != nil {
		return nil, err
	}
	
	// Parse collection name
	collection, err := p.consume(TokenIdentifier, "expected collection name")
	if err != nil {
		return nil, err
	}
	
	tableNode := &Node{Type: NodeTable, Value: collection.Value}
	dropNode.Children = append(dropNode.Children, tableNode)
	
	// Consume optional semicolon
	if p.check(TokenPunctuation) && p.peek().Value == ";" {
		p.advance()
	}

	return dropNode, nil
}

// parseUpdate parses an UPDATE statement
func (p *Parser) parseUpdate() (*Node, error) {
	updateNode := &Node{Type: NodeUpdate, Children: []*Node{}}

	// Consume UPDATE
	_, err := p.consumeKeyword("UPDATE", "expected UPDATE")
	if err != nil {
		return nil, err
	}
	
	// Parse table name
	table, err := p.consume(TokenIdentifier, "expected table name")
	if err != nil {
		return nil, err
	}
	
	tableNode := &Node{Type: NodeTable, Value: table.Value}
	updateNode.Children = append(updateNode.Children, tableNode)
	
	// Consume SET
	_, err = p.consumeKeyword("SET", "expected SET")
	if err != nil {
		return nil, err
	}
	
	// Parse assignments
	assignments := []*Node{}
	
	for {
		// Parse column name
		column, err := p.parseIdentifier()
		if err != nil {
			return nil, err
		}
		
		// Consume =
		_, err = p.consume(TokenOperator, "expected =")
		if err != nil {
			return nil, err
		}
		
		// Parse value
		value, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		
		// Create assignment node
		assignNode := &Node{Type: NodeBinaryOp, Value: "=", Children: []*Node{column, value}}
		assignments = append(assignments, assignNode)
		
		// Check for comma
		if p.check(TokenPunctuation) && p.peek().Value == "," {
			p.advance()
		} else {
			break
		}
	}
	
	// Add assignments as a single node
	assignmentsNode := &Node{Type: NodeIdentifier, Value: "assignments", Children: assignments}
	updateNode.Children = append(updateNode.Children, assignmentsNode)
	
	// Parse WHERE clause (optional)
	if p.check(TokenKeyword) && strings.ToUpper(p.peek().Value) == "WHERE" {
		p.advance()
		
		// Parse condition
		condition, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		
		whereNode := &Node{Type: NodeWhere, Children: []*Node{condition}}
		updateNode.Children = append(updateNode.Children, whereNode)
	}
	
	// Consume optional semicolon
	if p.check(TokenPunctuation) && p.peek().Value == ";" {
		p.advance()
	}

	return updateNode, nil
}

// parseExpression parses an expression
func (p *Parser) parseExpression() (*Node, error) {
	return p.parseLogicalOr()
}

// parseLogicalOr parses a logical OR expression
func (p *Parser) parseLogicalOr() (*Node, error) {
	left, err := p.parseLogicalAnd()
	if err != nil {
		return nil, err
	}
	
	for p.check(TokenKeyword) && strings.ToUpper(p.peek().Value) == "OR" {
		p.advance()
		
		right, err := p.parseLogicalAnd()
		if err != nil {
			return nil, err
		}
		
		left = &Node{Type: NodeBinaryOp, Value: "OR", Children: []*Node{left, right}}
	}
	
	return left, nil
}

// parseLogicalAnd parses a logical AND expression
func (p *Parser) parseLogicalAnd() (*Node, error) {
	left, err := p.parseEquality()
	if err != nil {
		return nil, err
	}
	
	for p.check(TokenKeyword) && strings.ToUpper(p.peek().Value) == "AND" {
		p.advance()
		
		right, err := p.parseEquality()
		if err != nil {
			return nil, err
		}
		
		left = &Node{Type: NodeBinaryOp, Value: "AND", Children: []*Node{left, right}}
	}
	
	return left, nil
}

// parseEquality parses an equality expression
func (p *Parser) parseEquality() (*Node, error) {
	left, err := p.parseComparison()
	if err != nil {
		return nil, err
	}
	
	for p.check(TokenOperator) && (p.peek().Value == "=" || p.peek().Value == "!=" || p.peek().Value == "<>") {
		op := p.advance()
		
		right, err := p.parseComparison()
		if err != nil {
			return nil, err
		}
		
		left = &Node{Type: NodeBinaryOp, Value: op.Value, Children: []*Node{left, right}}
	}
	
	return left, nil
}

// parseComparison parses a comparison expression
func (p *Parser) parseComparison() (*Node, error) {
	left, err := p.parseTerm()
	if err != nil {
		return nil, err
	}
	
	for p.check(TokenOperator) && (p.peek().Value == "<" || p.peek().Value == "<=" || p.peek().Value == ">" || p.peek().Value == ">=") {
		op := p.advance()
		
		right, err := p.parseTerm()
		if err != nil {
			return nil, err
		}
		
		left = &Node{Type: NodeBinaryOp, Value: op.Value, Children: []*Node{left, right}}
	}
	
	return left, nil
}

// parseTerm parses a term expression
func (p *Parser) parseTerm() (*Node, error) {
	left, err := p.parseFactor()
	if err != nil {
		return nil, err
	}
	
	for p.check(TokenOperator) && (p.peek().Value == "+" || p.peek().Value == "-") {
		op := p.advance()
		
		right, err := p.parseFactor()
		if err != nil {
			return nil, err
		}
		
		left = &Node{Type: NodeBinaryOp, Value: op.Value, Children: []*Node{left, right}}
	}
	
	return left, nil
}

// parseFactor parses a factor expression
func (p *Parser) parseFactor() (*Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	
	for p.check(TokenOperator) && (p.peek().Value == "*" || p.peek().Value == "/" || p.peek().Value == "%") {
		op := p.advance()
		
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		
		left = &Node{Type: NodeBinaryOp, Value: op.Value, Children: []*Node{left, right}}
	}
	
	return left, nil
}

// parseUnary parses a unary expression
func (p *Parser) parseUnary() (*Node, error) {
	if p.check(TokenOperator) && (p.peek().Value == "-" || p.peek().Value == "+" || p.peek().Value == "!") {
		op := p.advance()
		
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		
		return &Node{Type: NodeBinaryOp, Value: op.Value, Children: []*Node{right}}, nil
	}
	
	return p.parsePrimary()
}

// parsePrimary parses a primary expression
func (p *Parser) parsePrimary() (*Node, error) {
	// Handle parentheses
	if p.check(TokenPunctuation) && p.peek().Value == "(" {
		p.advance()
		
		expr, err := p.parseExpression()
		if err != nil {
			return nil, err
		}
		
		_, err = p.consume(TokenPunctuation, "expected )")
		if err != nil {
			return nil, err
		}
		
		return expr, nil
	}
	
	// Handle literals
	if p.check(TokenNumber) {
		token := p.advance()
		
		// Convert to float or int
		if strings.Contains(token.Value, ".") {
			_, err := strconv.ParseFloat(token.Value, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", token.Value)
			}
			return &Node{Type: NodeLiteral, Value: token.Value}, nil
		} else {
			_, err := strconv.ParseInt(token.Value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number: %s", token.Value)
			}
			return &Node{Type: NodeLiteral, Value: token.Value}, nil
		}
	}
	
	// Handle string literals
	if p.check(TokenString) {
		token := p.advance()
		return &Node{Type: NodeLiteral, Value: token.Value}, nil
	}
	
	// Handle vector literals
	if p.check(TokenPunctuation) && p.peek().Value == "[" {
		p.advance() // Consume the opening bracket
		
		// Parse vector values
		values := []string{}
		
		// Handle empty vector
		if p.check(TokenPunctuation) && p.peek().Value == "]" {
			p.advance() // Consume the closing bracket
			return &Node{Type: NodeVector, Value: "[]"}, nil
		}
		
		// Parse first value
		if p.check(TokenNumber) {
			values = append(values, p.advance().Value)
		} else {
			return nil, fmt.Errorf("expected number in vector, got %s", p.peek().Value)
		}
		
		// Parse remaining values
		for p.check(TokenPunctuation) && p.peek().Value == "," {
			p.advance() // Consume comma
			
			if p.check(TokenNumber) {
				values = append(values, p.advance().Value)
			} else {
				return nil, fmt.Errorf("expected number in vector, got %s", p.peek().Value)
			}
		}
		
		// Consume closing bracket
		if p.check(TokenPunctuation) && p.peek().Value == "]" {
			p.advance()
		} else {
			return nil, fmt.Errorf("expected ] to close vector, got %s", p.peek().Value)
		}
		
		// Construct vector string
		vectorStr := "[" + strings.Join(values, ",") + "]"
		return &Node{Type: NodeVector, Value: vectorStr}, nil
	}
	
	// Handle identifiers
	return p.parseIdentifier()
}

// parseIdentifier parses an identifier
func (p *Parser) parseIdentifier() (*Node, error) {
	if p.check(TokenIdentifier) {
		token := p.advance()
		return &Node{Type: NodeIdentifier, Value: token.Value}, nil
	}
	
	// Handle special "star" identifier
	if p.check(TokenOperator) && p.peek().Value == "*" {
		token := p.advance()
		return &Node{Type: NodeIdentifier, Value: token.Value}, nil
	}
	
	return nil, fmt.Errorf("expected identifier, got %s", p.peek().Value)
}

// Parse a SQL string into an AST
func Parse(sql string) (*Node, error) {
	// Tokenize the SQL
	tokenizer := NewTokenizer(sql)
	tokens, err := tokenizer.Tokenize()
	if err != nil {
		return nil, err
	}
	
	// Parse the tokens
	parser := NewParser(tokens)
	return parser.Parse()
} 