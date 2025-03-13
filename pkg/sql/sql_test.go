package sql_test

import (
	"strings"
	"testing"

	"github.com/ken/vector_database/pkg/core/distance"
	"github.com/ken/vector_database/pkg/core/vector"
	"github.com/ken/vector_database/pkg/sql/cli"
	"github.com/ken/vector_database/pkg/sql/executor"
	"github.com/ken/vector_database/pkg/sql/parser"
	"github.com/ken/vector_database/pkg/storage"
)

// TestSQLParser tests the SQL parser
func TestSQLParser(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		nodeType parser.NodeType
		wantErr  bool
	}{
		{
			name:     "Simple SELECT",
			query:    "SELECT id, dimension FROM vectors",
			nodeType: parser.NodeSelect,
			wantErr:  false,
		},
		{
			name:     "SELECT with LIMIT",
			query:    "SELECT id FROM vectors LIMIT 10",
			nodeType: parser.NodeSelect,
			wantErr:  false,
		},
		{
			name:     "SELECT with WHERE",
			query:    "SELECT id FROM vectors WHERE id = 'vec1'",
			nodeType: parser.NodeSelect,
			wantErr:  false,
		},
		{
			name:     "SELECT with NEAREST TO",
			query:    "SELECT id, distance FROM vectors NEAREST TO [1.0,2.0,3.0] LIMIT 5",
			nodeType: parser.NodeSelect,
			wantErr:  false,
		},
		{
			name:     "SELECT with NEAREST TO and USING",
			query:    "SELECT id, distance FROM vectors NEAREST TO [1.0,2.0,3.0] USING euclidean LIMIT 5",
			nodeType: parser.NodeSelect,
			wantErr:  false,
		},
		{
			name:     "INSERT",
			query:    "INSERT INTO vectors (id, vector) VALUES ('vec10', [1.0,2.0,3.0])",
			nodeType: parser.NodeInsert,
			wantErr:  false,
		},
		{
			name:     "DELETE",
			query:    "DELETE FROM vectors WHERE id = 'vec1'",
			nodeType: parser.NodeDelete,
			wantErr:  false,
		},
		{
			name:     "CREATE",
			query:    "CREATE COLLECTION vectors",
			nodeType: parser.NodeCreate,
			wantErr:  false,
		},
		{
			name:     "DROP",
			query:    "DROP COLLECTION vectors",
			nodeType: parser.NodeDrop,
			wantErr:  false,
		},
		{
			name:    "Invalid query",
			query:   "SELECT FROM WHERE",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ast, err := parser.Parse(tt.query)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse() error = nil, want error for query: %s", tt.query)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Parse() error = %v, want nil for query: %s", err, tt.query)
				return
			}
			
			if ast.Type != tt.nodeType {
				t.Errorf("Parse() node type = %v, want %v for query: %s", ast.Type, tt.nodeType, tt.query)
			}
		})
	}
}

// TestSQLExecution tests the SQL executor with a memory store
func TestSQLExecution(t *testing.T) {
	// Create a memory store for testing
	store := createTestStore()
	
	// Create a SQL service
	metric, _ := distance.GetMetric(distance.Euclidean)
	sqlService := cli.NewSQLService(store, executor.IndexTypeFlat, metric)
	
	// Test queries
	tests := []struct {
		name    string
		query   string
		want    string
		wantErr bool
	}{
		{
			name:    "SELECT all vectors",
			query:   "SELECT id, dimension FROM vectors",
			want:    "5 row(s) returned",
			wantErr: false,
		},
		{
			name:    "SELECT with LIMIT",
			query:   "SELECT id FROM vectors LIMIT 2",
			want:    "2 row(s) returned",
			wantErr: false,
		},
		{
			name:    "SELECT with WHERE",
			query:   "SELECT id FROM vectors WHERE id = 'vec1'",
			want:    "1 row(s) returned",
			wantErr: false,
		},
		{
			name:    "SELECT with NEAREST TO",
			query:   "SELECT id FROM vectors NEAREST TO [1.0, 0.0, 0.0] LIMIT 3",
			want:    "3 row(s) returned",
			wantErr: false,
		},
		{
			name:    "INSERT vector",
			query:   "INSERT INTO vectors (id, vector) VALUES ('vec6', [6.0,6.0,6.0])",
			want:    "Inserted 1 vector with ID",
			wantErr: false,
		},
		{
			name:    "DELETE vector",
			query:   "DELETE FROM vectors WHERE id = 'vec5'",
			want:    "Deleted",
			wantErr: false,
		},
		{
			name:    "COUNT query",
			query:   "SELECT COUNT(*) FROM vectors",
			want:    "row(s) returned",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := sqlService.Execute(tt.query)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Execute() error = nil, want error for query: %s", tt.query)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Execute() error = %v, want nil for query: %s", err, tt.query)
				return
			}
			
			if !strings.Contains(result, tt.want) {
				t.Errorf("Execute() result does not contain %q, result = %q for query: %s", tt.want, result, tt.query)
			}
		})
	}
}

// TestHNSWIndexSearch tests the SQL interface with HNSW index
func TestHNSWIndexSearch(t *testing.T) {
	// Create a memory store for testing
	store := createTestStore()
	
	// Create a SQL service with HNSW index
	metric, _ := distance.GetMetric(distance.Euclidean)
	sqlService := cli.NewSQLService(store, executor.IndexTypeHNSW, metric)
	
	// Test query
	query := "SELECT id, distance FROM vectors NEAREST TO [1.0, 0.0, 0.0] LIMIT 3"
	result, err := sqlService.Execute(query)
	
	if err != nil {
		t.Errorf("Execute() error = %v, want nil for query: %s", err, query)
		return
	}
	
	if !strings.Contains(result, "vec1") {
		t.Errorf("HNSW search did not find vec1 which should be closest to [1,0,0]. Result: %s", result)
	}
	
	if !strings.Contains(result, "3 row(s) returned") {
		t.Errorf("HNSW search did not return 3 results as requested. Result: %s", result)
	}
}

// createTestStore creates a test memory store with sample vectors
func createTestStore() storage.VectorStore {
	store := storage.NewMemoryStore()
	
	// Add some test vectors
	vectors := []*vector.Vector{
		vector.NewVector("vec1", []float32{1.0, 0.0, 0.0}),
		vector.NewVector("vec2", []float32{0.0, 1.0, 0.0}),
		vector.NewVector("vec3", []float32{0.0, 0.0, 1.0}),
		vector.NewVector("vec4", []float32{1.0, 1.0, 0.0}),
		vector.NewVector("vec5", []float32{0.0, 1.0, 1.0}),
	}
	
	for _, vec := range vectors {
		store.Insert(vec)
	}
	
	return store
} 