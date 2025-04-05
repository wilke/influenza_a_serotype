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
- `--cpu-profile`: Write CPU profile to file
- `--mem-profile`: Write memory profile to file
- `--log-file`: Path to log file

## Example

```bash
./pafprocessor --mapping DBs/v1.25/Influenza_A_segment_info1.tsv --paf test_data/small_test.paf --output results --sample test_sample
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