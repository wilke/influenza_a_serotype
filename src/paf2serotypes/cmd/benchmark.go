package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/log"
	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/metrics"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/spf13/cobra"
)

// BenchmarkResult represents the result of a benchmark run
type BenchmarkResult struct {
	Implementation string            `json:"implementation"`
	Runtime        time.Duration     `json:"runtime"`
	MemoryUsage    map[string]uint64 `json:"memory_usage"`
	CPUUsage       float64           `json:"cpu_usage"`
	IOStats        map[string]uint64 `json:"io_stats"`
}

// benchmarkCmd represents the benchmark command
var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Run benchmarks for paf2serotypes",
	Long: `Run benchmarks for paf2serotypes to measure performance.

This command runs both the Go and R implementations on the same input data
and reports detailed performance metrics including runtime, memory usage,
CPU utilization, and I/O operations.`,
	PreRunE: validateConfig,
	RunE:    runBenchmark,
}

func init() {
	// Add benchmark-specific flags
	benchmarkCmd.Flags().Int("iterations", 3, "Number of benchmark iterations")
	benchmarkCmd.Flags().Bool("profile", false, "Enable CPU and memory profiling")
	benchmarkCmd.Flags().String("profile-dir", ".", "Directory for profiling output")
	benchmarkCmd.Flags().String("r-script", "", "Path to the R script (required)")
	benchmarkCmd.Flags().String("r-output", "", "Output directory for R results")
	benchmarkCmd.Flags().String("report-file", "benchmark_report.json", "Output file for benchmark report")
	benchmarkCmd.Flags().Bool("verbose", false, "Enable verbose output")

	// Mark required flags
	benchmarkCmd.MarkFlagRequired("r-script")
}

// collectProcessMetrics collects metrics for a running process
func collectProcessMetrics(pid int32) (map[string]uint64, float64, map[string]uint64, error) {
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
		log.Warn("Failed to get I/O info: %v", err)
		ioStats := map[string]uint64{
			"read_count":  0,
			"write_count": 0,
			"read_bytes":  0,
			"write_bytes": 0,
		}
		return memoryUsage, cpuPercent, ioStats, nil
	}

	ioStats := map[string]uint64{
		"read_count":  ioCounters.ReadCount,
		"write_count": ioCounters.WriteCount,
		"read_bytes":  ioCounters.ReadBytes,
		"write_bytes": ioCounters.WriteBytes,
	}

	return memoryUsage, cpuPercent, ioStats, nil
}

// runGoImplementation runs the Go implementation and collects metrics
func runGoImplementation(cmd *cobra.Command, args []string) (*BenchmarkResult, error) {
	log.Info("Running Go implementation benchmark")

	// Start timing
	startTime := time.Now()
	metrics.Start("go_implementation")

	// Run process command
	if err := runProcess(cmd, args); err != nil {
		return nil, fmt.Errorf("Go implementation failed: %w", err)
	}

	// Stop timing
	metrics.Stop("go_implementation")
	duration := time.Since(startTime)

	// Get memory metrics from the metrics package
	report := metrics.Report()
	gauges := report["gauges"].(map[string]float64)

	memoryUsage := map[string]uint64{
		"alloc":       uint64(gauges["after_pipeline.memory.alloc"] * 1024 * 1024),
		"total_alloc": uint64(gauges["after_pipeline.memory.total_alloc"] * 1024 * 1024),
		"sys":         uint64(gauges["after_pipeline.memory.sys"] * 1024 * 1024),
		"num_gc":      uint64(gauges["after_pipeline.memory.num_gc"]),
	}

	// Get CPU usage
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		log.Warn("Failed to get CPU usage: %v", err)
	}

	var cpuUsage float64
	if len(cpuPercent) > 0 {
		cpuUsage = cpuPercent[0]
	}

	// Create result
	result := &BenchmarkResult{
		Implementation: "Go",
		Runtime:        duration,
		MemoryUsage:    memoryUsage,
		CPUUsage:       cpuUsage,
		IOStats:        map[string]uint64{}, // Not available directly
	}

	// Ensure CPUUsage is not NaN
	if math.IsNaN(result.CPUUsage) {
		result.CPUUsage = 0
	}

	return result, nil
}

// runRImplementation runs the R implementation and collects metrics
func runRImplementation(cmd *cobra.Command, args []string) (*BenchmarkResult, error) {
	// Get R script path
	rScript, _ := cmd.Flags().GetString("r-script")
	if rScript == "" {
		return nil, fmt.Errorf("R script path is required")
	}

	// Validate R script
	if _, err := os.Stat(rScript); os.IsNotExist(err) {
		return nil, fmt.Errorf("R script does not exist: %s", rScript)
	}

	// Set default R output directory
	rOutput, _ := cmd.Flags().GetString("r-output")
	if rOutput == "" {
		rOutput = filepath.Join(Config.OutputDir, "r_output")
	}

	// Create R output directory
	if err := os.MkdirAll(rOutput, 0755); err != nil {
		return nil, fmt.Errorf("failed to create R output directory: %w", err)
	}

	log.Info("Running R implementation benchmark")
	log.Info("  Script: %s", rScript)
	log.Info("  Output: %s", rOutput)

	// Start timing
	startTime := time.Now()

	// Prepare R command
	rCmd := exec.Command("Rscript", rScript, Config.MappingFile, Config.PafFile, Config.SampleName, rOutput, fmt.Sprintf("%f", Config.ScoreThreshold))

	// Start the command
	if err := rCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start R script: %w", err)
	}

	// Get process ID for metrics collection
	pid := rCmd.Process.Pid

	// Collect metrics at intervals
	var maxMemory uint64
	var maxCPU float64
	var finalIOStats map[string]uint64

	done := make(chan error)
	go func() {
		done <- rCmd.Wait()
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			memUsage, cpuUsage, ioStats, err := collectProcessMetrics(int32(pid))
			if err != nil {
				log.Warn("Failed to collect process metrics: %v", err)
				continue
			}

			if memUsage["rss"] > maxMemory {
				maxMemory = memUsage["rss"]
			}

			if cpuUsage > maxCPU {
				maxCPU = cpuUsage
			}

			finalIOStats = ioStats

		case err := <-done:
			if err != nil {
				return nil, fmt.Errorf("R implementation failed: %w", err)
			}

			// Stop timing
			duration := time.Since(startTime)

			// Create result
			result := &BenchmarkResult{
				Implementation: "R",
				Runtime:        duration,
				MemoryUsage: map[string]uint64{
					"rss": maxMemory,
				},
				CPUUsage: maxCPU,
				IOStats:  finalIOStats,
			}

			// Ensure CPUUsage is not NaN
			if math.IsNaN(result.CPUUsage) {
				result.CPUUsage = 0
			}

			return result, nil
		}
	}
}

func runBenchmark(cmd *cobra.Command, args []string) error {
	// Get benchmark-specific flags
	iterations, _ := cmd.Flags().GetInt("iterations")
	profile, _ := cmd.Flags().GetBool("profile")
	profileDir, _ := cmd.Flags().GetString("profile-dir")
	reportFile, _ := cmd.Flags().GetString("report-file")
	verbose, _ := cmd.Flags().GetBool("verbose")

	if verbose {
		log.Info("Verbose output enabled")
	}

	log.Info("Starting paf2serotypes benchmark")
	log.Info(Config.String())
	log.Info("Benchmark settings:")
	log.Info("  Iterations: %d", iterations)
	log.Info("  Profile: %v", profile)
	log.Info("  Profile Directory: %s", profileDir)
	log.Info("  Report File: %s", reportFile)

	// Create results slices
	goResults := make([]*BenchmarkResult, iterations)
	rResults := make([]*BenchmarkResult, iterations)

	// Run benchmark iterations
	for i := 0; i < iterations; i++ {
		log.Info("Running benchmark iteration %d/%d", i+1, iterations)

		// Run Go implementation
		goResult, err := runGoImplementation(cmd, args)
		if err != nil {
			return fmt.Errorf("Go benchmark iteration %d failed: %w", i+1, err)
		}
		goResults[i] = goResult

		// Run R implementation
		rResult, err := runRImplementation(cmd, args)
		if err != nil {
			return fmt.Errorf("R benchmark iteration %d failed: %w", i+1, err)
		}
		rResults[i] = rResult

		log.Info("Benchmark iteration %d completed", i+1)
		log.Info("  Go runtime: %v", goResult.Runtime)
		log.Info("  R runtime: %v", rResult.Runtime)
		log.Info("  Speedup: %.2fx", float64(rResult.Runtime)/float64(goResult.Runtime))
	}

	// Calculate average results
	var totalGoRuntime time.Duration
	var totalRRuntime time.Duration
	var totalGoMemory uint64
	var totalRMemory uint64
	var totalGoCPU float64
	var totalRCPU float64

	for i := 0; i < iterations; i++ {
		totalGoRuntime += goResults[i].Runtime
		totalRRuntime += rResults[i].Runtime
		totalGoMemory += goResults[i].MemoryUsage["rss"]
		totalRMemory += rResults[i].MemoryUsage["rss"]
		totalGoCPU += goResults[i].CPUUsage
		totalRCPU += rResults[i].CPUUsage
	}

	avgGoRuntime := totalGoRuntime / time.Duration(iterations)
	avgRRuntime := totalRRuntime / time.Duration(iterations)
	avgGoMemory := totalGoMemory / uint64(iterations)
	avgRMemory := totalRMemory / uint64(iterations)
	avgGoCPU := totalGoCPU / float64(iterations)
	avgRCPU := totalRCPU / float64(iterations)

	// Calculate speedup with safety checks
	runtimeSpeedup := 0.0
	if avgGoRuntime > 0 {
		runtimeSpeedup = float64(avgRRuntime) / float64(avgGoRuntime)
	}

	memoryImprovement := 0.0
	if avgGoMemory > 0 {
		memoryImprovement = float64(avgRMemory) / float64(avgGoMemory)
	}

	cpuImprovement := 0.0
	if avgGoCPU > 0 {
		cpuImprovement = avgRCPU / avgGoCPU
	}

	// Report results
	log.Info("Benchmark results summary:")
	log.Info("  Iterations: %d", iterations)
	log.Info("  Average Go runtime: %v", avgGoRuntime)
	log.Info("  Average R runtime: %v", avgRRuntime)
	log.Info("  Runtime speedup: %.2fx", runtimeSpeedup)
	log.Info("  Average Go memory usage: %.2f MB", float64(avgGoMemory)/1024/1024)
	log.Info("  Average R memory usage: %.2f MB", float64(avgRMemory)/1024/1024)
	log.Info("  Memory usage improvement: %.2fx", memoryImprovement)
	log.Info("  Average Go CPU usage: %.2f%%", avgGoCPU)
	log.Info("  Average R CPU usage: %.2f%%", avgRCPU)
	log.Info("  CPU usage improvement: %.2fx", cpuImprovement)

	// Helper function to handle NaN values
	safeFloat := func(f float64) interface{} {
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil
		}
		return f
	}

	// Create detailed report
	report := map[string]interface{}{
		"config": map[string]interface{}{
			"iterations":      iterations,
			"paf_file":        Config.PafFile,
			"mapping_file":    Config.MappingFile,
			"sample_name":     Config.SampleName,
			"score_threshold": Config.ScoreThreshold,
		},
		"go_results": goResults,
		"r_results":  rResults,
		"summary": map[string]interface{}{
			"avg_go_runtime":     avgGoRuntime.String(),
			"avg_r_runtime":      avgRRuntime.String(),
			"runtime_speedup":    safeFloat(runtimeSpeedup),
			"avg_go_memory":      avgGoMemory,
			"avg_r_memory":       avgRMemory,
			"memory_improvement": safeFloat(memoryImprovement),
			"avg_go_cpu":         avgGoCPU,
			"avg_r_cpu":          avgRCPU,
			"cpu_improvement":    safeFloat(cpuImprovement),
		},
	}

	// Write report to file
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report to JSON: %w", err)
	}

	if err := os.WriteFile(reportFile, reportJSON, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	log.Info("Benchmark report written to %s", reportFile)

	return nil
}
