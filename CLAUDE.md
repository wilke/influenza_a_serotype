# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the Influenza A Serotype Assignment Tool - a bioinformatics pipeline that assigns sequencing reads to influenza A serotypes through competitive alignment. The project has implementations in R (original), Python (wrapper/CLI), and Go (performance-optimized).

## Essential Commands

### Environment Setup
```bash
# Create and activate conda environment
mamba env create -f environment/iav_serotype.yaml
conda activate iav_serotype

# Install Python package
pip install .
```

### Build Commands
```bash
# Build Go implementations
cd src/paf2serotypes && go build && cd ../..
cd src/myPaf2Serotypes && go build && cd ../..
```

### Test Commands
```bash
# Python tests
python src/comparison/test_implementations.py

# Go tests
cd lib/go/paf2serotypes && go test ./...
cd src/myPaf2Serotypes && go test

# Run implementation comparison
./src/paf2serotypes/paf2serotypes compare --paf test_data/small_test.paf --mapping DBs/v1.25/Influenza_A_segment_info1.tsv --sample test_sample --output test_output --min-score 0.8 --r-script src/iav_serotype/parse_pafs_influenza_A.R
```

### Go Development
```bash
# Format Go code
go fmt ./...
```

## Architecture Overview

### Multi-Implementation Structure
- **R Implementation**: `src/iav_serotype/parse_pafs_influenza_A.R` - Original algorithm
- **Python CLI**: `src/iav_serotype/iav_serotype.py` - Main entry point and wrapper
- **Go Implementations**: Two performance-optimized versions
  - `src/paf2serotypes/` - First Go implementation
  - `src/myPaf2Serotypes/` - Alternative implementation

### Go Library Architecture (`lib/go/paf2serotypes/`)
- `config/`: Configuration management for processing parameters
- `io/`: PAF file reading and result writing
- `model/`: Core data structures (PAFRecord, SerotypeSummary, etc.)
- `processor/`: Business logic for serotype assignment
- `pipeline/`: Orchestrates the full processing pipeline
- `visualization/`: Generates summary plots and tables

### Data Processing Pipeline
1. FASTQ reads → minimap2 alignment → PAF format
2. PAF records → serotype assignment based on competitive scoring
3. Results → serotype-specific files, summary tables, and visualizations

### Key Algorithms
- Competitive alignment scoring: reads assigned to serotype with highest alignment score
- Minimum score threshold filtering (default 0.8)
- Chunking mechanism for processing large PAF files efficiently
- Segment-aware assignment (HA/NA segments determine serotype)

## Important Context

- Database version v1.25 contains 981,537 complete influenza A segments
- Supports both short paired-end reads and long reads (e.g., Nanopore)
- Active development focuses on Go implementation performance optimization
- Comparison tools exist to verify consistency between implementations
- The project uses CWL for workflow integration