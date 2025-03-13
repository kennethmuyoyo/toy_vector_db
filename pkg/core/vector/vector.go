package vector

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

var (
	// ErrInvalidDimension is returned when vector dimensions don't match
	ErrInvalidDimension = errors.New("invalid vector dimension")
)

// Vector represents a real-valued vector in n-dimensional space
type Vector struct {
	ID        string    // Unique identifier for the vector
	Values    []float32 // Vector components
	Dimension int       // Number of dimensions
}

// NewVector creates a new vector with the specified ID and values
func NewVector(id string, values []float32) *Vector {
	return &Vector{
		ID:        id,
		Values:    values,
		Dimension: len(values),
	}
}

// Zero creates a zero vector of the specified dimension
func Zero(dimension int) *Vector {
	values := make([]float32, dimension)
	return &Vector{
		ID:        "",
		Values:    values,
		Dimension: dimension,
	}
}

// Random creates a random vector with values uniformly distributed in [0, 1)
func Random(id string, dimension int) *Vector {
	// Initialize rand with time seed if not already initialized
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	
	values := make([]float32, dimension)
	for i := 0; i < dimension; i++ {
		values[i] = float32(r.Float64())
	}
	return &Vector{
		ID:        id,
		Values:    values,
		Dimension: dimension,
	}
}

// Copy creates a deep copy of the vector
func (v *Vector) Copy() *Vector {
	valuesCopy := make([]float32, v.Dimension)
	copy(valuesCopy, v.Values)
	return &Vector{
		ID:        v.ID,
		Values:    valuesCopy,
		Dimension: v.Dimension,
	}
}

// Encode serializes the vector to a byte slice
func (v *Vector) Encode() []byte {
	// Calculate buffer size: ID length (4 bytes) + ID + dimension (4 bytes) + values (4 bytes each)
	idBytes := []byte(v.ID)
	bufSize := 4 + len(idBytes) + 4 + 4*v.Dimension
	buf := make([]byte, bufSize)
	
	// Write ID length
	binary.LittleEndian.PutUint32(buf[0:], uint32(len(idBytes)))
	
	// Write ID
	copy(buf[4:], idBytes)
	
	// Write dimension
	binary.LittleEndian.PutUint32(buf[4+len(idBytes):], uint32(v.Dimension))
	
	// Write values
	for i, val := range v.Values {
		offset := 4 + len(idBytes) + 4 + i*4
		binary.LittleEndian.PutUint32(buf[offset:], math.Float32bits(val))
	}
	
	return buf
}

// Decode deserializes a vector from a byte slice
func Decode(buf []byte) (*Vector, error) {
	if len(buf) < 8 {
		return nil, errors.New("buffer too small to decode vector")
	}
	
	// Read ID length
	idLen := binary.LittleEndian.Uint32(buf[0:])
	
	if int(idLen) + 8 > len(buf) {
		return nil, errors.New("buffer too small to decode vector ID")
	}
	
	// Read ID
	id := string(buf[4 : 4+idLen])
	
	// Read dimension
	dim := binary.LittleEndian.Uint32(buf[4+idLen:])
	
	if int(idLen) + 8 + int(dim)*4 > len(buf) {
		return nil, fmt.Errorf("buffer too small to decode vector values, expected %d bytes", int(idLen) + 8 + int(dim)*4)
	}
	
	// Read values
	values := make([]float32, dim)
	for i := 0; i < int(dim); i++ {
		offset := 4 + int(idLen) + 4 + i*4
		bits := binary.LittleEndian.Uint32(buf[offset:])
		values[i] = math.Float32frombits(bits)
	}
	
	return &Vector{
		ID:        id,
		Values:    values,
		Dimension: int(dim),
	}, nil
}

// Normalize converts the vector to a unit vector (same direction, length 1)
func (v *Vector) Normalize() {
	magnitude := 0.0
	for _, val := range v.Values {
		magnitude += float64(val * val)
	}
	magnitude = math.Sqrt(magnitude)
	
	if magnitude > 0 {
		for i := range v.Values {
			v.Values[i] = float32(float64(v.Values[i]) / magnitude)
		}
	}
} 