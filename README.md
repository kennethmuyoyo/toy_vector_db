# VectoDB - Vector Database in Go

VectoDB is a high-performance vector database implemented in Go, designed for efficient similarity search and vector operations.

## Overview

This project implements a vector database from scratch in Go, providing:

- Efficient vector storage and retrieval
- Fast nearest-neighbor search using advanced indexing techniques
- Support for multiple distance metrics (Euclidean, Cosine, Dot Product)
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

### Next Steps

#### Phase 2: Indexing
- Flat index implementation
- HNSW (Hierarchical Navigable Small World) index
- Index persistence

#### Phase 3: Query Engine
- k-NN search with different metrics
- Metadata management
- Transaction support

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
```

## License

[MIT License](LICENSE) 