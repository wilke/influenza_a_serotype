// Package config provides configuration management for myPaf2Serotypes
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	// Input/Output configuration
	PAFFile     string `mapstructure:"paf_file"`
	MappingFile string `mapstructure:"mapping_file"`
	OutputDir   string `mapstructure:"output_dir"`
	Sample      string `mapstructure:"sample"`

	// Processing configuration
	MinScore      float64 `mapstructure:"min_score"`
	ScoreDistance float64 `mapstructure:"score_distance"`
	NumWorkers    int     `mapstructure:"num_workers"`

	// Feature flags
	VisualizePlot bool `mapstructure:"visualize_plot"`
	Debug         bool `mapstructure:"debug"`

	// Logging configuration
	LogLevel  string `mapstructure:"log_level"`
	LogFormat string `mapstructure:"log_format"` // "text" or "json"

	// Performance configuration
	ChannelBufferSize int           `mapstructure:"channel_buffer_size"`
	ProcessTimeout    time.Duration `mapstructure:"process_timeout"`

	// Advanced configuration
	FileBufferSize int `mapstructure:"file_buffer_size"`
}

// DefaultConfig returns a Config with default values
func DefaultConfig() *Config {
	return &Config{
		MinScore:          0.9,
		ScoreDistance:     0.003,
		NumWorkers:        4,
		LogLevel:          "info",
		LogFormat:         "text",
		ChannelBufferSize: 100,
		ProcessTimeout:    30 * time.Minute,
		FileBufferSize:    4096,
		VisualizePlot:     false,
		Debug:             false,
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Required fields
	if c.PAFFile == "" {
		return fmt.Errorf("paf_file is required")
	}
	if c.MappingFile == "" {
		return fmt.Errorf("mapping_file is required")
	}
	if c.OutputDir == "" {
		return fmt.Errorf("output_dir is required")
	}
	if c.Sample == "" {
		return fmt.Errorf("sample name is required")
	}

	// Validate file existence
	if _, err := os.Stat(c.PAFFile); os.IsNotExist(err) {
		return fmt.Errorf("paf_file does not exist: %s", c.PAFFile)
	}
	if _, err := os.Stat(c.MappingFile); os.IsNotExist(err) {
		return fmt.Errorf("mapping_file does not exist: %s", c.MappingFile)
	}

	// Validate numeric ranges
	if c.MinScore < 0 || c.MinScore > 1 {
		return fmt.Errorf("min_score must be between 0 and 1, got %f", c.MinScore)
	}
	if c.ScoreDistance < 0 || c.ScoreDistance > 1 {
		return fmt.Errorf("score_distance must be between 0 and 1, got %f", c.ScoreDistance)
	}
	if c.NumWorkers < 1 {
		return fmt.Errorf("num_workers must be at least 1, got %d", c.NumWorkers)
	}
	if c.ChannelBufferSize < 1 {
		return fmt.Errorf("channel_buffer_size must be at least 1, got %d", c.ChannelBufferSize)
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("invalid log_level: %s", c.LogLevel)
	}

	// Validate log format
	if c.LogFormat != "text" && c.LogFormat != "json" {
		return fmt.Errorf("log_format must be 'text' or 'json', got %s", c.LogFormat)
	}

	return nil
}

// LoadConfig loads configuration from multiple sources with the following precedence:
// 1. Command line flags (highest)
// 2. Environment variables
// 3. Configuration file
// 4. Default values (lowest)
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values
	config := DefaultConfig()
	v.SetDefault("min_score", config.MinScore)
	v.SetDefault("score_distance", config.ScoreDistance)
	v.SetDefault("num_workers", config.NumWorkers)
	v.SetDefault("log_level", config.LogLevel)
	v.SetDefault("log_format", config.LogFormat)
	v.SetDefault("channel_buffer_size", config.ChannelBufferSize)
	v.SetDefault("process_timeout", config.ProcessTimeout)
	v.SetDefault("file_buffer_size", config.FileBufferSize)
	v.SetDefault("visualize_plot", config.VisualizePlot)
	v.SetDefault("debug", config.Debug)

	// Set up environment variables
	v.SetEnvPrefix("MYPAF")
	v.AutomaticEnv()

	// Load configuration file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	} else {
		// Try to find config in standard locations
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		v.AddConfigPath("/etc/myPaf2Serotypes/")
		v.AddConfigPath("$HOME/.myPaf2Serotypes")

		if err := v.ReadInConfig(); err != nil {
			// It's okay if config file doesn't exist, we have defaults
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
		}
	}

	// Unmarshal configuration
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}

// MergeFlags merges command line flag values into the configuration
func (c *Config) MergeFlags(flags map[string]interface{}) {
	for key, value := range flags {
		switch key {
		case "paf":
			if v, ok := value.(string); ok && v != "" {
				c.PAFFile = v
			}
		case "mapping":
			if v, ok := value.(string); ok && v != "" {
				c.MappingFile = v
			}
		case "output":
			if v, ok := value.(string); ok && v != "" {
				c.OutputDir = v
			}
		case "sample":
			if v, ok := value.(string); ok && v != "" {
				c.Sample = v
			}
		case "min-score":
			if v, ok := value.(float64); ok {
				c.MinScore = v
			}
		case "score-distance":
			if v, ok := value.(float64); ok {
				c.ScoreDistance = v
			}
		case "num-workers":
			if v, ok := value.(int); ok && v > 0 {
				c.NumWorkers = v
			}
		case "visualize-plot":
			if v, ok := value.(bool); ok {
				c.VisualizePlot = v
			}
		case "debug":
			if v, ok := value.(bool); ok {
				c.Debug = v
			}
		}
	}
}

// CreateOutputDir creates the output directory if it doesn't exist
func (c *Config) CreateOutputDir() error {
	return os.MkdirAll(c.OutputDir, 0755)
}

// GetOutputPath returns the full path for an output file
func (c *Config) GetOutputPath(filename string) string {
	return filepath.Join(c.OutputDir, filename)
}