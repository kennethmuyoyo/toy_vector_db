# VectoDB - Vector Database in Go

VectoDB is a high-performance vector database implemented in Go, designed for efficient similarity search and vector operations.

## Overview

This project implements a vector database from scratch in Go, providing:

- Efficient vector storage and retrieval
- Fast nearest-neighbor search using advanced indexing techniques
- Support for multiple distance metrics (Euclidean, Cosine, Dot Product, Manhattan)
- REST API for integration with other applications
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
│   │   └── flat/      # Flat (brute force) index
│   └── api/           # API interfaces
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

### Phase 2: Indexing (In Progress)
- ✅ Flat index implementation (brute force approach)
- ✅ K-NN search with different metrics
- ✅ Index persistence
- ⏳ HNSW (Hierarchical Navigable Small World) index

### Next Steps
- Complete HNSW index implementation
- Add SQL query interface for familiar interaction
- Optimize for large-scale data

## Getting Started

### Installation

```bash
# Clone the repository
git clone https://github.com/your-username/vectodb.git
cd vectodb

# Build the project
go build -o vectodb ./cmd/vectodb
```

### Usage

```bash
# Create a random vector
./vectodb random my-vector 128

# Add a vector manually
./vectodb add my-vector2 1.0,2.0,3.0,4.0,5.0

# Get a vector
./vectodb get my-vector

# List all vectors
./vectodb list

# Delete a vector
./vectodb delete my-vector

# Search for similar vectors (using Euclidean distance)
./vectodb search my-vector 5

# Search with a different distance metric
./vectodb -metric=cosine search my-vector 5
```

## Distance Metrics

VectoDB supports the following distance metrics:

- **euclidean**: Euclidean distance (L2 norm)
- **cosine**: Cosine distance (1 - cosine similarity)
- **dotproduct**: Dot product distance (negative dot product)
- **manhattan**: Manhattan distance (L1 norm)

## License

[MIT License](LICENSE) 