#!/bin/bash

# compare.sh - Script to compare the outputs of the R and Go implementations
# Usage: ./compare.sh <outdir> <sample>

# Check if correct number of arguments is provided
if [ $# -ne 2 ]; then
    echo "Error: Incorrect number of arguments"
    echo "Usage: ./compare.sh <outdir> <sample>"
    echo "  <outdir> : Output directory containing results from both implementations"
    echo "  <sample> : Sample name used for the output files"
    exit 1
fi

# Assign arguments to variables
OUTPUT_DIR="$1"
SAMPLE_NAME="$2"
R_OUTPUT_DIR="${OUTPUT_DIR}/r_output"
REPORT_FILE="${OUTPUT_DIR}/comparison_report.json"

# Check if output directories exist
if [ ! -d "$OUTPUT_DIR" ]; then
    echo "Error: Output directory not found: $OUTPUT_DIR"
    exit 1
fi

if [ ! -d "$R_OUTPUT_DIR" ]; then
    echo "Error: R output directory not found: $R_OUTPUT_DIR"
    echo "Creating R output directory..."
    mkdir -p "$R_OUTPUT_DIR"
fi

# Check if output files exist
GO_SUMMARY_FILE="${OUTPUT_DIR}/${SAMPLE_NAME}_read_summary.tsv"
R_SUMMARY_FILE="${R_OUTPUT_DIR}/${SAMPLE_NAME}_read_summary.tsv"

# if [ ! -f "$GO_SUMMARY_FILE" ]; then
#     echo "Error: Go summary file not found: $GO_SUMMARY_FILE"
#     echo "Please run the Go implementation first using run_go.sh"
#     exit 1
# fi

# if [ ! -f "$R_SUMMARY_FILE" ]; then
#     echo "Error: R summary file not found: $R_SUMMARY_FILE"
#     echo "Please run the R implementation first using run_original_r.sh"
#     exit 1
# fi

echo "===== Comparing R and Go implementations ====="
echo "Output Directory: $OUTPUT_DIR"
echo "R Output Directory: $R_OUTPUT_DIR"
echo "Sample Name: $SAMPLE_NAME"
echo "Go Summary File: $GO_SUMMARY_FILE"
echo "R Summary File: $R_SUMMARY_FILE"
echo "Report File: $REPORT_FILE"
echo "========================================"

# Check if Go binary exists
GO_BINARY="src/paf2serotypes/paf2serotypes"
if [ ! -f "$GO_BINARY" ]; then
    echo "Error: Go binary not found: $GO_BINARY"
    echo "Please build the Go implementation first using run_go.sh"
    exit 1
fi

# Run comparison
echo "Starting comparison..."
"$GO_BINARY" compare \
    --r-script "src/iav_serotype/parse_pafs_influenza_A.R" \
    --mapping "DBs/v1.25/Influenza_A_segment_info1.tsv" \
    --paf "test_data/medium_test_20000.paf" \
    --r-output "$R_OUTPUT_DIR" \
    --report-file "$REPORT_FILE" \
    --sample "$SAMPLE_NAME" \
    --output "$OUTPUT_DIR"

# Check if the comparison ran successfully
if [ $? -eq 0 ]; then
    echo "Comparison completed successfully"
    echo "Report written to $REPORT_FILE"
    
    # Display summary of comparison results
    if [ -f "$REPORT_FILE" ]; then
        echo "========================================"
        echo "Comparison Summary:"
        echo "========================================"
        
        # Extract key metrics from the JSON report using grep and sed
        # This is a simple approach that works without requiring jq
        echo "Performance:"
        grep -o '"go_runtime":"[^"]*"' "$REPORT_FILE" | sed 's/"go_runtime":"\(.*\)"/  Go Runtime: \1/'
        grep -o '"r_runtime":"[^"]*"' "$REPORT_FILE" | sed 's/"r_runtime":"\(.*\)"/  R Runtime: \1/'
        grep -o '"speedup":[0-9.]*' "$REPORT_FILE" | sed 's/"speedup":\(.*\)/  Speedup: \1x/'
        
        echo "Output Comparison:"
        grep -o '"total_lines":[0-9]*' "$REPORT_FILE" | sed 's/"total_lines":\(.*\)/  Total Lines: \1/'
        grep -o '"matching_lines":[0-9]*' "$REPORT_FILE" | sed 's/"matching_lines":\(.*\)/  Matching Lines: \1/'
        grep -o '"different_lines":[0-9]*' "$REPORT_FILE" | sed 's/"different_lines":\(.*\)/  Different Lines: \1/'
        grep -o '"match_percentage":[0-9.]*' "$REPORT_FILE" | sed 's/"match_percentage":\(.*\)/  Match Percentage: \1%/'
        
        # Check if outputs match exactly
        DIFFERENT_LINES=$(grep -o '"different_lines":[0-9]*' "$REPORT_FILE" | sed 's/"different_lines":\(.*\)/\1/')
        if [ "$DIFFERENT_LINES" = "0" ]; then
            echo "  Result: Outputs match exactly!"
        else
            echo "  Result: Outputs differ!"
        fi
    else
        echo "Warning: Report file not found"
    fi
else
    echo "Error: Comparison failed"
    exit 1
fi

echo "========================================"
echo "For detailed results, open the HTML report:"
echo "${OUTPUT_DIR}/benchmark_report.html"
echo "========================================"