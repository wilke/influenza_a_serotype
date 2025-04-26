package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/config"
	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/log"
	"github.com/spf13/cobra"
)

var (
	// Config is the global configuration
	Config = config.NewDefaultConfig()
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "paf2serotypes",
	Short: "Process PAF files to identify influenza A serotypes",
	Long: `paf2serotypes is a tool for processing PAF (Pairwise Alignment Format) files
and identifying influenza A serotypes based on alignment scores.

It calculates Average Nucleotide Identity (ANI) and Alignment Fraction (AF)
metrics to assign serotypes to reads with confidence thresholds.`,
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Add persistent flags that are valid for all commands
	rootCmd.PersistentFlags().StringVar(&Config.MappingFile, "mapping", "", "Path to the mapping TSV file")
	rootCmd.PersistentFlags().StringVar(&Config.PafFile, "paf", "", "Path to the PAF file")
	rootCmd.PersistentFlags().StringVar(&Config.OutputDir, "output", ".", "Output directory")
	rootCmd.PersistentFlags().StringVar(&Config.SampleName, "sample", "sample", "Sample name")
	rootCmd.PersistentFlags().Float64Var(&Config.ScoreThreshold, "min-score", 0.8, "Minimum alignment score")
	rootCmd.PersistentFlags().Float64Var(&Config.AmbiguityThreshold, "ambiguity-threshold", 0.003, "Threshold for ambiguity detection")

	// Chunk size configuration
	rootCmd.PersistentFlags().IntVar(&Config.ChunkSize, "chunk-size", Config.ChunkSize, "Number of PAF entries per chunk")
	rootCmd.PersistentFlags().BoolVar(&Config.AdaptiveChunkSize, "adaptive-chunk", true, "Enable adaptive chunk sizing")
	rootCmd.PersistentFlags().IntVar(&Config.MinChunkSize, "min-chunk-size", 1000, "Minimum chunk size for adaptive sizing")
	rootCmd.PersistentFlags().IntVar(&Config.MaxChunkSize, "max-chunk-size", 10000, "Maximum chunk size for adaptive sizing")

	// Worker pool configuration
	rootCmd.PersistentFlags().IntVar(&Config.Workers, "workers", runtime.NumCPU(), "Number of worker goroutines")
	rootCmd.PersistentFlags().BoolVar(&Config.DynamicWorkerPool, "dynamic-workers", true, "Enable dynamic worker pool sizing")
	rootCmd.PersistentFlags().IntVar(&Config.MaxWorkers, "max-workers", runtime.NumCPU()*2, "Maximum number of workers for dynamic scaling")

	// Memory optimization configuration
	rootCmd.PersistentFlags().BoolVar(&Config.UseObjectPooling, "object-pooling", true, "Enable object pooling for memory optimization")

	// Visualization
	rootCmd.PersistentFlags().BoolVar(&Config.EnableVisualization, "visualize", true, "Enable visualization generation")

	// Add log level flag
	var logLevelStr string
	rootCmd.PersistentFlags().StringVar(&logLevelStr, "log-level", "info", "Log level (debug, info, warn, error)")

	// Parse log level
	cobra.OnInitialize(func() {
		level, err := log.ParseLogLevel(logLevelStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v, using info level\n", err)
			level = log.LevelInfo
		}
		Config.LogLevel = level
	})

	// Add commands
	rootCmd.AddCommand(processCmd)
	rootCmd.AddCommand(benchmarkCmd)
	rootCmd.AddCommand(compareCmd)
}

// validateConfig validates the configuration
func validateConfig(cmd *cobra.Command, args []string) error {
	return Config.Validate()
}
