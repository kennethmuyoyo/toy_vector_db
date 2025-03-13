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
	default:
		fmt.Printf("Unknown command: %s\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

// handleSearch performs a k-nearest neighbor search for a vector
func handleSearch(args []string, store storage.VectorStore, metric distance.Metric) {
	if len(args) < 3 {
		fmt.Println("Error: Missing vector ID and k")
		fmt.Println("Usage: vectodb search <vector-id> <k>")
		os.Exit(1)
	}
	
	// Parse k (number of nearest neighbors)
	k, err := strconv.Atoi(args[2])
	if err != nil {
		fmt.Printf("Error: Invalid value for k: %s\n", args[2])
		os.Exit(1)
	}

	if k < 1 {
		fmt.Println("Error: k must be greater than 0")
		os.Exit(1)
	}
	
	// Get the query vector
	queryVec, err := store.Get(args[1])
	if err != nil {
		if err == storage.ErrVectorNotFound {
			fmt.Printf("Vector %s not found\n", args[1])
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
	
	// Create and build the index
	idx := flat.NewFlatIndex(metric)
	if err := idx.Build(vectors); err != nil {
		fmt.Printf("Error building index: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Searching for %d nearest neighbors to vector %s using %s metric...\n", 
		k, queryVec.ID, metric.Name())
	
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
	fmt.Println("  search   Search for vectors")
	fmt.Println("  add      Add a vector")
	fmt.Println("  get      Get a vector")
	fmt.Println("  list     List all vectors")
	fmt.Println("  delete   Delete a vector")
	fmt.Println("  random   Create a random vector")
} 