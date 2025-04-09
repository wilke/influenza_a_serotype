# PAF Processor Implementation Comparison Framework

This framework provides tools for comparing the performance and output of different implementations of the PAF (Pairwise Alignment Format) processor for influenza A serotyping.

## Implementations Compared

1. **R Implementation** (`parse_pafs_influenza_A.R`): The original implementation in R
2. **Go Default Implementation** (`pafprocessor`): The Go implementation with its default algorithm
3. **Go R-Algorithm Implementation** (`pafprocessor --r-algorithm`): The Go implementation using the R-compatible algorithm

## Framework Components

### 1. Main Comparison Script (`compare_paf_processors.sh`)

This script runs all three implementations on a set of PAF files and collects performance metrics and outputs for comparison.

**Usage:**
```bash
./compare_paf_processors.sh [options] <paf_files_list> <output_dir> <workers> <threshold>
```

**Arguments:**
- `paf_files_list`: Text file containing paths to PAF files for testing, one per line
- `output_dir`: Directory to store all output files and comparison results
- `workers`: Number of worker goroutines for Go implementation
- `threshold`: Minimum alignment score threshold (0.0-1.0)

**Options:**
- `-h, --help`: Show help message and exit
- `-m, --mapping`: Path to the mapping TSV file (default: environment/iav_serotype.yaml)
- `-v, --verbose`: Enable verbose output

### 2. Visualization Script (`visualize_comparison.py`)

This Python script generates visualizations and reports from the comparison results.

**Usage:**
```bash
pythom visualize_comparison.py <output_dir>
```

**Arguments:**
- `output_dir`: Directory containing the comparison results

**Outputs:**
- Runtime comparison charts
- Memory usage comparison charts
- Read assignment comparison charts
- File-specific performance comparisons
- Markdown and HTML reports with all visualizations and output differences

### 3. Example Run Script (`run_example_comparison.sh`)

A simple script to run the comparison with example test files.

**Usage:**
```bash
./run_example_comparison.sh
```

## Output Directory Structure

The comparison framework creates the following directory structure in the output directory:

```
output_dir/
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
├── reports/                    # Generated reports
│   ├── images/                 # Visualization images
│   ├── comparison_report.md    # Markdown report
│   └── comparison_report.html  # HTML report
├── summary.tsv                 # Summary data in TSV format
└── final_summary.txt           # Text summary of comparison results
```

## Metrics Collected

The framework collects the following metrics for each implementation:

1. **Runtime**: Execution time in seconds
2. **Memory Usage**: Maximum memory usage in KB
3. **Exit Codes**: Exit codes for each run
4. **Read Assignments**: Number of reads assigned to each serotype (H1N1, H3N2, ambiguous)

## Output Comparison

The framework compares the outputs of the three implementations by:

1. Comparing the read assignments to each serotype
2. Generating diff files to show differences in read assignments
3. Calculating consistency metrics between implementations

## Requirements

- Bash shell
- R with required packages (data.table, stringr, dplyr, ggplot2)
- Go compiler
- Python 3 with required packages (pandas, matplotlib, seaborn)
- Optional: Python markdown package for HTML report generation

## Example Usage

1. Create a list of PAF files to test:
```
test_data/small_test.paf
test_data/medium_test_20000.paf
test_data/debug_test.paf
```

2. Run the comparison:
```bash
./compare_paf_processors.sh paf_files.txt results/ 4 0.8
```

3. Generate visualizations and reports:
```bash
pythom visualize_comparison.py results/
```

4. View the results:
```bash
open results/reports/comparison_report.html
```

## Extending the Framework

To add more PAF files for testing:
1. Add the PAF files to the test data directory
2. Create a new list file with the paths to the PAF files
3. Run the comparison with the new list file

To add more metrics for comparison:
1. Modify the `compare_paf_processors.sh` script to collect additional metrics
2. Update the `visualize_comparison.py` script to visualize the new metrics
3. Update the report templates to include the new metrics

## Troubleshooting

If you encounter issues with the comparison:

1. Check that all required dependencies are installed
2. Verify that the PAF files exist and are in the correct format
3. Check the log file in the output directory for error messages
4. Ensure that the mapping file exists and is in the correct format
5. Check that the R and Go implementations are properly installed and accessible