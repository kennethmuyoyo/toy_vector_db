# SQL Interface Design for VectoDB

## Overview

To provide a familiar interaction model for users, VectoDB will implement a SQL-like query interface that allows users to perform vector operations using SQL syntax. This will make it easier for users coming from traditional database backgrounds to work with vector data.

## SQL Commands

### Basic Operations

```sql
-- Create a collection (similar to a table)
CREATE COLLECTION vectors (dimension INT);

-- Drop a collection
DROP COLLECTION vectors;

-- Insert a vector
INSERT INTO vectors (id, vector) VALUES ('vec1', [1.0, 2.0, 3.0]);

-- Insert a vector with metadata
INSERT INTO vectors (id, vector, metadata) VALUES ('vec2', [4.0, 5.0, 6.0], '{"category": "product", "color": "red"}');

-- Delete a vector
DELETE FROM vectors WHERE id = 'vec1';

-- Get a vector
SELECT * FROM vectors WHERE id = 'vec1';

-- List all vectors
SELECT id FROM vectors;

-- Count vectors
SELECT COUNT(*) FROM vectors;
```

### Vector Search Operations

```sql
-- Find k nearest neighbors using default distance metric (Euclidean)
SELECT id, distance FROM vectors 
NEAREST TO (SELECT vector FROM vectors WHERE id = 'query-vec') 
LIMIT 5;

-- Find k nearest neighbors with a specific distance metric
SELECT id, distance FROM vectors 
NEAREST TO (SELECT vector FROM vectors WHERE id = 'query-vec') 
USING METRIC 'cosine'
LIMIT 5;

-- Find vectors within a distance threshold
SELECT id, distance FROM vectors 
NEAREST TO [1.0, 2.0, 3.0]
WHERE distance < 0.5
USING METRIC 'euclidean';

-- Vector search with metadata filtering
SELECT id, distance FROM vectors 
NEAREST TO (SELECT vector FROM vectors WHERE id = 'query-vec')
WHERE metadata.category = 'product'
LIMIT 10;
```

### Metadata Operations

```sql
-- Filter by metadata
SELECT id FROM vectors 
WHERE metadata.category = 'product' AND metadata.color = 'red';

-- Update metadata
UPDATE vectors 
SET metadata.verified = true 
WHERE id = 'vec1';
```

## Implementation Approach

1. **Parser**: Implement a SQL parser to translate SQL queries into internal operations
   - Use a parsing library (e.g., `participle` or `goyacc`)
   - Define grammar for vector-specific SQL extensions

2. **Query Planner**: Convert parsed queries into execution plans
   - Map SQL operations to vector database operations
   - Optimize search operations to use appropriate indexes

3. **Executor**: Execute the query plan and return results
   - Format results in a tabular format
   - Support multiple output formats (JSON, CSV, table)

4. **CLI Integration**: Add SQL mode to the CLI
   - Interactive SQL shell
   - Execute SQL scripts from files
   - Support for query history and auto-completion

## Metadata Management

To support metadata associated with vectors:
- Store metadata as JSON objects
- Index common metadata fields for efficient filtering
- Support nested JSON structures for complex metadata


## Future Extensions

- Support for JOIN operations between collections
- Vector operations in SQL (vector addition, dot product, etc.)
- User-defined functions for custom distance metrics
- Advanced filtering using vector properties (norm, dimensionality, etc.)
- Transaction support for atomic operations 