package hnsw

import (
	"encoding/gob"
	"errors"
	"math"
	"math/rand"
	"os"
	"sort"
	"sync"

	"github.com/ken/vector_database/pkg/core/distance"
	"github.com/ken/vector_database/pkg/core/vector"
	"github.com/ken/vector_database/pkg/index"
)

var (
	// ErrVectorNotFound is returned when a vector with the specified ID is not found
	ErrVectorNotFound = errors.New("vector not found")

	// ErrVectorAlreadyExists is returned when attempting to add a vector with an ID that already exists
	ErrVectorAlreadyExists = errors.New("vector already exists")

	// ErrInvalidK is returned when k is less than 1
	ErrInvalidK = errors.New("k must be greater than 0")

	// ErrNoVectors is returned when the index is empty
	ErrNoVectors = errors.New("index contains no vectors")

	// ErrMetricRequired is returned when a distance metric is required but not set
	ErrMetricRequired = errors.New("distance metric is required")
)

// HNSWConfig holds the configuration parameters for the HNSW index
type HNSWConfig struct {
	M              int     // Maximum number of connections per node (default: 16)
	EfConstruction int     // Size of the dynamic candidate list for construction (default: 200)
	EfSearch       int     // Size of the dynamic candidate list for search (default: 50)
	MaxLevel       int     // Maximum level in the graph (default: calculated based on size)
	LevelMult      float64 // Level probability multiplier (default: 1/ln(M))
}

// DefaultHNSWConfig returns the default configuration for HNSW
func DefaultHNSWConfig() HNSWConfig {
	m := 16
	return HNSWConfig{
		M:              m,
		EfConstruction: 200,
		EfSearch:       50,
		MaxLevel:       0, // Will be calculated based on data size
		LevelMult:      1.0 / math.Log(float64(m)),
	}
}

// Node represents a node in the HNSW graph
type Node struct {
	Vector   *vector.Vector           // The vector stored in this node
	Edges    []map[string]float32     // Edges[level][neighborID] = distance
	Level    int                      // The level of this node in the graph
	Deleted  bool                     // Whether this node has been marked as deleted
}

// HNSWIndex implements an HNSW (Hierarchical Navigable Small World) index
type HNSWIndex struct {
	nodes         map[string]*Node    // Map of vector ID to node
	entryPoint    string              // ID of the entry point node (highest level)
	currentMaxLevel int               // Current maximum level in the graph
	metric        distance.Metric     // Distance metric to use
	config        HNSWConfig          // Configuration parameters
	mu            sync.RWMutex        // Mutex for thread safety
	rng           *rand.Rand          // Random number generator for level assignment
}

// NewHNSWIndex creates a new HNSW index with the specified distance metric and configuration
func NewHNSWIndex(metric distance.Metric, config *HNSWConfig) *HNSWIndex {
	cfg := DefaultHNSWConfig()
	if config != nil {
		cfg = *config
	}

	return &HNSWIndex{
		nodes:          make(map[string]*Node),
		entryPoint:     "",
		currentMaxLevel: 0,
		metric:         metric,
		config:         cfg,
		rng:            rand.New(rand.NewSource(rand.Int63())),
	}
}

// Name returns the name of the index
func (idx *HNSWIndex) Name() string {
	return "hnsw"
}

// randomLevel generates a random level for a new node
func (idx *HNSWIndex) randomLevel() int {
	level := 0
	for level < idx.config.MaxLevel && idx.rng.Float64() < idx.config.LevelMult {
		level++
	}
	return level
}

// distance calculates the distance between two vectors
func (idx *HNSWIndex) distance(a, b *vector.Vector) (float32, error) {
	if idx.metric == nil {
		return 0, ErrMetricRequired
	}
	return idx.metric.Distance(a, b)
}

// Build constructs the index from a set of vectors
func (idx *HNSWIndex) Build(vectors []*vector.Vector) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Reset the index
	idx.nodes = make(map[string]*Node)
	idx.entryPoint = ""
	idx.currentMaxLevel = 0

	// Calculate maxLevel based on the number of vectors if not specified
	if idx.config.MaxLevel == 0 {
		n := len(vectors)
		if n > 0 {
			idx.config.MaxLevel = int(math.Log(float64(n)) / math.Log(float64(idx.config.M)))
		} else {
			idx.config.MaxLevel = 1 // Default for empty dataset
		}
	}

	// Add each vector to the index
	for _, vec := range vectors {
		err := idx.addInternal(vec.Copy()) // Store a copy of the vector
		if err != nil {
			return err
		}
	}

	return nil
}

// Add adds a vector to the index
func (idx *HNSWIndex) Add(vec *vector.Vector) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Check if the vector already exists
	if _, exists := idx.nodes[vec.ID]; exists {
		return ErrVectorAlreadyExists
	}

	return idx.addInternal(vec.Copy())
}

// addInternal adds a vector to the index (without locking)
func (idx *HNSWIndex) addInternal(vec *vector.Vector) error {
	// Generate a random level for the new node
	nodeLevel := idx.randomLevel()

	// Create a new node with the given level
	node := &Node{
		Vector:  vec,
		Edges:   make([]map[string]float32, nodeLevel+1),
		Level:   nodeLevel,
		Deleted: false,
	}

	// Initialize edge maps at each level
	for i := 0; i <= nodeLevel; i++ {
		node.Edges[i] = make(map[string]float32)
	}

	// Add the node to the index
	idx.nodes[vec.ID] = node

	// If this is the first node, set it as the entry point and return
	if idx.entryPoint == "" {
		idx.entryPoint = vec.ID
		idx.currentMaxLevel = nodeLevel
		return nil
	}

	// Get the entry point
	ep := idx.entryPoint
	
	// Update max level if needed
	if nodeLevel > idx.currentMaxLevel {
		idx.currentMaxLevel = nodeLevel
		idx.entryPoint = vec.ID
	}

	// Connect the new node to the graph
	for level := min(nodeLevel, idx.currentMaxLevel); level >= 0; level-- {
		// Search for nearest neighbors at current level
		neighbors := idx.searchLayerInternal(vec, ep, idx.config.EfConstruction, level)
		
		// Connect to M nearest neighbors at this level
		m := idx.config.M
		if level == 0 {
			// Allow more connections at the bottom level
			m = 2 * idx.config.M
		}

		// Cap number of neighbors to avoid too many connections
		if len(neighbors) > m {
			neighbors = neighbors[:m]
		}

		// Create bidirectional connections
		for _, nbr := range neighbors {
			// Connect new node to neighbor
			node.Edges[level][nbr.ID] = nbr.Distance
			
			// Connect neighbor to new node
			neighborNode := idx.nodes[nbr.ID]
			
			// Skip deleted nodes
			if neighborNode.Deleted {
				continue
			}
			
			// Ensure the neighbor has a map for this level
			if level <= neighborNode.Level {
				neighborNode.Edges[level][vec.ID] = nbr.Distance
				
				// Prune edges if neighbor has too many connections
				if len(neighborNode.Edges[level]) > m {
					idx.pruneConnections(neighborNode, level, m)
				}
			}
		}
		
		// Update entry point for next level
		if len(neighbors) > 0 {
			ep = neighbors[0].ID
		}
	}

	return nil
}

// pruneConnections reduces the number of connections at a specific level to m
func (idx *HNSWIndex) pruneConnections(node *Node, level, m int) {
	// If we already have fewer than m connections, do nothing
	if len(node.Edges[level]) <= m {
		return
	}

	// Create a sorted list of neighbors by distance
	neighbors := make([]struct {
		ID       string
		Distance float32
	}, 0, len(node.Edges[level]))

	for id, dist := range node.Edges[level] {
		neighbors = append(neighbors, struct {
			ID       string
			Distance float32
		}{id, dist})
	}

	// Sort by distance
	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].Distance < neighbors[j].Distance
	})

	// Create a new edge map with only the m closest neighbors
	newEdges := make(map[string]float32, m)
	for i := 0; i < m && i < len(neighbors); i++ {
		newEdges[neighbors[i].ID] = neighbors[i].Distance
	}

	// Replace the old edges with the pruned set
	node.Edges[level] = newEdges
}

// searchLayerInternal performs a search within a single layer of the HNSW graph
func (idx *HNSWIndex) searchLayerInternal(query *vector.Vector, entryID string, ef int, level int) []struct {
	ID       string
	Distance float32
} {
	// Check if the index is empty
	if len(idx.nodes) == 0 {
		return nil
	}

	// Get the entry point
	entryNode, exists := idx.nodes[entryID]
	if !exists || entryNode.Deleted {
		// If the entry point is invalid, find any non-deleted node
		for id, node := range idx.nodes {
			if !node.Deleted {
				entryID = id
				entryNode = node
				break
			}
		}
		// If no valid nodes found, return empty result
		if entryNode == nil || entryNode.Deleted {
			return nil
		}
	}

	// Calculate distance to entry point
	entryDist, err := idx.distance(query, entryNode.Vector)
	if err != nil {
		return nil
	}

	// Initialize visited set to avoid revisiting nodes
	visited := make(map[string]bool)
	visited[entryID] = true

	// Initialize candidate list with entry point
	candidates := &distQueue{maxSize: ef}
	candidates.push(entryID, entryDist)

	// Initialize result list with entry point
	results := &distQueue{maxSize: ef}
	results.push(entryID, entryDist)

	// Perform the search
	for !candidates.empty() {
		// Get the closest candidate
		current := candidates.pop()

		// If the furthest result is closer than this candidate, we're done
		if results.maxDist() < current.distance {
			break
		}

		// Get the current node
		currentNode := idx.nodes[current.id]
		if currentNode.Deleted || level > currentNode.Level {
			continue
		}

		// Check all neighbors at this level
		for neighborID, _ := range currentNode.Edges[level] {
			// Skip already visited nodes
			if visited[neighborID] {
				continue
			}
			visited[neighborID] = true

			// Get the neighbor node
			neighborNode := idx.nodes[neighborID]
			if neighborNode.Deleted {
				continue
			}

			// Calculate distance to the neighbor
			neighborDist, err := idx.distance(query, neighborNode.Vector)
			if err != nil {
				continue
			}

			// If the neighbor is closer than the furthest result or we haven't filled the results yet
			if results.size() < ef || neighborDist < results.maxDist() {
				candidates.push(neighborID, neighborDist)
				results.push(neighborID, neighborDist)
			}
		}
	}

	// Convert result queue to a sorted slice
	resultSlice := make([]struct {
		ID       string
		Distance float32
	}, results.size())

	for i := 0; !results.empty(); i++ {
		result := results.pop()
		resultSlice[i] = struct {
			ID       string
			Distance float32
		}{result.id, result.distance}
	}

	// Sort results by distance
	sort.Slice(resultSlice, func(i, j int) bool {
		return resultSlice[i].Distance < resultSlice[j].Distance
	})

	return resultSlice
}

// Delete removes a vector from the index
func (idx *HNSWIndex) Delete(id string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Check if the vector exists
	node, exists := idx.nodes[id]
	if !exists {
		return ErrVectorNotFound
	}

	// Mark the node as deleted
	node.Deleted = true

	// If the deleted node was the entry point, find a new entry point
	if idx.entryPoint == id {
		idx.updateEntryPoint()
	}

	return nil
}

// updateEntryPoint finds a new entry point after the current one is deleted
func (idx *HNSWIndex) updateEntryPoint() {
	// Reset the entry point and max level
	idx.entryPoint = ""
	idx.currentMaxLevel = 0

	// Find the node with the highest level that isn't deleted
	for id, node := range idx.nodes {
		if !node.Deleted && node.Level > idx.currentMaxLevel {
			idx.entryPoint = id
			idx.currentMaxLevel = node.Level
		}
	}
	
	// If no entry point was found (all nodes are deleted), try to find any non-deleted node
	if idx.entryPoint == "" {
		for id, node := range idx.nodes {
			if !node.Deleted {
				idx.entryPoint = id
				idx.currentMaxLevel = node.Level
				break
			}
		}
	}
}

// Search performs a k-nearest neighbor search
func (idx *HNSWIndex) Search(query *vector.Vector, k int) (index.SearchResults, error) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Check if the index is empty
	if len(idx.nodes) == 0 {
		return nil, ErrNoVectors
	}

	// Check if k is valid
	if k < 1 {
		return nil, ErrInvalidK
	}

	// Check if a metric is set
	if idx.metric == nil {
		return nil, ErrMetricRequired
	}

	// If no valid entry point exists, try to find one
	if idx.entryPoint == "" || idx.nodes[idx.entryPoint].Deleted {
		// Count non-deleted nodes
		validNodeCount := 0
		for _, node := range idx.nodes {
			if !node.Deleted {
				validNodeCount++
			}
		}

		// If no valid nodes, return error
		if validNodeCount == 0 {
			return nil, ErrNoVectors
		}

		// Release read lock and acquire write lock to update entry point
		idx.mu.RUnlock()
		idx.mu.Lock()
		idx.updateEntryPoint()
		idx.mu.Unlock()
		idx.mu.RLock()
		
		// If still no entry point, the index is effectively empty
		if idx.entryPoint == "" {
			return nil, ErrNoVectors
		}
	}

	// Start from the top level and descend to level 0
	ep := idx.entryPoint
	
	// Search from top level to level 1
	for level := idx.currentMaxLevel; level > 0; level-- {
		// Find closest node at this level
		neighbors := idx.searchLayerInternal(query, ep, 1, level)
		if len(neighbors) > 0 {
			ep = neighbors[0].ID
		}
	}

	// Perform the final search at level 0 with ef=k
	neighbors := idx.searchLayerInternal(query, ep, max(k, idx.config.EfSearch), 0)

	// Convert to SearchResults
	results := make(index.SearchResults, 0, min(k, len(neighbors)))
	for i, neighbor := range neighbors {
		if i >= k {
			break
		}
		
		node := idx.nodes[neighbor.ID]
		if node.Deleted {
			continue
		}
		
		results = append(results, index.SearchResult{
			ID:       neighbor.ID,
			Vector:   node.Vector.Copy(), // Return a copy to prevent modification
			Distance: neighbor.Distance,
		})
	}

	return results, nil
}

// Size returns the number of vectors in the index
func (idx *HNSWIndex) Size() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Count non-deleted nodes
	count := 0
	for _, node := range idx.nodes {
		if !node.Deleted {
			count++
		}
	}

	return count
}

// GetIDs returns all vector IDs in the index
func (idx *HNSWIndex) GetIDs() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Collect IDs of non-deleted nodes
	ids := make([]string, 0, len(idx.nodes))
	for id, node := range idx.nodes {
		if !node.Deleted {
			ids = append(ids, id)
		}
	}

	return ids
}

// Save persists the index to the specified path
func (idx *HNSWIndex) Save(path string) error {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// Create the file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a gob encoder
	encoder := gob.NewEncoder(file)

	// Create a serializable version of the index
	type indexData struct {
		Nodes           map[string]*Node
		EntryPoint      string
		CurrentMaxLevel int
		Config          HNSWConfig
		Metric          string
	}

	// Get the metric name
	var metricName string
	if idx.metric != nil {
		metricName = string(idx.metric.Name())
	}

	// Encode the index
	data := indexData{
		Nodes:           idx.nodes,
		EntryPoint:      idx.entryPoint,
		CurrentMaxLevel: idx.currentMaxLevel,
		Config:          idx.config,
		Metric:          metricName,
	}
	
	if err := encoder.Encode(data); err != nil {
		return err
	}

	return nil
}

// Load loads the index from the specified path
func (idx *HNSWIndex) Load(path string) error {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Open the file
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a gob decoder
	decoder := gob.NewDecoder(file)

	// Define the serializable version of the index
	type indexData struct {
		Nodes           map[string]*Node
		EntryPoint      string
		CurrentMaxLevel int
		Config          HNSWConfig
		Metric          string
	}

	// Decode the index
	var data indexData
	if err := decoder.Decode(&data); err != nil {
		return err
	}

	// Update the index
	idx.nodes = data.Nodes
	idx.entryPoint = data.EntryPoint
	idx.currentMaxLevel = data.CurrentMaxLevel
	idx.config = data.Config

	// Set the metric if it's not already set
	if idx.metric == nil && data.Metric != "" {
		metric, err := distance.GetMetric(distance.MetricType(data.Metric))
		if err != nil {
			return err
		}
		idx.metric = metric
	}

	// Initialize RNG if not already initialized
	if idx.rng == nil {
		idx.rng = rand.New(rand.NewSource(rand.Int63()))
	}

	return nil
}

// SetMetric sets the distance metric used by the index
func (idx *HNSWIndex) SetMetric(metric distance.Metric) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.metric = metric
}

// helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// distQueue is a priority queue for distances
type distItem struct {
	id       string
	distance float32
}

type distQueue struct {
	items   []distItem
	maxSize int
}

func (q *distQueue) push(id string, distance float32) {
	// Add the item
	q.items = append(q.items, distItem{id, distance})
	
	// If we're over capacity, remove the furthest item
	if q.maxSize > 0 && len(q.items) > q.maxSize {
		// Find the index of the furthest item
		maxIndex := 0
		maxDist := q.items[0].distance
		
		for i := 1; i < len(q.items); i++ {
			if q.items[i].distance > maxDist {
				maxDist = q.items[i].distance
				maxIndex = i
			}
		}
		
		// Remove the furthest item
		q.items = append(q.items[:maxIndex], q.items[maxIndex+1:]...)
	}
}

func (q *distQueue) pop() distItem {
	// Find the closest item
	minIndex := 0
	minDist := q.items[0].distance
	
	for i := 1; i < len(q.items); i++ {
		if q.items[i].distance < minDist {
			minDist = q.items[i].distance
			minIndex = i
		}
	}
	
	// Get the closest item
	item := q.items[minIndex]
	
	// Remove it from the queue
	q.items = append(q.items[:minIndex], q.items[minIndex+1:]...)
	
	return item
}

func (q *distQueue) empty() bool {
	return len(q.items) == 0
}

func (q *distQueue) size() int {
	return len(q.items)
}

func (q *distQueue) maxDist() float32 {
	if len(q.items) == 0 {
		return 0
	}
	
	// Find the maximum distance
	maxDist := q.items[0].distance
	for i := 1; i < len(q.items); i++ {
		if q.items[i].distance > maxDist {
			maxDist = q.items[i].distance
		}
	}
	
	return maxDist
} 