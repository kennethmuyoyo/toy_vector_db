package vector

import (
	"encoding/binary"
	"errors"
	"math"
	"math/rand"
	"strings"
	"time"
)

var (
	// ErrInvalidDimension is returned when vector dimensions don't match
	ErrInvalidDimension = errors.New("invalid vector dimension")
)

// Vector represents a real-valued vector in n-dimensional space
type Vector struct {
	ID        string            // Unique identifier for the vector
	Values    []float32         // Vector components
	Dimension int               // Number of dimensions
	Metadata  map[string]string // Additional metadata for the vector
}

// NewVector creates a new vector with the specified ID and values
func NewVector(id string, values []float32) *Vector {
	return &Vector{
		ID:        id,
		Values:    values,
		Dimension: len(values),
		Metadata:  make(map[string]string),
	}
}

// NewVectorWithMetadata creates a new vector with the specified ID, values, and metadata
func NewVectorWithMetadata(id string, values []float32, metadata map[string]string) *Vector {
	v := NewVector(id, values)
	if metadata != nil {
		v.Metadata = metadata
	}
	return v
}

// Zero creates a zero vector of the specified dimension
func Zero(dimension int) *Vector {
	values := make([]float32, dimension)
	return &Vector{
		ID:        "",
		Values:    values,
		Dimension: dimension,
		Metadata:  make(map[string]string),
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
		Metadata:  make(map[string]string),
	}
}

// Copy creates a deep copy of the vector
func (v *Vector) Copy() *Vector {
	valuesCopy := make([]float32, v.Dimension)
	copy(valuesCopy, v.Values)
	metadataCopy := make(map[string]string)
	for key, value := range v.Metadata {
		metadataCopy[key] = value
	}
	return &Vector{
		ID:        v.ID,
		Values:    valuesCopy,
		Dimension: v.Dimension,
		Metadata:  metadataCopy,
	}
}

// Encode serializes the vector to a byte slice
func (v *Vector) Encode() []byte {
	// Convert metadata to a string representation
	metadataStr := encodeMetadata(v.Metadata)
	metadataBytes := []byte(metadataStr)
	
	// Calculate buffer size: 
	// ID length (4 bytes) + ID + dimension (4 bytes) + values (4 bytes each) + metadata length (4 bytes) + metadata
	idBytes := []byte(v.ID)
	bufSize := 4 + len(idBytes) + 4 + 4*v.Dimension + 4 + len(metadataBytes)
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
	
	// Write metadata length
	metadataLenOffset := 4 + len(idBytes) + 4 + 4*v.Dimension
	binary.LittleEndian.PutUint32(buf[metadataLenOffset:], uint32(len(metadataBytes)))
	
	// Write metadata
	copy(buf[metadataLenOffset+4:], metadataBytes)
	
	return buf
}

// Decode deserializes a vector from a byte slice
func Decode(buf []byte) (*Vector, error) {
	if len(buf) < 8 {
		return nil, errors.New("buffer too small to decode vector")
	}
	
	// Read ID length
	idLen := binary.LittleEndian.Uint32(buf[0:4])
	
	if len(buf) < int(4+idLen+4) {
		return nil, errors.New("buffer too small to decode vector")
	}
	
	// Read ID
	id := string(buf[4 : 4+idLen])
	
	// Read dimension
	dim := binary.LittleEndian.Uint32(buf[4+idLen : 4+idLen+4])
	
	if len(buf) < int(4+idLen+4+dim*4) {
		return nil, errors.New("buffer too small to decode vector values")
	}
	
	// Read values
	values := make([]float32, dim)
	for i := 0; i < int(dim); i++ {
		offset := 4 + idLen + 4 + uint32(i)*4
		values[i] = math.Float32frombits(binary.LittleEndian.Uint32(buf[offset : offset+4]))
	}
	
	// Create vector
	v := &Vector{
		ID:        id,
		Values:    values,
		Dimension: int(dim),
		Metadata:  make(map[string]string),
	}
	
	// Read metadata if available
	metadataLenOffset := 4 + idLen + 4 + dim*4
	if len(buf) > int(metadataLenOffset+4) {
		metadataLen := binary.LittleEndian.Uint32(buf[metadataLenOffset : metadataLenOffset+4])
		
		if len(buf) >= int(metadataLenOffset+4+metadataLen) {
			metadataBytes := buf[metadataLenOffset+4 : metadataLenOffset+4+metadataLen]
			metadata := decodeMetadata(string(metadataBytes))
			v.Metadata = metadata
		}
	}
	
	return v, nil
}

// encodeMetadata converts a metadata map to a string representation
func encodeMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}
	
	// Simple encoding: key1=value1;key2=value2;...
	var result string
	for k, v := range metadata {
		// Escape = and ; characters in keys and values
		k = strings.ReplaceAll(k, "=", "\\=")
		k = strings.ReplaceAll(k, ";", "\\;")
		v = strings.ReplaceAll(v, "=", "\\=")
		v = strings.ReplaceAll(v, ";", "\\;")
		
		if result != "" {
			result += ";"
		}
		result += k + "=" + v
	}
	return result
}

// decodeMetadata converts a string representation back to a metadata map
func decodeMetadata(s string) map[string]string {
	result := make(map[string]string)
	if s == "" {
		return result
	}
	
	// Split by semicolons, but respect escaped semicolons
	pairs := splitRespectingEscapes(s, ';')
	
	for _, pair := range pairs {
		// Split by equals sign, but respect escaped equals signs
		kv := splitRespectingEscapes(pair, '=')
		if len(kv) == 2 {
			// Unescape characters
			k := strings.ReplaceAll(kv[0], "\\=", "=")
			k = strings.ReplaceAll(k, "\\;", ";")
			v := strings.ReplaceAll(kv[1], "\\=", "=")
			v = strings.ReplaceAll(v, "\\;", ";")
			
			result[k] = v
		}
	}
	
	return result
}

// splitRespectingEscapes splits a string by a delimiter, respecting escaped delimiters
func splitRespectingEscapes(s string, delimiter byte) []string {
	var result []string
	var current string
	var escaped bool
	
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			current += string(c)
			escaped = false
		} else if c == '\\' {
			escaped = true
		} else if c == delimiter {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	
	if current != "" || len(s) == 0 {
		result = append(result, current)
	}
	
	return result
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