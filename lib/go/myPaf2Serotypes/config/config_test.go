package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, 0.9, cfg.MinScore)
	assert.Equal(t, 0.003, cfg.ScoreDistance)
	assert.Equal(t, 4, cfg.NumWorkers)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, "text", cfg.LogFormat)
	assert.Equal(t, 100, cfg.ChannelBufferSize)
	assert.Equal(t, 4096, cfg.FileBufferSize)
	assert.False(t, cfg.VisualizePlot)
	assert.False(t, cfg.Debug)
}

func TestConfigValidate(t *testing.T) {
	// Create temporary files for testing
	tmpDir := t.TempDir()
	pafFile := filepath.Join(tmpDir, "test.paf")
	mappingFile := filepath.Join(tmpDir, "mapping.txt")
	
	// Create the files
	require.NoError(t, os.WriteFile(pafFile, []byte("test"), 0644))
	require.NoError(t, os.WriteFile(mappingFile, []byte("test"), 0644))

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid configuration",
			config: &Config{
				PAFFile:           pafFile,
				MappingFile:       mappingFile,
				OutputDir:         tmpDir,
				Sample:            "test_sample",
				MinScore:          0.9,
				ScoreDistance:     0.003,
				NumWorkers:        4,
				LogLevel:          "info",
				LogFormat:         "text",
				ChannelBufferSize: 100,
			},
			wantErr: false,
		},
		{
			name: "Missing PAF file",
			config: &Config{
				MappingFile: mappingFile,
				OutputDir:   tmpDir,
				Sample:      "test_sample",
			},
			wantErr: true,
			errMsg:  "paf_file is required",
		},
		{
			name: "Missing mapping file",
			config: &Config{
				PAFFile:   pafFile,
				OutputDir: tmpDir,
				Sample:    "test_sample",
			},
			wantErr: true,
			errMsg:  "mapping_file is required",
		},
		{
			name: "Missing output directory",
			config: &Config{
				PAFFile:     pafFile,
				MappingFile: mappingFile,
				Sample:      "test_sample",
			},
			wantErr: true,
			errMsg:  "output_dir is required",
		},
		{
			name: "Missing sample name",
			config: &Config{
				PAFFile:     pafFile,
				MappingFile: mappingFile,
				OutputDir:   tmpDir,
			},
			wantErr: true,
			errMsg:  "sample name is required",
		},
		{
			name: "Non-existent PAF file",
			config: &Config{
				PAFFile:     "/non/existent/file.paf",
				MappingFile: mappingFile,
				OutputDir:   tmpDir,
				Sample:      "test_sample",
			},
			wantErr: true,
			errMsg:  "paf_file does not exist",
		},
		{
			name: "Invalid min score",
			config: &Config{
				PAFFile:     pafFile,
				MappingFile: mappingFile,
				OutputDir:   tmpDir,
				Sample:      "test_sample",
				MinScore:    1.5,
			},
			wantErr: true,
			errMsg:  "min_score must be between 0 and 1",
		},
		{
			name: "Invalid num workers",
			config: &Config{
				PAFFile:     pafFile,
				MappingFile: mappingFile,
				OutputDir:   tmpDir,
				Sample:      "test_sample",
				NumWorkers:  0,
			},
			wantErr: true,
			errMsg:  "num_workers must be at least 1",
		},
		{
			name: "Invalid log level",
			config: &Config{
				PAFFile:     pafFile,
				MappingFile: mappingFile,
				OutputDir:   tmpDir,
				Sample:      "test_sample",
				LogLevel:    "invalid",
			},
			wantErr: true,
			errMsg:  "invalid log_level",
		},
		{
			name: "Invalid log format",
			config: &Config{
				PAFFile:     pafFile,
				MappingFile: mappingFile,
				OutputDir:   tmpDir,
				Sample:      "test_sample",
				LogLevel:    "info",
				LogFormat:   "xml",
			},
			wantErr: true,
			errMsg:  "log_format must be 'text' or 'json'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set defaults for fields not specified in test
			if tt.config.LogLevel == "" {
				tt.config.LogLevel = "info"
			}
			if tt.config.LogFormat == "" {
				tt.config.LogFormat = "text"
			}
			if tt.config.NumWorkers == 0 && !tt.wantErr {
				tt.config.NumWorkers = 4
			}
			if tt.config.ChannelBufferSize == 0 {
				tt.config.ChannelBufferSize = 100
			}

			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMergeFlags(t *testing.T) {
	cfg := DefaultConfig()

	flags := map[string]interface{}{
		"paf":            "/path/to/file.paf",
		"mapping":        "/path/to/mapping.txt",
		"output":         "/path/to/output",
		"sample":         "test_sample",
		"min-score":      0.85,
		"score-distance": 0.005,
		"num-workers":    8,
		"visualize-plot": true,
		"debug":          true,
	}

	cfg.MergeFlags(flags)

	assert.Equal(t, "/path/to/file.paf", cfg.PAFFile)
	assert.Equal(t, "/path/to/mapping.txt", cfg.MappingFile)
	assert.Equal(t, "/path/to/output", cfg.OutputDir)
	assert.Equal(t, "test_sample", cfg.Sample)
	assert.Equal(t, 0.85, cfg.MinScore)
	assert.Equal(t, 0.005, cfg.ScoreDistance)
	assert.Equal(t, 8, cfg.NumWorkers)
	assert.True(t, cfg.VisualizePlot)
	assert.True(t, cfg.Debug)
}

func TestCreateOutputDir(t *testing.T) {
	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "nested", "output", "dir")

	cfg := &Config{
		OutputDir: outputDir,
	}

	err := cfg.CreateOutputDir()
	require.NoError(t, err)

	// Check that directory was created
	info, err := os.Stat(outputDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestGetOutputPath(t *testing.T) {
	cfg := &Config{
		OutputDir: "/path/to/output",
	}

	path := cfg.GetOutputPath("results.txt")
	assert.Equal(t, filepath.Join("/path/to/output", "results.txt"), path)
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")
	configContent := `
min_score: 0.85
score_distance: 0.005
num_workers: 8
log_level: debug
log_format: json
channel_buffer_size: 200
`
	require.NoError(t, os.WriteFile(configFile, []byte(configContent), 0644))

	// Test loading config from file
	cfg, err := LoadConfig(configFile)
	require.NoError(t, err)

	assert.Equal(t, 0.85, cfg.MinScore)
	assert.Equal(t, 0.005, cfg.ScoreDistance)
	assert.Equal(t, 8, cfg.NumWorkers)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "json", cfg.LogFormat)
	assert.Equal(t, 200, cfg.ChannelBufferSize)
}

func TestLoadConfigWithEnvironmentVariables(t *testing.T) {
	// Set environment variables
	os.Setenv("MYPAF_MIN_SCORE", "0.75")
	os.Setenv("MYPAF_NUM_WORKERS", "16")
	defer os.Unsetenv("MYPAF_MIN_SCORE")
	defer os.Unsetenv("MYPAF_NUM_WORKERS")

	// Load config (no file, should use env vars and defaults)
	cfg, err := LoadConfig("")
	require.NoError(t, err)

	assert.Equal(t, 0.75, cfg.MinScore)
	assert.Equal(t, 16, cfg.NumWorkers)
	assert.Equal(t, "info", cfg.LogLevel) // Should use default
}