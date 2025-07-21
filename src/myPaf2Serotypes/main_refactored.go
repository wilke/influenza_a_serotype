// Example of refactored main.go using the new packages
// This demonstrates how to integrate the new library packages

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/config"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/errors"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/models"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/processor"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/utils"
)

// Application holds all dependencies
type Application struct {
	config *config.Config
	logger utils.Logger
}

// NewApplication creates a new application instance
func NewApplication(cfg *config.Config, logger utils.Logger) *Application {
	return &Application{
		config: cfg,
		logger: logger,
	}
}

// Run executes the main application logic
func (app *Application) Run(ctx context.Context) error {
	// Log start of processing
	app.logger.Info("Starting myPaf2Serotypes",
		utils.String("version", "2.0.0"),
		utils.String("paf_file", app.config.PAFFile),
		utils.String("mapping_file", app.config.MappingFile),
		utils.String("output_dir", app.config.OutputDir),
	)

	// Create output directory
	if err := app.config.CreateOutputDir(); err != nil {
		return errors.IOError("main", "create_output_dir", "failed to create output directory", err)
	}

	// Create and run the pipeline
	pipeline, err := processor.NewDefaultPipeline(app.config, app.logger)
	if err != nil {
		return errors.Wrap(err, errors.ErrTypeConfiguration, "main", "create_pipeline", 
			"failed to create processing pipeline")
	}
	defer pipeline.Close()

	// Run the pipeline
	if err := pipeline.Run(ctx); err != nil {
		return errors.Wrap(err, errors.ErrTypeProcessing, "main", "run_pipeline",
			"pipeline execution failed")
	}

	// Get and log metrics
	metrics := pipeline.GetMetrics()
	app.logger.Info("Processing complete",
		utils.Int64("records_processed", metrics.RecordsProcessed),
		utils.Int64("scores_calculated", metrics.ScoresCalculated),
		utils.Int64("assignments_made", metrics.AssignmentsMade),
		utils.Float64("duration_seconds", metrics.ProcessingDuration),
		utils.Int64("errors", metrics.ErrorCount),
	)

	return nil
}

// parseFlags parses command line flags and returns them as a map
func parseFlags() map[string]interface{} {
	flags := make(map[string]interface{})

	// Define flags
	pafFile := flag.String("paf", "", "PAF file to process")
	mappingFile := flag.String("mapping", "", "Mapping file")
	outputDir := flag.String("output", "", "Output directory")
	sample := flag.String("sample", "", "Sample name")
	minScore := flag.Float64("min-score", 0.9, "Minimum alignment score")
	scoreDistance := flag.Float64("score-distance", 0.003, "Score distance for ambiguous assignments")
	numWorkers := flag.Int("num-workers", 4, "Number of worker goroutines")
	visualizePlot := flag.Bool("visualize-plot", false, "Generate visualization plots")
	debug := flag.Bool("debug", false, "Enable debug mode")
	configFile := flag.String("config", "", "Configuration file path")

	flag.Parse()

	// Store non-empty values
	if *pafFile != "" {
		flags["paf"] = *pafFile
	}
	if *mappingFile != "" {
		flags["mapping"] = *mappingFile
	}
	if *outputDir != "" {
		flags["output"] = *outputDir
	}
	if *sample != "" {
		flags["sample"] = *sample
	}
	flags["min-score"] = *minScore
	flags["score-distance"] = *scoreDistance
	flags["num-workers"] = *numWorkers
	flags["visualize-plot"] = *visualizePlot
	flags["debug"] = *debug
	if *configFile != "" {
		flags["config"] = *configFile
	}

	return flags
}

func main() {
	// Parse command line flags
	flags := parseFlags()

	// Load configuration
	configFile, _ := flags["config"].(string)
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Merge command line flags into configuration
	cfg.MergeFlags(flags)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := utils.NewLogger(cfg.LogLevel, cfg.LogFormat, cfg.Debug)

	// Create application
	app := NewApplication(cfg, logger)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Run application in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- app.Run(ctx)
	}()

	// Wait for completion or signal
	select {
	case err := <-errChan:
		if err != nil {
			logger.Error("Application failed", utils.Err(err))
			os.Exit(1)
		}
	case sig := <-sigChan:
		logger.Info("Received signal, shutting down", utils.String("signal", sig.String()))
		cancel()
		
		// Wait for graceful shutdown with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()
		
		select {
		case <-shutdownCtx.Done():
			logger.Error("Shutdown timeout exceeded, forcing exit")
			os.Exit(1)
		case err := <-errChan:
			if err != nil && !errors.IsType(err, errors.ErrTypeCanceled) {
				logger.Error("Application failed during shutdown", utils.Err(err))
				os.Exit(1)
			}
		}
	}

	logger.Info("Application shutdown complete")
}