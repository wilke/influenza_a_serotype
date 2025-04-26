#!/bin/bash

# run_original_r.sh - Script to run the original R implementation
# Usage: ./run_original_r.sh <flu_db> <paf> <sample> <outdir> <score_thresh>

# Check if correct number of arguments is provided
if [ $# -ne 5 ]; then
    echo "Error: Incorrect number of arguments"
    echo "Usage: ./run_original_r.sh <flu_db> <paf> <sample> <outdir> <score_thresh>"
    echo "  <flu_db>       : Path to the influenza mapping file (e.g., DBs/v1.25/Influenza_A_segment_info1.tsv)"
    echo "  <paf>          : Path to the PAF file to process"
    echo "  <sample>       : Sample name for output files"
    echo "  <outdir>       : Output directory for results"
    echo "  <score_thresh> : Score threshold for assignments (e.g., 0.8)"
    exit 1
fi

# Assign arguments to variables
FLU_DB="$1"
PAF_FILE="$2"
SAMPLE_NAME="$3"
OUTPUT_DIR="$4"
SCORE_THRESHOLD="$5"
R_SCRIPT="src/iav_serotype/parse_pafs_influenza_A.R"

# Check if files and directories exist
if [ ! -f "$FLU_DB" ]; then
    echo "Error: Flu database file not found: $FLU_DB"
    exit 1
fi

if [ ! -f "$PAF_FILE" ]; then
    echo "Error: PAF file not found: $PAF_FILE"
    exit 1
fi

if [ ! -f "$R_SCRIPT" ]; then
    echo "Error: R script not found: $R_SCRIPT"
    exit 1
fi

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

echo "===== Running R implementation ====="
echo "Flu DB: $FLU_DB"
echo "PAF File: $PAF_FILE"
echo "Sample Name: $SAMPLE_NAME"
echo "Output Directory: $OUTPUT_DIR"
echo "Score Threshold: $SCORE_THRESHOLD"
echo "R Script: $R_SCRIPT"
echo "========================================"

# Run R script
echo "Starting R script execution..."
Rscript "$R_SCRIPT" "$FLU_DB" "$PAF_FILE" "$SAMPLE_NAME" "$OUTPUT_DIR" "$SCORE_THRESHOLD"

# Check if the R script ran successfully
if [ $? -eq 0 ]; then
    echo "R script ran successfully"
    echo "Output files:"
    ls -la "$OUTPUT_DIR"
    echo "Results written to $OUTPUT_DIR/${SAMPLE_NAME}_read_summary.tsv"
else
    echo "Error: R script failed"
    exit 1
fi

echo "========================================"