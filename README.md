# VectoDB - Vector Database in Go

Toy VectoDB is a vector database implemented in Go, designed for efficient similarity search and vector operations.

## Overview

This project implements a vector database from scratch in Go, providing:

- Efficient vector storage and retrieval
- Fast nearest-neighbor search using advanced indexing techniques
- Support for multiple distance metrics (Euclidean, Cosine, Dot Product, Manhattan)
- SQL-like query interface for familiar database interaction
- Metadata support for storing additional information with vectors
- Command-line interface for database management

## Project Structure

```
├── cmd/               # Application entry points
│   └── vectodb/       # Main executable
├── pkg/               # Public packages
│   ├── core/          # Core functionality
│   │   ├── vector/    # Vector operations
│   │   └── distance/  # Distance functions
│   ├── storage/       # Storage layer
│   ├── index/         # Indexing implementations
│   │   ├── flat/      # Flat (brute force) index
│   │   └── hnsw/      # Hierarchical Navigable Small World index
│   ├── sql/           # SQL interface
│   │   ├── parser/    # SQL parser
│   │   ├── planner/   # Query planner
│   │   ├── executor/  # Query executor
│   │   └── cli/       # CLI integration
│   ├── embedding/     # Embedding engine 
│   │   ├── models/    # Embedding models integration
│   │   └── pipeline/  # Processing pipelines for different content types
│   └── api/           # API interfaces (planned)
├── internal/          # Private packages
│   ├── config/        # Configuration
│   └── util/          # Utilities
└── tests/             # Integration tests
```

## Current Implementation Status

### Phase 1: Core Foundation (Completed)
- ✅ Project setup and configuration
- ✅ Vector operations and data structures
- ✅ Distance functions (Euclidean, Cosine, Dot Product, Manhattan)
- ✅ Basic file-based storage layer
- ✅ Command-line interface for basic operations

### Phase 2: Indexing (Completed)
- ✅ Flat index implementation (brute force approach)
- ✅ HNSW (Hierarchical Navigable Small World) index implementation
- ✅ K-NN search with different metrics
- ✅ Index persistence

### Phase 3: SQL Interface (Completed)
- ✅ SQL-like query language parser
- ✅ Query planning and optimization
- ✅ Query execution engine
- ✅ Integration with CLI
- ✅ Support for vector-specific operations (NEAREST TO clause)
- ✅ Pattern matching with LIKE operator
- ✅ Metadata filtering support

### Phase 4: Enhanced Features (Completed)
- ✅ Metadata support for vectors
- ✅ Consistent 384-dimension support (matching embedding model)
- ✅ Text embedding capabilities

### Next Steps
- Implement REST API for integration with other applications
- Create web interface for visualization and management
- Performance Testing
- Implement additional index types

## Getting Started

### Installation

```bash
# Clone the repository
git clone https://github.com/kennethmuyoyo/toy_vector_db.git
cd vectodb

# Build the project
go build -o vectodb ./cmd/vectodb
```

### Usage

#### Basic Vector Operations

```bash
# Create a random vector (now automatically using 384 dimensions)
./vectodb random my-vector

# Add a vector manually (must have 384 dimensions to match embedding model)
./vectodb add my-vector2 0.1,0.2,...,0.3

# Get a vector (now shows metadata if present)
./vectodb get my-vector

# List all vectors
./vectodb list

# Delete a vector
./vectodb delete my-vector

# Set metadata for a vector
./vectodb set-metadata my-vector category "image"
```

#### Search Operations

```bash
# Search for similar vectors with flat index (using Euclidean distance)
./vectodb search flat my-vector 5

# Search for similar vectors with HNSW index (using Euclidean distance)
./vectodb search hnsw my-vector 5

# Search with a different distance metric
./vectodb -metric=cosine search hnsw my-vector 5

# Search using text query (embedding generated automatically)
./vectodb search-text "what is vector database"
```

#### SQL Interface

VectoDB supports an SQL-like query language with enhanced capabilities:

```bash
# List all vectors
./vectodb sql "SELECT id, dimension FROM vectors"

# Get a specific vector by ID
./vectodb sql "SELECT id, vector FROM vectors WHERE id = 'my-vector'"

# Find vectors similar to a specified vector (vector search)
./vectodb sql "SELECT id, distance FROM vectors NEAREST TO [1.0,2.0,3.0,...] LIMIT 5"

# Change the distance metric for similarity search
./vectodb sql "SELECT id, distance FROM vectors NEAREST TO [1.0,2.0,3.0,...] USING cosine LIMIT 5"

# Use LIKE operator for pattern matching on vector IDs
./vectodb sql "SELECT id FROM vectors WHERE id LIKE 'test%'"

# Filter vectors by metadata
./vectodb sql "SELECT id FROM vectors WHERE metadata.category = 'image'"

# Use LIKE operator with metadata
./vectodb sql "SELECT id FROM vectors WHERE metadata.tags LIKE '%important%'"

# Add a new vector
./vectodb sql "INSERT INTO vectors (id, vector) VALUES ('vec123', [1.0,2.0,3.0,...])"

# Delete a vector
./vectodb sql "DELETE FROM vectors WHERE id = 'vec123'"

# Count vectors
./vectodb sql "SELECT COUNT(*) FROM vectors"
```

Options:
```bash
# Enable verbose output (shows query plan and execution time)
./vectodb -verbose sql "SELECT id FROM vectors LIMIT 5"

# Switch between index types
./vectodb -index=hnsw sql "SELECT id, distance FROM vectors NEAREST TO [1.0,2.0,3.0,...] LIMIT 5"
```

## Index Types

VectoDB currently supports two types of indices:

### Flat Index
- Brute-force approach that compares the query vector to all vectors in the database
- Provides exact nearest neighbor results
- Suitable for small datasets or when exact results are required
- Time complexity: O(n) where n is the number of vectors

### HNSW Index (Hierarchical Navigable Small World)
- Graph-based approximate nearest neighbor search algorithm
- Provides significantly faster search times than flat index, especially for large datasets
- Tunable parameters to balance between speed and accuracy
- Time complexity: O(log n) where n is the number of vectors
- Key parameters:
  - M: Maximum number of connections per node (default: 16)
  - efConstruction: Search list size during index construction (default: 200)
  - efSearch: Search list size during queries (default: 50)

## Distance Metrics

VectoDB supports the following distance metrics:

- **euclidean**: Euclidean distance (L2 norm)
- **cosine**: Cosine distance (1 - cosine similarity)
- **dotproduct**: Dot product distance (negative dot product)
- **manhattan**: Manhattan distance (L1 norm)

## SQL Query Language

VectoDB implements a SQL-like query language with extensions for vector operations:

### Supported SQL Commands

- **SELECT**: Retrieve vectors and their properties
  ```sql
  SELECT id, dimension FROM vectors [WHERE condition] [LIMIT n]
  ```

- **SELECT with NEAREST TO**: Perform similarity search
  ```sql
  SELECT id, distance FROM vectors NEAREST TO [vector] [USING metric] [LIMIT n]
  ```

- **INSERT**: Add a new vector
  ```sql
  INSERT INTO vectors (id, vector) VALUES ('id', [values])
  ```

- **DELETE**: Remove vectors
  ```sql
  DELETE FROM vectors WHERE condition
  ```

- **CREATE/DROP**: Create or drop collections
  ```sql
  CREATE COLLECTION vectors
  DROP COLLECTION vectors
  ```

### Special SQL Features

- **Vector Literals**: Vector data can be specified using square brackets
  ```sql
  [1.0, 2.0, 3.0, 4.0]
  ```

- **NEAREST TO Clause**: Extension for similarity search
  ```sql
  NEAREST TO [1.0, 2.0, 3.0]
  ```

- **USING Clause**: Specify distance metric
  ```sql
  USING euclidean|cosine|dotproduct|manhattan
  ```

- **LIKE Operator**: Pattern matching for IDs and metadata
  ```sql
  WHERE id LIKE 'pattern%'
  WHERE metadata.field LIKE '%pattern%'
  ```

- **Metadata Filtering**: Filter vectors based on metadata
  ```sql
  WHERE metadata.category = 'image'
  ```

## Vector Metadata

VectoDB now supports storing and querying metadata alongside vectors, making it more useful for real-world applications:

### Metadata Features

- **Associating Information**: Store additional information with vectors (categories, tags, timestamps, etc.)
- **Filtering**: Filter vectors based on metadata attributes
- **Pattern Matching**: Use the LIKE operator for pattern matching on metadata fields
- **Persistence**: Metadata is stored persistently alongside vector data

### Metadata Commands

```bash
# Set metadata for a vector
./vectodb set-metadata my-vector category "image"

# View metadata when retrieving a vector
./vectodb get my-vector

# Filter vectors based on metadata
./vectodb sql "SELECT id FROM vectors WHERE metadata.category = 'image'"
```

## Embedding Capabilities

VectoDB includes embedding functionality for text:

- **Text Embedding**: Generate vector embeddings from text using a pre-trained model
  ```bash
  ./vectodb embed text doc1 "This is a document to embed"
  ```

- **Text Search**: Search for similar text using semantic search
  ```bash
  ./vectodb search-text "find similar documents to this query"
  ```

## Planned Embedding Engine

The planned embedding engine will expand the current embedding capabilities:

### Key Features (Planned)

- **Enhanced Content Type Support**: Process text, JSON, images, audio, etc.
- **Additional Embedding Models**: Support for more embedding models
- **Pipeline Architecture**: Customizable processing pipelines for different content types
- **Improved Metadata Storage**: Enhanced metadata capabilities
- **Automatic Updates**: Keep embeddings in sync with content changes

### Extended SQL Interface (Planned)

The SQL interface will be extended further to support enhanced embedding operations:

```sql
-- Store content with automatic embedding generation
INSERT INTO documents (id, content, metadata) 
VALUES ('doc1', 'This is a document to embed', '{"category": "article"}')

-- Search using natural language
SELECT id, content, similarity FROM documents 
WHERE SIMILAR TO 'find documents about vector databases' 
LIMIT 5
```

## License

[MIT License](LICENSE) 