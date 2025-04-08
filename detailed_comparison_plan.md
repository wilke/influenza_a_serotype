# Detailed Comparison Plan for PAF Processor Implementations

This document outlines the comprehensive framework we've created for comparing the R and Go implementations of the PAF processor for influenza A serotyping.

## Overview

We've developed a suite of tools to perform a detailed comparison of three implementations:
1. **R Implementation** (`parse_pafs_influenza_A.R`)
2. **Go Default Implementation** (`pafprocessor`)
3. **Go R-Algorithm Implementation** (`pafprocessor --r-algorithm`)

## Components Created

### 1. Test Framework (`compare_paf_processors.sh`)

A comprehensive Bash script that:
- Takes a list of PAF files, output directory, worker count, and threshold as input
- Runs all three implementations on each PAF file
- Collects runtime metrics, memory usage, and exit codes
- Organizes outputs in a structured directory
- Generates a summary of results

### 2. Visualization Tool (`visualize_comparison.py`)

A Python script that:
- Generates visualizations of performance metrics
- Creates bar charts for runtime and memory usage
- Produces stacked bar charts for read assignments
- Generates both markdown and HTML reports

### 3. Memory Analysis Tool (`analyze_memory_usage.py`)

A specialized Python script for:
- Analyzing memory usage patterns over time
- Calculating key memory metrics (peak, average, growth rate)
- Identifying potential memory leaks or inefficiencies
- Generating memory efficiency metrics (reads per MB)

### 4. Output Comparison Tool (`compare_paf_processor_outputs.py` and `compare_paf_processor_utils.py`)

A set of Python scripts for:
- Comparing read assignments between implementations
- Calculating agreement metrics (percentage agreement, Cohen's Kappa)
- Generating confusion matrices and heatmaps
- Identifying specific differences in serotype assignments

### 5. Test Data Generator (`generate_test_data.py`)

A utility script for:
- Generating synthetic memory usage data for testing
- Simulating different memory patterns (stable, growing, fluctuating)
- Creating test data for the memory analysis tool

### 6. Documentation

Comprehensive documentation including:
- `PAF_PROCESSOR_COMPARISON.md`: Overview of the comparison framework
- `compare_paf_processor_README.md`: Documentation for the output comparison tool

## Workflow

The complete workflow for comparing the implementations is:

1. **Prepare Test Data**:
   - Create a list of PAF files to test
   - Ensure the mapping file is available

2. **Run Comparison**:
   ```bash
   ./compare_paf_processors.sh paf_files.txt results/ 4 0.8
   ```

3. **Generate Visualizations**:
   ```bash
   python visualize_comparison.py results/
   ```

4. **Analyze Memory Usage**:
   ```bash
   python analyze_memory_usage.py results/
   ```

5. **Compare Outputs**:
   ```bash
   python compare_paf_processor_outputs.py results/
   ```

6. **Review Reports**:
   - Performance report: `results/reports/comparison_report.html`
   - Memory analysis: `results/memory_analysis/memory_analysis_report.html`
   - Output comparison: `results/reports/output_comparison_report.html`

## Key Metrics Collected

### Performance Metrics
- Runtime (seconds)
- Memory usage (KB)
- Exit codes
- Reads processed per second

### Memory Metrics
- Peak memory usage
- Average memory usage
- Memory growth rate
- Memory stability
- Reads processed per MB

### Output Metrics
- Percentage agreement between implementations
- Cohen's Kappa coefficient
- Confusion matrices
- Serotype distribution differences
- Read assignment changes

## Directory Structure

The comparison framework creates the following directory structure:

```
results/
├── r_implementation/           # R implementation outputs
│   └── sample_name/            # Outputs for each sample
├── go_default/                 # Go default implementation outputs
│   └── sample_name/            # Outputs for each sample
├── go_r_algorithm/             # Go R-algorithm implementation outputs
│   └── sample_name/            # Outputs for each sample
├── comparison/                 # Comparison results
│   ├── runtime/                # Runtime comparison data
│   ├── memory/                 # Memory usage comparison data
│   └── outputs/                # Output differences
├── memory_analysis/            # Memory analysis results
│   ├── images/                 # Memory usage visualizations
│   ├── memory_analysis_report.md  # Markdown report
│   └── memory_analysis_report.html # HTML report
├── reports/                    # Generated reports
│   ├── images/                 # Visualization images
│   ├── comparison_report.md    # Performance comparison (markdown)
│   ├── comparison_report.html  # Performance comparison (HTML)
│   ├── output_comparison_report.md  # Output comparison (markdown)
│   └── output_comparison_report.html # Output comparison (HTML)
├── summary.tsv                 # Summary data in TSV format
└── final_summary.txt           # Text summary of comparison results
```

## Conclusion

This comprehensive comparison framework provides a detailed analysis of the three PAF processor implementations, focusing on:

1. **Performance**: Runtime and memory usage
2. **Memory Patterns**: Identifying potential memory issues
3. **Output Consistency**: Ensuring consistent serotype assignments

The framework is designed to be extensible, allowing for additional metrics and visualizations to be added as needed. It provides both high-level summaries and detailed analyses, making it suitable for both quick comparisons and in-depth investigations.