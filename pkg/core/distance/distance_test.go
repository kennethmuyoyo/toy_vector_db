package distance

import (
	"testing"

	"github.com/ken/vector_database/pkg/core/vector"
)

func TestEuclideanDistance(t *testing.T) {
	a := vector.NewVector("a", []float32{1.0, 2.0, 3.0})
	b := vector.NewVector("b", []float32{4.0, 5.0, 6.0})
	
	metric := &EuclideanDistance{}
	
	// Expected distance: sqrt((4-1)^2 + (5-2)^2 + (6-3)^2) = sqrt(27) = 5.196
	expected := float32(5.196)
	
	dist, err := metric.Distance(a, b)
	if err != nil {
		t.Fatalf("Failed to calculate distance: %v", err)
	}
	
	// Allow for small floating-point errors
	if dist < expected-0.01 || dist > expected+0.01 {
		t.Errorf("Expected distance to be %f, got %f", expected, dist)
	}
}

func TestCosineDistance(t *testing.T) {
	a := vector.NewVector("a", []float32{1.0, 0.0, 0.0})
	b := vector.NewVector("b", []float32{0.0, 1.0, 0.0})
	c := vector.NewVector("c", []float32{1.0, 1.0, 0.0})
	
	metric := &CosineDistance{}
	
	// Orthogonal vectors should have distance 1.0
	dist, err := metric.Distance(a, b)
	if err != nil {
		t.Fatalf("Failed to calculate distance: %v", err)
	}
	
	if dist < 0.99 || dist > 1.01 {
		t.Errorf("Expected distance between orthogonal vectors to be 1.0, got %f", dist)
	}
	
	// 45-degree angle should have distance 1 - cos(45°) = 1 - 1/sqrt(2) ≈ 0.293
	expected := float32(0.293)
	dist, err = metric.Distance(a, c)
	if err != nil {
		t.Fatalf("Failed to calculate distance: %v", err)
	}
	
	if dist < expected-0.01 || dist > expected+0.01 {
		t.Errorf("Expected distance to be %f, got %f", expected, dist)
	}
}

func TestDotProductDistance(t *testing.T) {
	a := vector.NewVector("a", []float32{1.0, 2.0, 3.0})
	b := vector.NewVector("b", []float32{4.0, 5.0, 6.0})
	
	metric := &DotProductDistance{}
	
	// Expected dot product: 1*4 + 2*5 + 3*6 = 4 + 10 + 18 = 32
	// Distance is negative dot product: -32
	expected := float32(-32.0)
	
	dist, err := metric.Distance(a, b)
	if err != nil {
		t.Fatalf("Failed to calculate distance: %v", err)
	}
	
	if dist != expected {
		t.Errorf("Expected distance to be %f, got %f", expected, dist)
	}
}

func TestManhattanDistance(t *testing.T) {
	a := vector.NewVector("a", []float32{1.0, 2.0, 3.0})
	b := vector.NewVector("b", []float32{4.0, 5.0, 6.0})
	
	metric := &ManhattanDistance{}
	
	// Expected distance: |4-1| + |5-2| + |6-3| = 3 + 3 + 3 = 9
	expected := float32(9.0)
	
	dist, err := metric.Distance(a, b)
	if err != nil {
		t.Fatalf("Failed to calculate distance: %v", err)
	}
	
	if dist != expected {
		t.Errorf("Expected distance to be %f, got %f", expected, dist)
	}
}

func TestGetMetric(t *testing.T) {
	tests := []struct {
		name     string
		metric   MetricType
		expected MetricType
		wantErr  bool
	}{
		{"Euclidean", Euclidean, Euclidean, false},
		{"Cosine", Cosine, Cosine, false},
		{"DotProduct", DotProduct, DotProduct, false},
		{"Manhattan", Manhattan, Manhattan, false},
		{"Unknown", "unknown", "", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric, err := GetMetric(tt.metric)
			
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for metric %s, got nil", tt.metric)
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Failed to get metric: %v", err)
			}
			
			if metric.Name() != tt.expected {
				t.Errorf("Expected metric %s, got %s", tt.expected, metric.Name())
			}
		})
	}
}

func TestInvalidDimension(t *testing.T) {
	a := vector.NewVector("a", []float32{1.0, 2.0, 3.0})
	b := vector.NewVector("b", []float32{4.0, 5.0})
	
	metrics := []Metric{
		&EuclideanDistance{},
		&CosineDistance{},
		&DotProductDistance{},
		&ManhattanDistance{},
	}
	
	for _, metric := range metrics {
		t.Run(string(metric.Name()), func(t *testing.T) {
			_, err := metric.Distance(a, b)
			
			if err != vector.ErrInvalidDimension {
				t.Errorf("Expected ErrInvalidDimension, got %v", err)
			}
		})
	}
} 