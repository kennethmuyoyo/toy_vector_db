package distance

import (
	"errors"
	"math"

	"github.com/ken/vector_database/pkg/core/vector"
)

// MetricType represents the type of distance metric
type MetricType string

const (
	// Euclidean distance metric
	Euclidean MetricType = "euclidean"
	
	// Cosine distance metric (1 - cosine similarity)
	Cosine MetricType = "cosine"
	
	// DotProduct distance metric (negative dot product; closer to -∞ means more similar)
	DotProduct MetricType = "dotproduct"
	
	// Manhattan distance metric (L1 norm)
	Manhattan MetricType = "manhattan"
)

// Metric is an interface for distance calculations
type Metric interface {
	// Distance calculates the distance between two vectors
	Distance(a, b *vector.Vector) (float32, error)
	
	// Name returns the name of the metric
	Name() MetricType
}

// GetMetric returns a distance metric implementation by name
func GetMetric(metric MetricType) (Metric, error) {
	switch metric {
	case Euclidean:
		return &EuclideanDistance{}, nil
	case Cosine:
		return &CosineDistance{}, nil
	case DotProduct:
		return &DotProductDistance{}, nil
	case Manhattan:
		return &ManhattanDistance{}, nil
	default:
		return nil, errors.New("unknown distance metric")
	}
}

// EuclideanDistance implements the Euclidean (L2) distance metric
type EuclideanDistance struct{}

func (d *EuclideanDistance) Distance(a, b *vector.Vector) (float32, error) {
	if a.Dimension != b.Dimension {
		return 0, vector.ErrInvalidDimension
	}

	var sum float64
	for i := 0; i < a.Dimension; i++ {
		diff := float64(a.Values[i] - b.Values[i])
		sum += diff * diff
	}

	return float32(math.Sqrt(sum)), nil
}

func (d *EuclideanDistance) Name() MetricType {
	return Euclidean
}

// CosineDistance implements the Cosine distance metric
type CosineDistance struct{}

func (d *CosineDistance) Distance(a, b *vector.Vector) (float32, error) {
	if a.Dimension != b.Dimension {
		return 0, vector.ErrInvalidDimension
	}

	var dotProduct, normA, normB float64
	for i := 0; i < a.Dimension; i++ {
		dotProduct += float64(a.Values[i] * b.Values[i])
		normA += float64(a.Values[i] * a.Values[i])
		normB += float64(b.Values[i] * b.Values[i])
	}

	// Handle zero vectors
	if normA == 0 || normB == 0 {
		return 1.0, nil // Maximum distance
	}

	similarity := dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
	
	// Clamp to [-1, 1] to handle floating-point errors
	if similarity > 1.0 {
		similarity = 1.0
	} else if similarity < -1.0 {
		similarity = -1.0
	}
	
	return float32(1.0 - similarity), nil
}

func (d *CosineDistance) Name() MetricType {
	return Cosine
}

// DotProductDistance implements the Dot Product distance metric
type DotProductDistance struct{}

func (d *DotProductDistance) Distance(a, b *vector.Vector) (float32, error) {
	if a.Dimension != b.Dimension {
		return 0, vector.ErrInvalidDimension
	}

	var dotProduct float64
	for i := 0; i < a.Dimension; i++ {
		dotProduct += float64(a.Values[i] * b.Values[i])
	}

	// We return negative dot product because larger dot product = more similar
	// and we want smaller distance = more similar
	return float32(-dotProduct), nil
}

func (d *DotProductDistance) Name() MetricType {
	return DotProduct
}

// ManhattanDistance implements the Manhattan (L1) distance metric
type ManhattanDistance struct{}

func (d *ManhattanDistance) Distance(a, b *vector.Vector) (float32, error) {
	if a.Dimension != b.Dimension {
		return 0, vector.ErrInvalidDimension
	}

	var sum float64
	for i := 0; i < a.Dimension; i++ {
		diff := math.Abs(float64(a.Values[i] - b.Values[i]))
		sum += diff
	}

	return float32(sum), nil
}

func (d *ManhattanDistance) Name() MetricType {
	return Manhattan
} 