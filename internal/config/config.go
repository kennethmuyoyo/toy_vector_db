package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Storage  StorageConfig  `yaml:"storage"`
	Vector   VectorConfig   `yaml:"vector"`
	Indexing IndexingConfig `yaml:"indexing"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// StorageConfig holds storage-related configuration
type StorageConfig struct {
	DataDir string `yaml:"data_dir"`
}

// VectorConfig holds vector-related configuration
type VectorConfig struct {
	DefaultDimension int `yaml:"default_dimension"`
}

// IndexingConfig holds indexing-related configuration
type IndexingConfig struct {
	Type           string `yaml:"type"`
	HNSWMaxLinks   int    `yaml:"hnsw_max_links"`
	HNSWEFConstruct int    `yaml:"hnsw_ef_construct"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 8080,
		},
		Storage: StorageConfig{
			DataDir: "./data",
		},
		Vector: VectorConfig{
			DefaultDimension: 128,
		},
		Indexing: IndexingConfig{
			Type:           "hnsw",
			HNSWMaxLinks:   16,
			HNSWEFConstruct: 200,
		},
	}
}

// LoadConfig loads the configuration from a file
func LoadConfig(path string) (*Config, error) {
	// Start with default config
	config := DefaultConfig()

	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if the file exists
	_, err = os.Stat(absPath)
	if os.IsNotExist(err) {
		return config, nil // Return default config if file doesn't exist
	}

	// Read the file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

// SaveConfig saves the configuration to a file
func SaveConfig(config *Config, path string) error {
	// Convert config to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
} 