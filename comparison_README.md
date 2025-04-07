# Influenza A Serotyping Implementation Comparison

This directory contains tools for comparing the R and Go implementations of the Influenza A serotyping algorithm.

## Files

- `detailed_comparison_plan.md`: A detailed plan for comparing the R and Go implementations
- `compare_implementations.py`: Python script to run both implementations and compare their results
- `run_comparison.sh`: Shell script to run the comparison on the small test dataset

## Requirements

- Python 3.6+
- Required Python packages:
  - pandas
  - numpy
  - matplotlib
  - seaborn
- R with the following packages:
  - data.table
  - stringr
  - dplyr
  - ggplot2
- Go 1.16+

## Usage

### Running the Comparison

To run the comparison on the small test dataset:

```bash
./run_comparison.sh
```

This will:
1. Build the Go implementation
2. Run the comparison script on the small test dataset
3. Generate a report in the `comparison_results` directory

### Custom Comparison

To run a custom comparison, you can use the Python script directly:

```bash
python3 compare_implementations.py --datasets small medium large --score-threshold 0.8 --workers 4 --chunk-size 1000
```

Parameters:
- `--datasets`: Space-separated list of datasets to use (small, medium, large)
- `--score-threshold`: Score threshold for read assignment (default: 0.8)
- `--workers`: Number of worker goroutines for Go implementation (default: 4)
- `--chunk-size`: Chunk size for Go implementation (default: 1000)

## Output

The comparison script generates the following outputs in the `comparison_results` directory:

1. A timestamp-based directory for each run
2. For each dataset:
   - R and Go implementation outputs
   - Performance comparison metrics and charts
   - Result comparison metrics and charts
3. A comprehensive report (`comparison_report.md`) summarizing the findings

## Extending the Comparison

To add more datasets or metrics to the comparison:

1. Add new datasets to the `TEST_DATASETS` dictionary in `compare_implementations.py`
2. Modify the comparison functions to include additional metrics
3. Update the report generation function to include the new metrics

## Algorithm Differences

The key differences between the R and Go implementations are:

1. **Parallelism**: The Go implementation uses goroutines for parallel processing, while the R script processes data sequentially.
2. **Memory Management**: The Go implementation processes data in chunks, which is more memory-efficient for large files.
3. **Alignment Score Calculation**: The R script calculates alignment score as `ANI * AF`, while the Go program uses a different formula.
4. **Read Assignment Logic**: The read assignment logic differs between implementations, with the R script using a threshold of 0.003 for score differences.