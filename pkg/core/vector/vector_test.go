package vector

import (
	"testing"
)

func TestNewVector(t *testing.T) {
	id := "test-vector"
	values := []float32{1.0, 2.0, 3.0, 4.0, 5.0}
	
	v := NewVector(id, values)
	
	if v.ID != id {
		t.Errorf("Expected ID %s, got %s", id, v.ID)
	}
	
	if v.Dimension != len(values) {
		t.Errorf("Expected dimension %d, got %d", len(values), v.Dimension)
	}
	
	for i, val := range v.Values {
		if val != values[i] {
			t.Errorf("Expected value at index %d to be %f, got %f", i, values[i], val)
		}
	}
}

func TestZero(t *testing.T) {
	dimension := 5
	v := Zero(dimension)
	
	if v.Dimension != dimension {
		t.Errorf("Expected dimension %d, got %d", dimension, v.Dimension)
	}
	
	for i, val := range v.Values {
		if val != 0.0 {
			t.Errorf("Expected value at index %d to be 0.0, got %f", i, val)
		}
	}
}

func TestRandom(t *testing.T) {
	id := "random-vector"
	dimension := 5
	v := Random(id, dimension)
	
	if v.ID != id {
		t.Errorf("Expected ID %s, got %s", id, v.ID)
	}
	
	if v.Dimension != dimension {
		t.Errorf("Expected dimension %d, got %d", dimension, v.Dimension)
	}
	
	// Check that values are in the range [0, 1)
	for i, val := range v.Values {
		if val < 0.0 || val >= 1.0 {
			t.Errorf("Expected value at index %d to be in range [0, 1), got %f", i, val)
		}
	}
}

func TestCopy(t *testing.T) {
	id := "test-vector"
	values := []float32{1.0, 2.0, 3.0, 4.0, 5.0}
	
	original := NewVector(id, values)
	copy := original.Copy()
	
	// Check that the copy has the same values
	if copy.ID != original.ID {
		t.Errorf("Expected ID %s, got %s", original.ID, copy.ID)
	}
	
	if copy.Dimension != original.Dimension {
		t.Errorf("Expected dimension %d, got %d", original.Dimension, copy.Dimension)
	}
	
	for i, val := range copy.Values {
		if val != original.Values[i] {
			t.Errorf("Expected value at index %d to be %f, got %f", i, original.Values[i], val)
		}
	}
	
	// Modify the original and check that the copy is unchanged
	original.Values[0] = 99.0
	
	if copy.Values[0] == original.Values[0] {
		t.Errorf("Copy should not be affected by changes to original")
	}
}

func TestEncodeDecode(t *testing.T) {
	id := "test-vector"
	values := []float32{1.0, 2.0, 3.0, 4.0, 5.0}
	
	original := NewVector(id, values)
	encoded := original.Encode()
	
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Failed to decode vector: %v", err)
	}
	
	// Check that the decoded vector matches the original
	if decoded.ID != original.ID {
		t.Errorf("Expected ID %s, got %s", original.ID, decoded.ID)
	}
	
	if decoded.Dimension != original.Dimension {
		t.Errorf("Expected dimension %d, got %d", original.Dimension, decoded.Dimension)
	}
	
	for i, val := range decoded.Values {
		if val != original.Values[i] {
			t.Errorf("Expected value at index %d to be %f, got %f", i, original.Values[i], val)
		}
	}
}

func TestNormalize(t *testing.T) {
	values := []float32{3.0, 4.0} // 3-4-5 triangle
	v := NewVector("test", values)
	
	v.Normalize()
	
	// Check that the vector is now a unit vector
	var sum float32
	for _, val := range v.Values {
		sum += val * val
	}
	
	// Allow for small floating-point errors
	if sum < 0.99 || sum > 1.01 {
		t.Errorf("Expected normalized vector to have magnitude 1.0, got %f", sum)
	}
	
	// Check specific values (3/5 and 4/5)
	expected := []float32{0.6, 0.8}
	for i, val := range v.Values {
		if val < expected[i]-0.01 || val > expected[i]+0.01 {
			t.Errorf("Expected value at index %d to be %f, got %f", i, expected[i], val)
		}
	}
} 