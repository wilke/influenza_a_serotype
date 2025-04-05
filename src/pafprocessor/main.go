package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/influenza_a_serotype/lib/go/pafprocessor"
)

func main() {
	// Parse command-line arguments
	mappingFile := flag.String("mapping", "", "Path to the mapping TSV file")
	pafFile := flag.String("paf", "", "Path to the PAF file")
	minAlignmentScore := flag.Float64("min-score", 0.8, "Minimum alignment score")
	paired := flag.Bool("paired", false, "Group by strand if true")
	outputDir := flag.String("output", ".", "Output directory")
	sampleName := flag.String("sample", "sample", "Sample name")
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	numWorkers := flag.Int("workers", runtime.NumCPU(), "Number of worker goroutines")
	chunkSize := flag.Int("chunk-size", 1000, "Number of PAF entries per chunk")
	cpuProfile := flag.String("cpu-profile", "", "Write CPU profile to file")
	memProfile := flag.String("mem-profile", "", "Write memory profile to file")
	logFile := flag.String("log-file", "", "Path to log file")
	flag.Parse()

	// Validate arguments
	if *mappingFile == "" {
		fmt.Println("Error: mapping file is required")
		flag.Usage()
		os.Exit(1)
	}
	if *pafFile == "" {
		fmt.Println("Error: PAF file is required")
		flag.Usage()
		os.Exit(1)
	}

	// Setup CPU profiling
	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			fmt.Printf("Error creating CPU profile: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Printf("Error starting CPU profile: %v\n", err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
	}

	// Setup logging
	logLevelValue, err := pafprocessor.ParseLogLevel(*logLevel)
	if err != nil {
		fmt.Printf("Error parsing log level: %v\n", err)
		os.Exit(1)
	}

	logConfig := pafprocessor.LoggerConfig{
		Level:      logLevelValue,
		OutputPath: *logFile,
	}

	if err := pafprocessor.InitLogger(logConfig); err != nil {
		fmt.Printf("Error initializing logger: %v\n", err)
		os.Exit(1)
	}

	// Log startup information
	pafprocessor.Logger.Infof("Starting PAF processor")
	pafprocessor.Logger.Infof("Mapping file: %s", *mappingFile)
	pafprocessor.Logger.Infof("PAF file: %s", *pafFile)
	pafprocessor.Logger.Infof("Minimum alignment score: %f", *minAlignmentScore)
	pafprocessor.Logger.Infof("Paired: %t", *paired)
	pafprocessor.Logger.Infof("Output directory: %s", *outputDir)
	pafprocessor.Logger.Infof("Sample name: %s", *sampleName)
	pafprocessor.Logger.Infof("Number of workers: %d", *numWorkers)
	pafprocessor.Logger.Infof("Chunk size: %d", *chunkSize)

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		pafprocessor.Logger.Errorf("Error creating output directory: %v", err)
		os.Exit(1)
	}

	// Start processing
	startTime := time.Now()

	// Read mapping file
	mapping, err := pafprocessor.ReadMappingFile(*mappingFile)
	if err != nil {
		pafprocessor.Logger.Errorf("Error reading mapping file: %v", err)
		os.Exit(1)
	}

	// Read PAF file
	chunks, err := pafprocessor.ReadPAFFile(*pafFile, *chunkSize)
	if err != nil {
		pafprocessor.Logger.Errorf("Error reading PAF file: %v", err)
		os.Exit(1)
	}

	// Process chunks
	summaries, err := pafprocessor.ProcessAllChunks(chunks, mapping, *minAlignmentScore, *paired, *numWorkers)
	if err != nil {
		pafprocessor.Logger.Errorf("Error processing chunks: %v", err)
		os.Exit(1)
	}

	// Generate results
	if err := pafprocessor.GenerateResults(summaries, *outputDir, *sampleName); err != nil {
		pafprocessor.Logger.Errorf("Error generating results: %v", err)
		os.Exit(1)
	}

	// Log total execution time
	totalTime := time.Since(startTime)
	pafprocessor.Logger.Infof("Total execution time: %s", totalTime)

	// Write memory profile
	if *memProfile != "" {
		f, err := os.Create(*memProfile)
		if err != nil {
			pafprocessor.Logger.Errorf("Error creating memory profile: %v", err)
			os.Exit(1)
		}
		defer f.Close()
		runtime.GC() // Get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			pafprocessor.Logger.Errorf("Error writing memory profile: %v", err)
			os.Exit(1)
		}
	}

	// Print performance metrics
	pafprocessor.PrintPerformanceMetrics()

	// Write performance summary
	perfSummaryFile := filepath.Join(*outputDir, fmt.Sprintf("%s_performance_summary.txt", *sampleName))
	perfSummary, err := os.Create(perfSummaryFile)
	if err != nil {
		pafprocessor.Logger.Errorf("Error creating performance summary file: %v", err)
		os.Exit(1)
	}
	defer perfSummary.Close()

	fmt.Fprintf(perfSummary, "PAF Processor Performance Summary\n")
	fmt.Fprintf(perfSummary, "================================\n\n")
	fmt.Fprintf(perfSummary, "Sample: %s\n", *sampleName)
	fmt.Fprintf(perfSummary, "PAF file: %s\n", *pafFile)
	fmt.Fprintf(perfSummary, "Mapping file: %s\n", *mappingFile)
	fmt.Fprintf(perfSummary, "Number of workers: %d\n", *numWorkers)
	fmt.Fprintf(perfSummary, "Chunk size: %d\n\n", *chunkSize)
	fmt.Fprintf(perfSummary, "Total execution time: %s\n\n", totalTime)

	fmt.Fprintf(perfSummary, "Performance Metrics:\n")
	metrics := pafprocessor.GetPerformanceMetrics()
	for operation, duration := range metrics {
		fmt.Fprintf(perfSummary, "  %s: %s\n", operation, duration)
	}

	pafprocessor.Logger.Infof("Performance summary written to %s", perfSummaryFile)
	pafprocessor.Logger.Infof("Processing complete")
}
