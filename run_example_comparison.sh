#!/bin/bash

# run_example_comparison.sh
# A simple script to run the PAF processor comparison with example test files

set -e

# Create a temporary file with the list of PAF files to test
PAF_FILES_LIST="test_paf_files.txt"
OUTPUT_DIR="test_comparison_results"
WORKERS=4
THRESHOLD=0.8

conda activate iav

# Create the list of PAF files to test
echo "Creating list of PAF files to test..."
cat > "$PAF_FILES_LIST" << EOF
test_data/small_test.paf
test_data/debug_test.paf
test_data/medium_test_20000.paf
EOF

echo "PAF files to test:"
cat "$PAF_FILES_LIST"

# Make the comparison script executable
chmod +x compare_paf_processors.sh
chmod +x visualize_comparison.py

# Run the comparison
echo "Running PAF processor comparison..."
./compare_paf_processors.sh "$PAF_FILES_LIST" "$OUTPUT_DIR" "$WORKERS" "$THRESHOLD"

# Run the visualization
echo "Generating visualization and reports..."
pythom visualize_comparison.py "$OUTPUT_DIR"

echo "Comparison completed. Results are in $OUTPUT_DIR"
echo "Markdown report: $OUTPUT_DIR/reports/comparison_report.md"
echo "HTML report: $OUTPUT_DIR/reports/comparison_report.html"