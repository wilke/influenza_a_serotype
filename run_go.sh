#!/bin/bash

# run_go.sh - Script to run the Go implementation
# Usage: ./run_go.sh <flu_db> <paf> <sample> <outdir> <score_thresh>

# Check if correct number of arguments is provided
if [ $# -ne 5 ]; then
    echo "Error: Incorrect number of arguments"
    echo "Usage: ./run_go.sh <flu_db> <paf> <sample> <outdir> <score_thresh>"
    echo "  <flu_db>       : Path to the influenza mapping file (e.g., DBs/v1.25/Influenza_A_segment_info1.tsv)"
    echo "  <paf>          : Path to the PAF file to process"
    echo "  <sample>       : Sample name for output files"
    echo "  <outdir>       : Output directory for results"
    echo "  <score_thresh> : Score threshold for assignments (e.g., 0.8)"
    exit 1
fi

# Assign arguments to variables
MAPPING_FILE="$1"
PAF_FILE="$2"
SAMPLE_NAME="$3"
OUTPUT_DIR="$4"
SCORE_THRESHOLD="$5"

# Check if files exist
if [ ! -f "$MAPPING_FILE" ]; then
    echo "Error: Mapping file not found: $MAPPING_FILE"
    exit 1
fi

if [ ! -f "$PAF_FILE" ]; then
    echo "Error: PAF file not found: $PAF_FILE"
    exit 1
fi

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

echo "===== Running Go implementation ====="
echo "Mapping File: $MAPPING_FILE"
echo "PAF File: $PAF_FILE"
echo "Sample Name: $SAMPLE_NAME"
echo "Output Directory: $OUTPUT_DIR"
echo "Score Threshold: $SCORE_THRESHOLD"
echo "========================================"

# Check if Go binary exists, if not build it
GO_BINARY="src/paf2serotypes/paf2serotypes"
if [ ! -f "$GO_BINARY" ]; then
    echo "Building Go binary..."
    cd src/paf2serotypes
    go build -o paf2serotypes
    if [ $? -ne 0 ]; then
        echo "Error: Failed to build Go binary"
        exit 1
    fi
    cd ../..
    echo "Go binary built successfully"
fi

# Run Go implementation
echo "Starting Go implementation..."
"$GO_BINARY" process \
    --mapping "$MAPPING_FILE" \
    --paf "$PAF_FILE" \
    --sample "$SAMPLE_NAME" \
    --output "$OUTPUT_DIR" \
    --min-score "$SCORE_THRESHOLD"

# Check if the Go implementation ran successfully
if [ $? -eq 0 ]; then
    echo "Go implementation ran successfully"
    echo "Output files:"
    ls -la "$OUTPUT_DIR"
    echo "Results written to $OUTPUT_DIR/${SAMPLE_NAME}_read_summary.tsv"
else
    echo "Error: Go implementation failed"
    exit 1
fi

echo "========================================"