# PAF Processor

A Go executable for processing PAF (Pairwise mApping Format) files and mapping them to serotypes.

## Overview

This tool reads a PAF file and a TSV mapping file, processes the records in parallel, and outputs the results. It includes detailed logging and performance benchmarking for each step of the process.

## Features

- Parallel processing of PAF records
- Detailed logging
- Performance benchmarking
- Memory and CPU profiling

## Usage

```bash
./pafprocessor --mapping <mapping_file> --paf <paf_file> [options]
```

### Required Arguments

- `--mapping`: Path to the mapping TSV file
- `--paf`: Path to the PAF file

### Optional Arguments

- `--min-score`: Minimum alignment score (default: 0.8)
- `--paired`: Group by strand if true (default: false)
- `--output`: Output directory (default: ".")
- `--sample`: Sample name (default: "sample")
- `--log-level`: Log level (debug, info, warn, error) (default: "info")
- `--workers`: Number of worker goroutines (default: number of CPUs)
- `--chunk-size`: Number of PAF entries per chunk (default: 1000)
- `--r-algorithm`: Use R-compatible algorithm for read assignment (default: false)
- `--cpu-profile`: Write CPU profile to file
- `--mem-profile`: Write memory profile to file
- `--log-file`: Path to log file
## Example

```bash
./pafprocessor --mapping DBs/v1.25/Influenza_A_segment_info1.tsv --paf test_data/small_test.paf --output results --sample test_sample
```

### Using the R-compatible Algorithm

To use the algorithm that matches the R implementation (parse_pafs_influenza_A.R):

```bash
./pafprocessor --mapping DBs/v1.25/Influenza_A_segment_info1.tsv --paf test_data/small_test.paf --output results --sample test_sample --r-algorithm
```

The R-compatible algorithm differs from the default Go implementation in two key ways:

1. **Alignment Score Calculation**:
   - R algorithm: `align_score = ANI * AF` (where ANI is the alignment identity and AF is the alignment fraction)
   - Default Go algorithm: `alignScore = NumMatches / AlignLength`

2. **Read Assignment Logic**:
   - R algorithm: Takes the top 2 scores and assigns the read to the top serotype if either:
     - All top serotypes are the same, OR
     - The difference between the top score and second score is less than 0.003
     - Otherwise, it assigns "ambiguous"
   - Default Go algorithm: Assigns the read to the serotype if all filtered serotypes are the same, otherwise "ambiguous"
```

## Output

The tool generates the following outputs:

1. **Summary File**: A TSV file containing the summary of the processed records.
2. **Read Assignment Files**: Files containing the read assignments for each serotype.
3. **Performance Report**: A report of the performance metrics.

## Performance Benchmarking

The tool includes detailed performance metrics for:

1. **File Reading**: Time taken to read the mapping file and PAF file.
2. **Record Processing**: Time taken to process each chunk of records.
3. **Result Generation**: Time taken to generate the final results.
4. **Memory Usage**: Memory usage at each stage of the process.
5. **Overall Execution**: Total time taken to execute the program.

## Logging

The tool includes detailed logging for:

1. **Startup**: Command-line arguments and configuration.
2. **File Reading**: Number of records read from each file.
3. **Record Processing**: Number of records processed, filtered, and grouped.
4. **Result Generation**: Number of results generated.
5. **Performance**: Execution time and memory usage for each step.
6. **Errors**: Any errors encountered during execution.

## Benchmark Mode

The benchmark mode allows you to measure and compare the performance between the Go and R implementations of the PAF processor. This mode is particularly useful when optimizing code and tracking performance improvements.

### Usage

```bash
./pafprocessor benchmark \
  --paf <paf_file> \
  --mapping <mapping_file> \
  --sample <sample_name> \
  --output <output_dir> \
  --min-score <score_threshold> \
  --r-script <r_script_path> \
  --r-output <r_output_dir> \
  --iterations <num_iterations> \
  --report-file <report_file>
```

### Required Arguments

- `--r-script`: Path to the R script implementation

### Optional Arguments

- `--iterations`: Number of benchmark iterations (default: 3)
- `--profile`: Enable CPU and memory profiling (default: false)
- `--profile-dir`: Directory for profiling output (default: ".")
- `--r-output`: Output directory for R results (default: "<output_dir>/r_output")
- `--report-file`: Output file for benchmark report (default: "benchmark_report.json")
- `--verbose`: Enable verbose output (default: false)

### Metrics Collected

The benchmark mode collects the following metrics for both implementations:

- **Runtime**: Total execution time for processing the input data
- **Memory Usage**: Peak memory consumption (RSS, VMS, Swap)
- **CPU Utilization**: Average CPU usage percentage
- **I/O Operations**: Number of read/write operations and bytes

### Output

The benchmark mode generates a detailed JSON report containing:

1. **Configuration**: Input parameters and settings used for the benchmark
2. **Raw Results**: Detailed metrics for each iteration of both implementations
3. **Summary**: Average metrics and improvement ratios between implementations

Recent updates include a fix for JSON marshaling of NaN values in performance metrics, ensuring the benchmark reports are always valid JSON.

### Example

```bash
./pafprocessor benchmark \
  --paf test_data/small_test.paf \
  --mapping DBs/v1.25/Influenza_A_segment_info1.tsv \
  --sample benchmark_test \
  --output test_output \
  --r-script src/iav_serotype/parse_pafs_influenza_A.R \
  --iterations 5 \
  --report-file test_output/benchmark_report.json
```

## Compare Mode

The compare mode verifies the correctness of the Go implementation by comparing its output with the original R implementation. This mode is essential when making changes to ensure the Go implementation produces correct results.

### Usage

```bash
./pafprocessor compare \
  --paf <paf_file> \
  --mapping <mapping_file> \
  --sample <sample_name> \
  --output <output_dir> \
  --min-score <score_threshold> \
  --r-script <r_script_path> \
  --r-output <r_output_dir> \
  --report-file <report_file>
```

### Required Arguments

- `--r-script`: Path to the R script implementation

### Optional Arguments

- `--r-output`: Output directory for R results (default: "<output_dir>/R")
- `--report-file`: Output file for comparison report (default: "comparison_report.json")
- `--verbose`: Enable verbose output (default: false)

### Comparison Checks

The compare mode performs the following checks:

1. **Line-by-Line Comparison**: Verifies that each line in the output files matches
2. **Serotype Assignment Comparison**: Checks that serotype assignments are identical
3. **Count Verification**: Ensures that the count of each serotype matches between implementations

### Output

The compare mode generates a detailed JSON report showing:

1. **Performance Metrics**: Runtime, memory usage, and CPU utilization for both implementations
2. **Output Comparison**: Line counts, matching status, and any differences found
3. **Serotype Analysis**: Counts of serotypes and any discrepancies between implementations

### Example

```bash
./pafprocessor compare \
  --paf test_data/small_test.paf \
  --mapping DBs/v1.25/Influenza_A_segment_info1.tsv \
  --sample test_sample \
  --output test_output \
  --r-script src/iav_serotype/parse_pafs_influenza_A.R \
  --report-file test_output/comparison_report.json
```

## Benchmark Script

A benchmark script (`benchmark_script.sh`) is provided to run both the benchmark and compare commands with a single command. The script also generates an HTML report visualizing the benchmark results.

### Usage

```bash
./benchmark_script.sh
```

### HTML Report

The HTML report includes:

- Performance comparison charts
- Memory usage visualization
- Output comparison results
- Detailed metrics tables