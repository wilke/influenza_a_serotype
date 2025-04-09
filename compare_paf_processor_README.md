# PAF Processor Output Comparison Tool

This tool compares the outputs of different PAF processor implementations for influenza A serotyping. It analyzes the read assignments from each implementation and generates comprehensive reports with visualizations.

## Features

- Parses output files from multiple PAF processor implementations
- Compares read assignments between implementations
- Identifies differences in serotype assignments
- Calculates agreement metrics (percentage agreement, Cohen's Kappa)
- Generates visualizations showing the differences
- Creates comprehensive markdown and HTML reports

## Components

### 1. Main Script (`compare_paf_processor_outputs.py`)

The main script that orchestrates the comparison process:
- Parses command-line arguments
- Collects read assignments from output files
- Creates a unified DataFrame for analysis
- Calls utility functions for metrics calculation and visualization
- Generates the final reports

### 2. Utility Module (`compare_paf_processor_utils.py`)

A module containing utility functions for:
- Calculating agreement metrics
- Generating visualizations
- Creating reports

## Usage

```bash
python compare_paf_processor_outputs.py <output_dir> [options]
```

### Arguments

- `output_dir`: Directory containing comparison results (with r_implementation, go_default, and go_r_algorithm subdirectories)

### Options

- `--report-dir`: Directory to save the output comparison report (default: output_dir/reports)
- `--verbose`, `-v`: Enable verbose output

## Output

The tool generates the following outputs in the report directory:

### Visualizations

- **Confusion matrices**: Shows how reads are assigned differently between implementations
- **Agreement metrics**: Bar charts showing percentage agreement and Cohen's Kappa
- **Serotype distribution**: Bar charts showing the distribution of reads across serotypes
- **Assignment changes**: Heatmaps showing how reads change assignments between implementations

### Reports

- **Markdown report**: A comprehensive report in markdown format
- **HTML report**: An HTML version of the report with styling

## Requirements

- Python 3.6+
- Required packages:
  - pandas
  - numpy
  - matplotlib
  - seaborn
  - scikit-learn
  - markdown (optional, for HTML report generation)

## Example

1. Run the comparison framework to generate outputs:
```bash
./compare_paf_processors.sh paf_files.txt results/ 4 0.8
```

2. Compare the outputs:
```bash
python compare_paf_processor_outputs.py results/
```

3. View the generated report:
```bash
open results/reports/output_comparison_report.html
```

## Report Structure

The generated report includes:

1. **Overview**: Introduction to the compared implementations
2. **Summary Statistics**: Read counts and serotype distribution
3. **Agreement Metrics**: Percentage agreement and Cohen's Kappa
4. **Confusion Matrices**: Detailed view of assignment differences
5. **Read Assignment Changes**: Analysis of how reads change assignments
6. **Conclusion**: Summary of findings and recommendations

## Extending the Tool

To add support for additional implementations:

1. Add the implementation name to the `implementations` list in `compare_paf_processor_outputs.py`
2. Ensure the output directory structure follows the same pattern as existing implementations
3. Run the tool to generate updated comparisons

## Troubleshooting

If you encounter issues:

1. Check that the output directory structure is correct
2. Verify that the serotype files exist and are in the expected format
3. Ensure all required Python packages are installed
4. Run with the `--verbose` flag for detailed logging