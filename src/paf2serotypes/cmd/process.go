package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/io"
	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/log"
	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/metrics"
	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/pipeline"
	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/processor"
	"github.com/spf13/cobra"
)

// processCmd represents the process command
var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Process PAF files to identify influenza A serotypes",
	Long: `Process PAF files to identify influenza A serotypes based on alignment scores.

This command reads a PAF file and a mapping file, calculates alignment metrics,
assigns serotypes to reads, and generates output files.`,
	PreRunE: validateConfig,
	RunE:    runProcess,
}

func runProcess(cmd *cobra.Command, args []string) error {
	// Start timing
	startTime := time.Now()
	metrics.Start("total")

	// Setup logging
	if err := Config.SetupLogging(); err != nil {
		return fmt.Errorf("failed to setup logging: %w", err)
	}

	log.Info("Starting paf2serotypes process")
	log.Info(Config.String())

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(Config.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create readers
	log.Info("Reading mapping file: %s", Config.MappingFile)
	metrics.Start("read_mapping")
	mappingReader := io.NewMappingReader(Config.MappingFile)
	mappingData, err := mappingReader.ReadChunk(0)
	if err != nil {
		return fmt.Errorf("failed to read mapping file: %w", err)
	}
	metrics.Stop("read_mapping")
	metrics.LogMemoryUsage("after_read_mapping")

	// Create PAF reader with options
	log.Info("Processing PAF file: %s", Config.PafFile)

	// Configure reader options based on config
	readerOptions := io.DefaultPafReaderOptions()
	readerOptions.AdaptiveChunk = Config.AdaptiveChunkSize
	readerOptions.MinChunkSize = Config.MinChunkSize
	readerOptions.MaxChunkSize = Config.MaxChunkSize
	readerOptions.UseRecordPool = Config.UseObjectPooling

	pafReader, err := io.NewPafReaderWithOptions(Config.PafFile, readerOptions)
	if err != nil {
		return fmt.Errorf("failed to create PAF reader: %w", err)
	}

	// Create processor
	proc := processor.NewPafProcessor(Config.ScoreThreshold, Config.AmbiguityThreshold)

	// Create writers
	resultWriter, err := io.NewResultWriter(Config.OutputDir, Config.SampleName)
	if err != nil {
		return fmt.Errorf("failed to create result writer: %w", err)
	}

	// Create visualization writer if enabled
	var vizWriter io.Writer
	if Config.EnableVisualization {
		var err error
		vizWriter, err = io.NewVisualizationWriter(Config.OutputDir, Config.SampleName)
		if err != nil {
			return fmt.Errorf("failed to create visualization writer: %w", err)
		}
		log.Info("Visualization enabled")
	} else {
		log.Info("Visualization disabled")
	}

	// Create pipeline with dynamic worker pool if enabled
	var p *pipeline.Pipeline
	if Config.DynamicWorkerPool {
		log.Info("Using dynamic worker pool with base %d workers, max %d workers",
			Config.Workers, Config.MaxWorkers)
		p = pipeline.NewPipelineWithOptions(pipeline.PipelineOptions{
			Workers:        Config.Workers,
			MaxWorkers:     Config.MaxWorkers,
			DynamicWorkers: true,
		})
	} else {
		log.Info("Using fixed worker pool with %d workers", Config.Workers)
		p = pipeline.NewPipeline(Config.Workers)
	}

	// Configure chunk processor with options
	chunkProcessorOptions := processor.ChunkProcessorOptions{
		Workers:          Config.Workers,
		MaxWorkers:       Config.MaxWorkers,
		DynamicWorkers:   Config.DynamicWorkerPool,
		UseObjectPooling: Config.UseObjectPooling,
	}

	// Create chunk processor with options
	chunkProcessor := processor.NewChunkProcessorWithOptions(proc, chunkProcessorOptions)

	// Add pipeline stages
	p.AddStage(pipeline.NewReadStage(pafReader, Config.ChunkSize))
	p.AddStage(pipeline.NewProcessStageWithProcessor(chunkProcessor, mappingData))
	p.AddStage(pipeline.NewMergeStage())
	p.AddStage(pipeline.NewWriteStage(resultWriter))

	// Add visualization stage if enabled
	if Config.EnableVisualization && vizWriter != nil {
		p.AddStage(pipeline.NewWriteStage(vizWriter))
	}

	// Run pipeline
	log.Info("Running pipeline with chunk size %d (adaptive: %v, min: %d, max: %d)",
		Config.ChunkSize, Config.AdaptiveChunkSize, Config.MinChunkSize, Config.MaxChunkSize)
	metrics.Start("pipeline")
	if err := p.Run(ctx); err != nil {
		return fmt.Errorf("pipeline failed: %w", err)
	}
	metrics.Stop("pipeline")
	metrics.LogMemoryUsage("after_pipeline")

	// Stop timing
	metrics.Stop("total")
	duration := time.Since(startTime)

	// Log results
	log.Info("Processing completed in %v", duration)
	log.Info("Results written to %s", filepath.Join(Config.OutputDir, fmt.Sprintf("%s_read_summary.tsv", Config.SampleName)))
	log.Info("Serotype counts written to %s", filepath.Join(Config.OutputDir, fmt.Sprintf("%s_serotype_counts.csv", Config.SampleName)))

	if Config.EnableVisualization {
		log.Info("Visualization written to %s", filepath.Join(Config.OutputDir, fmt.Sprintf("%s_read_serotype_assignment.pdf", Config.SampleName)))
	}

	// Print metrics report
	report := metrics.Report()
	log.Info("Performance metrics:")
	for category, metrics := range report {
		log.Info("  %s:", category)
		switch m := metrics.(type) {
		case map[string]string:
			for k, v := range m {
				log.Info("    %s: %s", k, v)
			}
		case map[string]int64:
			for k, v := range m {
				log.Info("    %s: %d", k, v)
			}
		case map[string]float64:
			for k, v := range m {
				log.Info("    %s: %.2f", k, v)
			}
		}
	}

	return nil
}
