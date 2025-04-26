package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/log"
)

// Config represents the application configuration
type Config struct {
	// Input files
	MappingFile string
	PafFile     string

	// Output settings
	OutputDir  string
	SampleName string

	// Processing settings
	ScoreThreshold      float64
	AmbiguityThreshold  float64
	ChunkSize           int
	MinChunkSize        int  // Minimum chunk size for adaptive sizing
	MaxChunkSize        int  // Maximum chunk size for adaptive sizing
	AdaptiveChunkSize   bool // Whether to use adaptive chunk sizing
	Workers             int
	MaxWorkers          int  // Maximum number of workers for dynamic scaling
	DynamicWorkerPool   bool // Whether to use dynamic worker pool sizing
	UseObjectPooling    bool // Whether to use object pooling for frequently created objects
	EnableVisualization bool

	// Logging settings
	LogLevel log.LogLevel
}

// NewDefaultConfig creates a new default configuration
func NewDefaultConfig() *Config {
	// Get system information for default settings
	numCPU := runtime.NumCPU()

	// Calculate memory-based defaults
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Available memory in GB (with 20% safety margin)
	availableMemGB := float64(memStats.Sys) / (1024 * 1024 * 1024) * 0.8

	// Default chunk size based on available memory
	// Assume each record takes about 200 bytes on average
	// Target using about 10% of available memory for chunk buffer
	defaultChunkSize := int(availableMemGB * 1024 * 1024 * 100)
	if defaultChunkSize < 1000 {
		defaultChunkSize = 1000 // Minimum default
	}
	if defaultChunkSize > 10000 {
		defaultChunkSize = 10000 // Maximum default
	}

	return &Config{
		OutputDir:           ".",
		SampleName:          "sample",
		ScoreThreshold:      0.8,
		AmbiguityThreshold:  0.003,
		ChunkSize:           defaultChunkSize,
		MinChunkSize:        1000,
		MaxChunkSize:        10000,
		AdaptiveChunkSize:   true,
		Workers:             numCPU,
		MaxWorkers:          numCPU * 2,
		DynamicWorkerPool:   true,
		UseObjectPooling:    true,
		EnableVisualization: true,
		LogLevel:            log.LevelInfo,
	}
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Check required fields
	if c.MappingFile == "" {
		log.Error("mapping file is required")
		return fmt.Errorf("mapping file is required")
	}
	if c.PafFile == "" {
		log.Error("PAF file is required")
		return fmt.Errorf("PAF file is required")
	}

	// Check file existence
	if _, err := os.Stat(c.MappingFile); os.IsNotExist(err) {
		log.Error("mapping file does not exist: %s", c.MappingFile)
		return fmt.Errorf("mapping file does not exist: %s", c.MappingFile)
	}
	if _, err := os.Stat(c.PafFile); os.IsNotExist(err) {
		log.Error("PAF file does not exist: %s", c.PafFile)
		return fmt.Errorf("PAF file does not exist: %s", c.PafFile)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(c.OutputDir, 0755); err != nil {
		log.Error("failed to create output directory: %v", err)
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Validate numeric values
	if c.ScoreThreshold <= 0 || c.ScoreThreshold > 1 {
		log.Error("score threshold must be between 0 and 1")
		return fmt.Errorf("score threshold must be between 0 and 1")
	}
	if c.AmbiguityThreshold < 0 || c.AmbiguityThreshold > 1 {
		log.Error("ambiguity threshold must be between 0 and 1")
		return fmt.Errorf("ambiguity threshold must be between 0 and 1")
	}
	if c.ChunkSize <= 0 {
		log.Error("chunk size must be greater than 0")
		return fmt.Errorf("chunk size must be greater than 0")
	}
	if c.Workers <= 0 {
		log.Error("workers must be greater than 0")
		return fmt.Errorf("workers must be greater than 0")
	}

	// Validate adaptive chunk size settings
	if c.AdaptiveChunkSize {
		if c.MinChunkSize <= 0 {
			log.Error("minimum chunk size must be greater than 0")
			return fmt.Errorf("minimum chunk size must be greater than 0")
		}
		if c.MaxChunkSize <= c.MinChunkSize {
			log.Error("maximum chunk size must be greater than minimum chunk size")
			return fmt.Errorf("maximum chunk size must be greater than minimum chunk size")
		}
	}

	// Validate dynamic worker pool settings
	if c.DynamicWorkerPool {
		if c.MaxWorkers < c.Workers {
			log.Error("maximum workers must be greater than or equal to workers")
			return fmt.Errorf("maximum workers must be greater than or equal to workers")
		}
	}

	return nil
}

// SetupLogging sets up logging based on the configuration
func (c *Config) SetupLogging() error {
	// Set log level
	log.SetLevel(c.LogLevel)

	// Create log file
	logFile := filepath.Join(c.OutputDir, fmt.Sprintf("%s.log", c.SampleName))
	file, err := os.Create(logFile)
	if err != nil {
		log.Error("failed to create log file: %v", err)
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Set log writer to both stdout and file
	log.SetWriter(file)

	return nil
}

// String returns a string representation of the configuration
func (c *Config) String() string {
	log.Info("Generating configuration string representation")
	return fmt.Sprintf(`Configuration:
  Input:
    Mapping File: %s
    PAF File: %s
  Output:
    Output Directory: %s
    Sample Name: %s
  Processing:
    Score Threshold: %.3f
    Ambiguity Threshold: %.3f
    Chunk Size: %d
    Adaptive Chunk Size: %t
    Min Chunk Size: %d
    Max Chunk Size: %d
    Workers: %d
    Dynamic Worker Pool: %t
    Max Workers: %d
    Use Object Pooling: %t
    Enable Visualization: %t
  Logging:
    Log Level: %s`,
		c.MappingFile,
		c.PafFile,
		c.OutputDir,
		c.SampleName,
		c.ScoreThreshold,
		c.AmbiguityThreshold,
		c.ChunkSize,
		c.AdaptiveChunkSize,
		c.MinChunkSize,
		c.MaxChunkSize,
		c.Workers,
		c.DynamicWorkerPool,
		c.MaxWorkers,
		c.UseObjectPooling,
		c.EnableVisualization,
		c.LogLevel.String())
}
