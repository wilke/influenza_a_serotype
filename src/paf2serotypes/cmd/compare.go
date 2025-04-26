package cmd

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/log"
	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/metrics"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cobra"
)

// ComparisonResult represents the result of a comparison between Go and R implementations
type ComparisonResult struct {
	// Performance metrics
	GoRuntime  time.Duration     `json:"go_runtime"`
	RRuntime   time.Duration     `json:"r_runtime"`
	Speedup    float64           `json:"speedup"`
	GoMemory   map[string]uint64 `json:"go_memory"`
	RMemory    map[string]uint64 `json:"r_memory"`
	GoCPUUsage float64           `json:"go_cpu_usage"`
	RCPUUsage  float64           `json:"r_cpu_usage"`
	GoIOStats  map[string]uint64 `json:"go_io_stats"`
	RIOStats   map[string]uint64 `json:"r_io_stats"`

	// Output comparison
	OutputMatch         bool           `json:"output_match"`
	GoLineCount         int            `json:"go_line_count"`
	RLineCount          int            `json:"r_line_count"`
	SerotypeCounts      map[string]int `json:"serotype_counts"`
	UniqueSerotypes     int            `json:"unique_serotypes"`
	TotalSerotypes      int            `json:"total_serotypes"`
	SerotypeDifferences map[string]int `json:"serotype_differences"`
}

// compareCmd represents the compare command
var compareCmd = &cobra.Command{
	Use:   "compare",
	Short: "Compare with R implementation",
	Long: `Compare the Go implementation with the R implementation.

This command runs both the Go and R implementations on the same input
and compares the results for both correctness and performance.

It generates a detailed report showing:
1. Performance metrics (runtime, memory usage, CPU utilization, I/O operations)
2. Output comparison (matching lines, differences, serotype assignments)
3. Visualization of the comparison results`,
	PreRunE: validateConfig,
	RunE:    runCompare,
}

func init() {
	// Add compare-specific flags
	compareCmd.Flags().String("r-script", "", "Path to the R script (required)")
	compareCmd.Flags().String("r-output", "", "Output directory for R results")
	compareCmd.Flags().String("report-file", "comparison_report.json", "Output file for comparison report")
	compareCmd.Flags().Bool("verbose", false, "Enable verbose output")

	// Mark required flags
	compareCmd.MarkFlagRequired("r-script")
}

// collectProcessMetrics collects metrics for a running process
func collectProcessMetricsForCompare(pid int32) (map[string]uint64, float64, map[string]uint64, error) {
	proc, err := process.NewProcess(pid)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to get process info: %w", err)
	}

	// Memory metrics
	memInfo, err := proc.MemoryInfo()
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to get memory info: %w", err)
	}
	memoryUsage := map[string]uint64{
		"rss":  memInfo.RSS,  // Resident Set Size
		"vms":  memInfo.VMS,  // Virtual Memory Size
		"swap": memInfo.Swap, // Swap
	}

	// CPU metrics
	cpuPercent, err := proc.CPUPercent()
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to get CPU info: %w", err)
	}

	// I/O metrics
	ioCounters, err := proc.IOCounters()
	if err != nil {
		return memoryUsage, cpuPercent, nil, fmt.Errorf("failed to get I/O info: %w", err)
	}
	ioStats := map[string]uint64{
		"read_count":  ioCounters.ReadCount,
		"write_count": ioCounters.WriteCount,
		"read_bytes":  ioCounters.ReadBytes,
		"write_bytes": ioCounters.WriteBytes,
	}

	return memoryUsage, cpuPercent, ioStats, nil
}

// compareOutputFiles compares the output files from Go and R implementations
func compareOutputFiles(goFile, rFile string) (int, int, bool, error) {
	// Read Go file
	goLines, err := readLines(goFile)
	if err != nil {
		return 0, 0, false, fmt.Errorf("failed to read Go file: %w", err)
	}

	// Read R file
	rLines, err := readLines(rFile)
	if err != nil {
		return 0, 0, false, fmt.Errorf("failed to read R file: %w", err)
	}

	// Get line counts
	goLineCount := len(goLines)
	rLineCount := len(rLines)

	// Check if line counts match
	outputMatch := goLineCount == rLineCount

	if goLineCount != rLineCount {
		log.Warn("Line count mismatch: Go=%d, R=%d", goLineCount, rLineCount)
		outputMatch = false
	}

	return goLineCount, rLineCount, outputMatch, nil
}

// readLines reads a file and returns its lines
func readLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// compareSerotypeCounts compares serotype counts between Go and R implementations
func compareSerotypeCounts(goFile, rFile string) (map[string]int, map[string]int, int, int, error) {
	// Read Go file
	goSerotypes, err := readSerotypeCounts(goFile)
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("failed to read Go serotype counts: %w", err)
	}

	// Read R file
	rSerotypes, err := readSerotypeCounts(rFile)
	if err != nil {
		return nil, nil, 0, 0, fmt.Errorf("failed to read R serotype counts: %w", err)
	}

	// Calculate differences
	differences := make(map[string]int)

	// Check serotypes in Go output
	for serotype, goCount := range goSerotypes {
		if rCount, ok := rSerotypes[serotype]; ok {
			diff := goCount - rCount
			if diff != 0 {
				differences[serotype] = diff
			}
		} else {
			differences[serotype] = goCount
		}
	}

	// Check serotypes in R output that are not in Go output
	for serotype, rCount := range rSerotypes {
		if _, ok := goSerotypes[serotype]; !ok {
			differences[serotype] = -rCount
		}
	}

	// Calculate unique serotypes (number of keys in the map)
	uniqueSerotypes := len(goSerotypes)

	// Calculate total serotypes (sum of all counts)
	totalSerotypes := 0
	for _, count := range goSerotypes {
		totalSerotypes += count
	}

	return goSerotypes, differences, uniqueSerotypes, totalSerotypes, nil
}

// readSerotypeCounts reads serotype counts from a file
func readSerotypeCounts(filePath string) (map[string]int, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", filePath)
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Parse as CSV
	reader := csv.NewReader(file)
	reader.Comma = '\t' // TSV format

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	// Count serotypes
	serotypeCounts := make(map[string]int)

	// Skip header
	for i := 1; i < len(records); i++ {
		if len(records[i]) >= 3 { // Ensure we have enough columns
			serotype := records[i][2] // Assuming serotype is in the 3rd column
			serotypeCounts[serotype]++
		}
	}

	return serotypeCounts, nil
}

// generateComparisonReport generates a detailed comparison report
func generateComparisonReport(result *ComparisonResult, reportFile string) error {
	// Create report JSON
	reportJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(reportFile, reportJSON, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	return nil
}

func runCompare(cmd *cobra.Command, args []string) error {
	// Get compare-specific flags
	rScript, _ := cmd.Flags().GetString("r-script")
	rOutput, _ := cmd.Flags().GetString("r-output")
	reportFile, _ := cmd.Flags().GetString("report-file")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Validate R script
	if rScript == "" {
		return fmt.Errorf("R script path is required")
	}
	if _, err := os.Stat(rScript); os.IsNotExist(err) {
		return fmt.Errorf("R script does not exist: %s", rScript)
	}

	// Create subdirectories for Go and R outputs
	goOutput := filepath.Join(Config.OutputDir, "go")
	if err := os.MkdirAll(goOutput, 0755); err != nil {
		return fmt.Errorf("failed to create Go output directory: %w", err)
	}

	// Set default R output directory
	if rOutput == "" {
		rOutput = filepath.Join(Config.OutputDir, "R")
	}

	// Create R output directory
	if err := os.MkdirAll(rOutput, 0755); err != nil {
		return fmt.Errorf("failed to create R output directory: %w", err)
	}

	log.Info("Starting comparison between Go and R implementations")
	log.Info("Go configuration:")
	log.Info(Config.String())
	log.Info("  Output: %s", goOutput)
	log.Info("R configuration:")
	log.Info("  Script: %s", rScript)
	log.Info("  Output: %s", rOutput)

	// Initialize comparison result
	result := &ComparisonResult{
		GoMemory:  make(map[string]uint64),
		RMemory:   make(map[string]uint64),
		GoIOStats: make(map[string]uint64),
		RIOStats:  make(map[string]uint64),
	}

	// Run Go implementation
	log.Info("Running Go implementation")
	goStartTime := time.Now()
	metrics.Start("go_implementation")

	// Get current process for metrics
	goPid := int32(os.Getpid())
	_, _, goIOBefore, err := collectProcessMetricsForCompare(goPid)
	if err != nil {
		log.Warn("Failed to collect Go metrics before: %v", err)
	}

	// Temporarily modify Config.OutputDir to use the Go subdirectory
	originalOutputDir := Config.OutputDir
	Config.OutputDir = goOutput

	if err := runProcess(cmd, args); err != nil {
		// Restore original output directory
		Config.OutputDir = originalOutputDir
		return fmt.Errorf("Go implementation failed: %w", err)
	}

	// Restore original output directory
	Config.OutputDir = originalOutputDir

	goMemAfter, goCPUAfter, goIOAfter, err := collectProcessMetricsForCompare(goPid)
	if err != nil {
		log.Warn("Failed to collect Go metrics after: %v", err)
	}

	metrics.Stop("go_implementation")
	goDuration := time.Since(goStartTime)
	log.Info("Go implementation completed in %v", goDuration)

	// Calculate Go metrics
	result.GoRuntime = goDuration
	result.GoMemory = goMemAfter
	result.GoCPUUsage = goCPUAfter

	// Calculate I/O differences
	if goIOBefore != nil && goIOAfter != nil {
		result.GoIOStats = map[string]uint64{
			"read_count":  goIOAfter["read_count"] - goIOBefore["read_count"],
			"write_count": goIOAfter["write_count"] - goIOBefore["write_count"],
			"read_bytes":  goIOAfter["read_bytes"] - goIOBefore["read_bytes"],
			"write_bytes": goIOAfter["write_bytes"] - goIOBefore["write_bytes"],
		}
	}

	// Run R implementation
	log.Info("Running R implementation")
	rStartTime := time.Now()

	// Prepare R command
	rCmd := exec.Command("Rscript", rScript, Config.MappingFile, Config.PafFile, Config.SampleName, rOutput, fmt.Sprintf("%f", Config.ScoreThreshold))

	// Capture output if verbose
	if verbose {
		rCmd.Stdout = os.Stdout
		rCmd.Stderr = os.Stderr
	}

	// Start the command
	if err := rCmd.Start(); err != nil {
		return fmt.Errorf("failed to start R script: %w", err)
	}

	// Get process ID for metrics collection
	rPid := int32(rCmd.Process.Pid)

	// Collect metrics at intervals
	var maxRMemory map[string]uint64
	var maxRCPU float64
	var finalRIOStats map[string]uint64

	done := make(chan error)
	go func() {
		done <- rCmd.Wait()
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rMem, rCPU, rIO, err := collectProcessMetricsForCompare(rPid)
			if err != nil {
				log.Warn("Failed to collect R metrics: %v", err)
				continue
			}

			if maxRMemory == nil || rMem["rss"] > maxRMemory["rss"] {
				maxRMemory = rMem
			}

			if rCPU > maxRCPU {
				maxRCPU = rCPU
			}

			finalRIOStats = rIO

		case err := <-done:
			if err != nil {
				return fmt.Errorf("R implementation failed: %w", err)
			}

			// Stop timing
			rDuration := time.Since(rStartTime)
			log.Info("R implementation completed in %v", rDuration)

			// Set R metrics
			result.RRuntime = rDuration
			result.RMemory = maxRMemory
			result.RCPUUsage = maxRCPU
			result.RIOStats = finalRIOStats

			// Calculate speedup
			result.Speedup = float64(rDuration) / float64(goDuration)

			// Exit the loop
			goto CompareOutputs
		}
	}

CompareOutputs:
	// Compare results
	log.Info("Comparing output files")
	goSummaryFile := filepath.Join(goOutput, fmt.Sprintf("%s_read_summary.tsv", Config.SampleName))
	rSummaryFile := filepath.Join(rOutput, fmt.Sprintf("%s_read_summary.tsv", Config.SampleName))

	// Check if files exist
	if _, err := os.Stat(goSummaryFile); os.IsNotExist(err) {
		return fmt.Errorf("Go summary file does not exist: %s", goSummaryFile)
	}
	if _, err := os.Stat(rSummaryFile); os.IsNotExist(err) {
		return fmt.Errorf("R summary file does not exist: %s", rSummaryFile)
	}

	// Compare output files
	goLineCount, rLineCount, outputMatch, err := compareOutputFiles(goSummaryFile, rSummaryFile)
	if err != nil {
		log.Warn("Failed to compare output files: %v", err)
	} else {
		result.GoLineCount = goLineCount
		result.RLineCount = rLineCount
		result.OutputMatch = outputMatch

		log.Info("Output comparison:")
		log.Info("  Go line count: %d", goLineCount)
		log.Info("  R line count: %d", rLineCount)

		if result.OutputMatch {
			log.Info("  Line counts match!")
		} else {
			log.Warn("  Line counts differ!")
		}
	}

	// Compare serotype counts
	goSerotypeCounts, serotypeDifferences, uniqueSerotypes, totalSerotypes, err := compareSerotypeCounts(goSummaryFile, rSummaryFile)
	if err != nil {
		log.Warn("Failed to compare serotype counts: %v", err)
	} else {
		result.SerotypeCounts = goSerotypeCounts
		result.SerotypeDifferences = serotypeDifferences
		result.UniqueSerotypes = uniqueSerotypes
		result.TotalSerotypes = totalSerotypes

		log.Info("Serotype comparison:")
		log.Info("  Unique serotypes: %d", uniqueSerotypes)
		log.Info("  Total serotypes: %d", totalSerotypes)
		log.Info("  Serotypes with differences: %d", len(serotypeDifferences))

		if len(serotypeDifferences) > 0 {
			log.Warn("  Serotype differences detected:")
			for serotype, diff := range serotypeDifferences {
				log.Warn("    %s: %+d", serotype, diff)
			}
		} else {
			log.Info("  Serotype counts match exactly!")
		}
	}

	// Report performance comparison
	log.Info("Performance comparison:")
	log.Info("  Go implementation: %v", result.GoRuntime)
	log.Info("  R implementation: %v", result.RRuntime)
	log.Info("  Speedup: %.2fx", result.Speedup)

	if result.GoMemory != nil && result.RMemory != nil {
		goRSS := float64(result.GoMemory["rss"]) / 1024 / 1024
		rRSS := float64(result.RMemory["rss"]) / 1024 / 1024
		memoryImprovement := rRSS / goRSS

		log.Info("  Go memory usage: %.2f MB", goRSS)
		log.Info("  R memory usage: %.2f MB", rRSS)
		log.Info("  Memory improvement: %.2fx", memoryImprovement)
	}

	log.Info("  Go CPU usage: %.2f%%", result.GoCPUUsage)
	log.Info("  R CPU usage: %.2f%%", result.RCPUUsage)

	if result.GoIOStats != nil && result.RIOStats != nil {
		log.Info("  Go I/O operations: %d reads, %d writes",
			result.GoIOStats["read_count"], result.GoIOStats["write_count"])
		log.Info("  R I/O operations: %d reads, %d writes",
			result.RIOStats["read_count"], result.RIOStats["write_count"])
	}

	// Generate comparison report in the main output directory
	reportFilePath := filepath.Join(Config.OutputDir, reportFile)
	if err := generateComparisonReport(result, reportFilePath); err != nil {
		log.Warn("Failed to generate comparison report: %v", err)
	} else {
		log.Info("Comparison report written to %s", reportFilePath)
	}

	return nil
}
