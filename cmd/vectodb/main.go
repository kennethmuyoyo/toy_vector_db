package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/ken/vector_database/internal/config"
	"github.com/ken/vector_database/pkg/core/distance"
	"github.com/ken/vector_database/pkg/core/vector"
	"github.com/ken/vector_database/pkg/index/flat"
	"github.com/ken/vector_database/pkg/index/hnsw"
	"github.com/ken/vector_database/pkg/index"
	"github.com/ken/vector_database/pkg/sql/cli"
	"github.com/ken/vector_database/pkg/sql/executor"
	"github.com/ken/vector_database/pkg/storage"
)

const (
	appName    = "VectoDB"
	appVersion = "0.1.0"
)

func main() {
	// Define command-line flags
	var (
		showVersion = flag.Bool("version", false, "Display version information")
		configFile  = flag.String("config", "config.yaml", "Path to configuration file")
		metricName  = flag.String("metric", "euclidean", "Distance metric to use (euclidean, cosine, dotproduct, manhattan)")
		verbose     = flag.Bool("verbose", false, "Enable verbose output")
		indexType   = flag.String("index", "flat", "Index type to use (flat, hnsw)")
	)

	// Parse command-line arguments
	flag.Parse()

	// Display version and exit if requested
	if *showVersion {
		fmt.Printf("%s version %s\n", appName, appVersion)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(cfg.Storage.DataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Parse the metric type
	metricType := distance.MetricType(*metricName)
	metric, err := distance.GetMetric(metricType)
	if err != nil {
		log.Fatalf("Invalid distance metric: %v", err)
	}

	// Create vector store
	store, err := storage.NewFileStore(cfg.Storage.DataDir)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}
	defer store.Close()

	// Get the subcommand
	args := flag.Args()
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}

	// Process subcommands
	switch args[0] {
	case "serve":
		fmt.Println("Starting VectoDB server...")
		// TODO: Implement server startup
	case "import":
		if len(args) < 2 {
			fmt.Println("Error: Missing file path")
			fmt.Println("Usage: vectodb import <file>")
			os.Exit(1)
		}
		fmt.Printf("Importing vectors from %s...\n", args[1])
		// TODO: Implement vector import
	case "export":
		if len(args) < 2 {
			fmt.Println("Error: Missing file path")
			fmt.Println("Usage: vectodb export <file>")
			os.Exit(1)
		}
		fmt.Printf("Exporting vectors to %s...\n", args[1])
		// TODO: Implement vector export
	case "search":
		handleSearch(args, store, metric)
	case "add":
		if len(args) < 3 {
			fmt.Println("Error: Missing vector ID and values")
			fmt.Println("Usage: vectodb add <vector-id> <value1,value2,...>")
			os.Exit(1)
		}
		
		// Parse vector values
		valueStrs := strings.Split(args[2], ",")
		values := make([]float32, len(valueStrs))
		
		for i, valStr := range valueStrs {
			val, err := strconv.ParseFloat(valStr, 32)
			if err != nil {
				fmt.Printf("Error: Invalid vector value at index %d: %s\n", i, valStr)
				os.Exit(1)
			}
			values[i] = float32(val)
		}
		
		// Create and store vector
		v := vector.NewVector(args[1], values)
		if err := store.Insert(v); err != nil {
			if err == storage.ErrVectorAlreadyExists {
				fmt.Printf("Error: Vector with ID %s already exists\n", args[1])
			} else {
				fmt.Printf("Error: %v\n", err)
			}
			os.Exit(1)
		}
		
		fmt.Printf("Added vector %s with dimension %d\n", v.ID, v.Dimension)
	case "get":
		if len(args) < 2 {
			fmt.Println("Error: Missing vector ID")
			fmt.Println("Usage: vectodb get <vector-id>")
			os.Exit(1)
		}
		
		// Get vector from store
		v, err := store.Get(args[1])
		if err != nil {
			if err == storage.ErrVectorNotFound {
				fmt.Printf("Vector %s not found\n", args[1])
			} else {
				fmt.Printf("Error: %v\n", err)
			}
			os.Exit(1)
		}
		
		// Print vector
		fmt.Printf("Vector %s (dimension: %d):\n", v.ID, v.Dimension)
		
		// Print metadata if available
		if len(v.Metadata) > 0 {
			fmt.Println("Metadata:")
			for key, value := range v.Metadata {
				fmt.Printf("  %s: %s\n", key, value)
			}
			fmt.Println("Values:")
		} else {
			fmt.Println("Values:")
		}
		
		// Print vector values
		for i, val := range v.Values {
			fmt.Printf("  [%d]: %f\n", i, val)
		}
	case "list":
		// List all vectors
		ids, err := store.List()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		
		count, _ := store.Count()
		fmt.Printf("Found %d vectors:\n", count)
		for _, id := range ids {
			fmt.Println(id)
		}
	case "delete":
		if len(args) < 2 {
			fmt.Println("Error: Missing vector ID")
			fmt.Println("Usage: vectodb delete <vector-id>")
			os.Exit(1)
		}
		
		// Delete vector from store
		err := store.Delete(args[1])
		if err != nil {
			if err == storage.ErrVectorNotFound {
				fmt.Printf("Vector %s not found\n", args[1])
			} else {
				fmt.Printf("Error: %v\n", err)
			}
			os.Exit(1)
		}
		
		fmt.Printf("Vector %s deleted\n", args[1])
	case "random":
		if len(args) < 3 {
			fmt.Println("Error: Missing vector ID and dimension")
			fmt.Println("Usage: vectodb random <vector-id> <dimension>")
			os.Exit(1)
		}
		
		// Parse dimension
		dim, err := strconv.Atoi(args[2])
		if err != nil {
			fmt.Printf("Error: Invalid dimension: %s\n", args[2])
			os.Exit(1)
		}
		
		// Create random vector
		v := vector.Random(args[1], dim)
		
		// Store vector
		if err := store.Insert(v); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("Created random vector %s with dimension %d\n", v.ID, v.Dimension)
	case "sql":
		handleSQL(args, store, metric, *indexType, *verbose)
	case "embed":
		if len(args) < 2 {
			fmt.Println("Error: Missing embed type")
			fmt.Println("Usage: vectodb embed [text|file|json] <id> <content>")
			os.Exit(1)
		}
		
		// Pass the remaining arguments to the embed command handler
		if err := HandleEmbedCommand(args[1:]); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "search-text":
		if len(args) < 1 {
			fmt.Println("Error: Missing text query")
			fmt.Println("Usage: vectodb search-text <text query>")
			os.Exit(1)
		}
		textQuery := strings.Join(args, " ")
		HandleSearchTextCommand(textQuery, metric, *indexType, *verbose)
	case "set-metadata":
		if len(args) < 4 {
			fmt.Println("Error: Missing parameters")
			fmt.Println("Usage: vectodb set-metadata <vector-id> <key> <value>")
			os.Exit(1)
		}
		
		// Get vector from store
		v, err := store.Get(args[1])
		if err != nil {
			if err == storage.ErrVectorNotFound {
				fmt.Printf("Vector %s not found\n", args[1])
			} else {
				fmt.Printf("Error: %v\n", err)
			}
			os.Exit(1)
		}
		
		// Set metadata
		key := args[2]
		value := args[3]
		
		// Initialize metadata map if nil
		if v.Metadata == nil {
			v.Metadata = make(map[string]string)
		}
		
		v.Metadata[key] = value
		
		// Update vector in store
		if err := store.Update(v); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("Set metadata %s=%s for vector %s\n", key, value, v.ID)
	default:
		fmt.Printf("Unknown command: %s\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

// handleSQL executes SQL queries against the vector database
func handleSQL(args []string, store storage.VectorStore, metric distance.Metric, indexType string, verbose bool) {
	if len(args) < 2 {
		fmt.Println("Error: Missing SQL query")
		fmt.Println("Usage: vectodb sql \"<query>\"")
		fmt.Println("Examples:")
		fmt.Println("  vectodb sql \"SELECT id, dimension FROM vectors LIMIT 5\"")
		fmt.Println("  vectodb sql \"SELECT id, dimension FROM vectors WHERE id LIKE 'test%'\"")
		fmt.Println("  vectodb sql \"SELECT id FROM vectors WHERE metadata.category = 'image'\"")
		fmt.Println("  vectodb sql \"SELECT id FROM vectors WHERE metadata.tags LIKE '%important%'\"")
		fmt.Println("  vectodb sql \"SELECT id, distance FROM vectors NEAREST TO [1.0,2.0,3.0] USING euclidean LIMIT 3\"")
		fmt.Println("  vectodb sql \"INSERT INTO vectors (id, vector) VALUES ('vec123', [1.0,2.0,3.0])\"")
		fmt.Println("  vectodb sql \"DELETE FROM vectors WHERE id = 'vec123'\"")
		os.Exit(1)
	}
	
	// Convert index type string to executor.IndexType
	var idxType executor.IndexType
	switch strings.ToLower(indexType) {
	case "flat":
		idxType = executor.IndexTypeFlat
	case "hnsw":
		idxType = executor.IndexTypeHNSW
	default:
		fmt.Printf("Error: Unsupported index type: %s\n", indexType)
		fmt.Println("Supported index types: flat, hnsw")
		os.Exit(1)
	}
	
	// Create SQL service
	sqlService := cli.NewSQLService(store, idxType, metric)
	sqlService.SetVerbose(verbose)
	
	// Execute SQL query
	result, err := sqlService.Execute(args[1])
	if err != nil {
		fmt.Printf("SQL Error: %v\n", err)
		os.Exit(1)
	}
	
	// Print result
	fmt.Println(result)
}

// handleSearch performs a k-nearest neighbor search for a vector
func handleSearch(args []string, store storage.VectorStore, metric distance.Metric) {
	if len(args) < 4 {
		fmt.Println("Error: Missing parameters")
		fmt.Println("Usage: vectodb search <index-type> <vector-id> <k>")
		fmt.Println("  index-type: The type of index to use (flat, hnsw)")
		fmt.Println("  vector-id: The ID of the query vector")
		fmt.Println("  k: The number of nearest neighbors to find")
		os.Exit(1)
	}
	
	// Get the index type
	indexType := args[1]
	if indexType != "flat" && indexType != "hnsw" {
		fmt.Printf("Error: Unsupported index type: %s\n", indexType)
		fmt.Println("Supported index types: flat, hnsw")
		os.Exit(1)
	}
	
	// Parse k (number of nearest neighbors)
	k, err := strconv.Atoi(args[3])
	if err != nil {
		fmt.Printf("Error: Invalid value for k: %s\n", args[3])
		os.Exit(1)
	}

	if k < 1 {
		fmt.Println("Error: k must be greater than 0")
		os.Exit(1)
	}
	
	// Get the query vector
	queryVec, err := store.Get(args[2])
	if err != nil {
		if err == storage.ErrVectorNotFound {
			fmt.Printf("Vector %s not found\n", args[2])
		} else {
			fmt.Printf("Error: %v\n", err)
		}
		os.Exit(1)
	}
	
	// List all vectors
	ids, err := store.List()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Get all vectors
	vectors := make([]*vector.Vector, 0, len(ids))
	for _, id := range ids {
		v, err := store.Get(id)
		if err != nil {
			fmt.Printf("Error getting vector %s: %v\n", id, err)
			continue
		}
		vectors = append(vectors, v)
	}
	
	// Create an appropriate index based on the specified type
	var idx index.Index
	switch indexType {
	case "flat":
		idx = flat.NewFlatIndex(metric)
	case "hnsw":
		// Create an HNSW index with default configuration
		idx = hnsw.NewHNSWIndex(metric, nil)
	}
	
	// Build the index
	if err := idx.Build(vectors); err != nil {
		fmt.Printf("Error building index: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Searching for %d nearest neighbors to vector %s using %s index with %s metric...\n", 
		k, queryVec.ID, idx.Name(), metric.Name())
	
	// Perform the search
	results, err := idx.Search(queryVec, k)
	if err != nil {
		fmt.Printf("Error during search: %v\n", err)
		os.Exit(1)
	}
	
	// Display results
	fmt.Printf("Found %d results:\n", len(results))
	for i, result := range results {
		// Skip the query vector itself
		if result.ID == queryVec.ID {
			continue
		}
		fmt.Printf("%d. %s (distance: %.6f)\n", i+1, result.ID, result.Distance)
	}
}

func printUsage() {
	fmt.Printf("%s - A vector database implemented in Go\n\n", appName)
	fmt.Println("Usage:")
	fmt.Println("  vectodb [flags] <command>")
	fmt.Println("\nFlags:")
	flag.PrintDefaults()
	fmt.Println("\nCommands:")
	fmt.Println("  serve    Start the VectoDB server")
	fmt.Println("  import   Import vectors from a file")
	fmt.Println("  export   Export vectors to a file")
	fmt.Println("  search   Search for vectors (Usage: vectodb search <index-type> <vector-id> <k>)")
	fmt.Println("           index-type: flat, hnsw")
	fmt.Println("  sql      Execute SQL query (Usage: vectodb sql \"<query>\")")
	fmt.Println("  add      Add a vector")
	fmt.Println("  get      Get a vector")
	fmt.Println("  list     List all vectors")
	fmt.Println("  delete   Delete a vector")
	fmt.Println("  random   Create a random vector")
	fmt.Println("  embed    Embed text or file content as a vector")
	fmt.Println("  search-text <text query>  Search using text similarity")
	fmt.Println("  set-metadata <vector-id> <key> <value>  Set vector metadata")
} 