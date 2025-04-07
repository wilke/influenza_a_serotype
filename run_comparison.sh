#!/bin/bash

# Run comparison script for Influenza A serotyping implementations
# This script runs the comparison on the small test dataset

# Ensure the Go binary is built
echo "Building Go implementation..."
cd src/pafprocessor
go build
cd ../..

# Create results directory
mkdir -p comparison_results

# Run the comparison script
echo "Running comparison on small dataset..."
python compare_implementations.py --datasets small --score-threshold 0.8 --workers 4 --chunk-size 1000

echo "Comparison complete! Check the comparison_results directory for the report."