# PAF Processor Output Comparison Tool

This tool compares the outputs of different PAF processor implementations for influenza A serotyping. It analyzes the read assignments from each implementation and generates comprehensive reports with visualizations to evaluate performance, accuracy, and consistency across implementations.

## Features

- Parses output files from multiple PAF processor implementations
- Compares read assignments between implementations
- Identifies differences in serotype assignments
- Calculates agreement metrics (percentage agreement, Cohen's Kappa)
- Generates visualizations showing the differences
- Creates comprehensive markdown and HTML reports
- Analyzes memory usage patterns and efficiency
- Evaluates performance scaling with dataset size
- Compares algorithmic differences and their impact

## Components

### 1. Main Script (`compare_paf_processor_outputs.py`)

The main script that orchestrates the comparison process:
- Parses command-line arguments
- Collects read assignments from output files of all three implementations:
  - Python/R Implementation (original reference implementation using ANI * AF scoring)
  - Go Default Implementation (with modified algorithm using ANI only)
  - Go R-Algorithm Implementation (replicates R algorithm logic with ANI * AF)
- Creates a unified DataFrame for analysis
- Calls utility functions for metrics calculation and visualization
- Generates the final reports with detailed analysis of differences

### 2. Utility Module (`compare_paf_processor_utils.py`)

A module containing utility functions for:
- Calculating agreement metrics (percentage agreement, Cohen's Kappa)
- Generating visualizations (confusion matrices, heatmaps, bar charts)
- Creating reports in both markdown and HTML formats
- Analyzing algorithmic differences between implementations
- Evaluating memory efficiency and performance characteristics
- Identifying patterns in assignment differences

### 3. Visualization Tool (`visualize_comparison.py`)

A dedicated visualization tool that:
- Creates comprehensive visual representations of comparison results
- Generates interactive charts for performance metrics
- Produces detailed visualizations of memory usage patterns
- Creates comparative visualizations of algorithm differences
- Embeds visualizations in HTML reports for easy interpretation

## Usage

```bash
python compare_paf_processor_outputs.py <output_dir> [options]
```

### Arguments

- `output_dir`: Directory containing comparison results (with r_implementation, go_default, and go_r_algorithm subdirectories)

### Options

- `--report-dir`: Directory to save the output comparison report (default: output_dir/reports)
- `--verbose`, `-v`: Enable verbose output
- `--detailed-analysis`: Generate additional detailed analysis of algorithmic differences
- `--memory-analysis`: Include memory usage analysis in the report
- `--include-raw-data`: Include raw data tables in the report
- `--interactive-charts`: Generate interactive HTML charts (requires plotly)

## Output

The tool generates the following outputs in the report directory:

### Visualizations

- **Confusion matrices**: Shows how reads are assigned differently between implementations
- **Agreement metrics**: Bar charts showing percentage agreement and Cohen's Kappa
- **Serotype distribution**: Bar charts showing the distribution of reads across serotypes
- **Assignment changes**: Heatmaps showing how reads change assignments between implementations
- **Algorithm comparison**: Visualizations highlighting the impact of different scoring methods (ANI vs ANI*AF)
- **Memory usage**: Charts showing memory consumption patterns across implementations
- **Performance scaling**: Visualizations of how performance scales with dataset size and worker count
- **Time series analysis**: Charts showing processing time and memory usage over time

### Reports

- **Markdown report**: A comprehensive report in markdown format
- **HTML report**: An HTML version of the report with styling and interactive elements
- **Performance summary**: Detailed analysis of runtime and memory efficiency
- **Algorithm impact analysis**: Assessment of how algorithm differences affect serotype assignments
- **Edge case analysis**: Detailed examination of reads with different assignments
- **Memory efficiency report**: Analysis of memory usage patterns and optimization opportunities

## Requirements

- Python 3.6+
- Required packages:
  - pandas
  - numpy
  - matplotlib
  - seaborn
  - scikit-learn
  - markdown (optional, for HTML report generation)
  - plotly (optional, for interactive visualizations)
  - psutil (optional, for detailed memory analysis)

## Example

1. Run the comparison framework to generate outputs:
```bash
./compare_paf_processors.sh paf_files.txt results/ 4 0.8
```

2. Compare the outputs:
```bash
python compare_paf_processor_outputs.py results/ --detailed-analysis --memory-analysis
```

3. View the generated report:
```bash
open results/reports/output_comparison_report.html
```

4. Analyze memory usage patterns:
```bash
python analyze_memory_usage.py results/
```

5. Generate interactive visualizations:
```bash
python visualize_comparison.py results/ --interactive
```

## Report Structure

The generated report includes:

1. **Overview**: Introduction to the compared implementations
   - Python/R Implementation (original reference implementation using ANI * AF scoring)
   - Go Default Implementation (with modified algorithm using ANI only)
   - Go R-Algorithm Implementation (replicates R algorithm logic with ANI * AF)

2. **Summary Statistics**: Read counts and serotype distribution
   - Total reads processed by each implementation
   - Distribution of serotype assignments
   - Processing time and memory usage
   - Throughput metrics (reads per second)

3. **Agreement Metrics**: Percentage agreement and Cohen's Kappa
   - Pairwise agreement between implementations
   - Statistical significance of agreement
   - Confidence intervals for agreement metrics

4. **Confusion Matrices**: Detailed view of assignment differences
   - Visualization of how reads are assigned differently
   - Identification of systematic differences
   - Quantification of specific assignment changes

5. **Algorithm Impact Analysis**:
   - Effects of using ANI vs ANI*AF for scoring
   - Impact of the 0.003 score threshold
   - Memory optimization through chunked processing
   - Performance gains from parallelization
   - Scaling efficiency with multiple workers

6. **Read Assignment Changes**: Analysis of how reads change assignments
   - Patterns in assignment changes
   - Identification of edge cases
   - Characterization of ambiguous assignments

7. **Memory Efficiency Analysis**:
   - Memory usage patterns over time
   - Peak memory consumption comparison
   - Memory efficiency relative to dataset size
   - Impact of chunk size on memory usage

8. **Performance Comparison**:
   - Runtime comparison across implementations
   - CPU utilization patterns
   - Scaling with dataset size
   - Impact of parallelization on performance

9. **Conclusion**: Summary of findings and recommendations
   - Performance comparison
   - Accuracy assessment
   - Memory efficiency evaluation
   - Recommendations for specific use cases

## Extending the Tool

To add support for additional implementations:

1. Add the implementation name to the `implementations` list in `compare_paf_processor_outputs.py`
2. Ensure the output directory structure follows the same pattern as existing implementations
3. Update the comparison logic to include the new implementation
4. Add visualization support for the new implementation
5. Run the tool to generate updated comparisons

To add new metrics or visualizations:

1. Implement the metric calculation in `compare_paf_processor_utils.py`
2. Add visualization code to `visualize_comparison.py`
3. Update the report generation to include the new metrics and visualizations

## Troubleshooting

If you encounter issues:

1. Check that the output directory structure is correct
2. Verify that the serotype files exist and are in the expected format
3. Ensure all required Python packages are installed
4. Run with the `--verbose` flag for detailed logging
5. Check memory usage if processing large files
6. Verify that the implementations are using compatible parameters
7. Examine the log files for each implementation for specific errors
8. For memory issues, try reducing the chunk size or increasing available memory
9. For performance issues, experiment with different worker counts