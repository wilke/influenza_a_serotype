# Analysis of PAF Processor Implementations: R vs Go

## Algorithm Overview

### R Implementation (parse_pafs_influenza_A.R)

The R implementation processes PAF (Pairwise Alignment Format) files to identify influenza A serotypes through the following steps:

1. **Data Loading**: Reads PAF file and mapping database using `fread()` from the data.table package
2. **Data Merging**: Joins PAF entries with serotype information from the mapping database
3. **Alignment Score Calculation**:
   - Groups data by read name, target name, serotype, segment, and strand
   - Calculates key metrics:
     - ANI (Average Nucleotide Identity) = total matches / total alignment length
     - AF (Alignment Fraction) = total alignment length / total read length
     - Alignment Score = ANI * AF
4. **Read Assignment**:
   - Groups by read name, serotype, and segment
   - Takes top 2 scoring alignments
   - Assigns reads to a serotype if:
     - The difference between top scores is small (< 0.003)
     - OR all top hits are from the same serotype
   - Otherwise marks as "ambiguous"
5. **Output Generation**:
   - Writes summary table with read assignments
   - Creates serotype-specific files containing read names
   - Generates a bar plot visualization of assignments

### Go Implementation (pafprocessor)

The Go implementation offers two algorithmic approaches:

#### Default Go Algorithm
1. **Data Loading**: Reads PAF and mapping files in chunks for memory efficiency
2. **Alignment Score Calculation**:
   - Calculates alignment score as: NumMatches / AlignLength (ANI only)
   - Groups by read name, target name, serotype, segment, and optionally strand
3. **Read Assignment**:
   - Filters entries by minimum alignment score
   - Assigns reads based on whether all top hits have the same serotype
   - If serotypes differ, marks as "ambiguous"

#### R-Compatible Algorithm (--r-algorithm option)
1. **Alignment Score Calculation**:
   - Calculates ANI = NumMatches / AlignLength
   - Calculates AF = AlignLength / ReadLength
   - Alignment Score = ANI * AF (matching R implementation)
2. **Read Assignment**:
   - Takes top 2 scores
   - Assigns reads to a serotype if:
     - The difference between top scores is small (< 0.003)
     - OR all top hits are from the same serotype
   - Otherwise marks as "ambiguous"

## Key Differences Between Implementations

1. **Alignment Score Calculation**:
   - R: Always uses ANI * AF
   - Go (default): Uses only ANI (NumMatches / AlignLength)
   - Go (--r-algorithm): Matches R implementation using ANI * AF

2. **Processing Approach**:
   - R: Processes entire dataset in memory using data.table
   - Go: Uses chunked processing with worker pools for parallel execution

3. **Grouping Options**:
   - R: Always groups by strand
   - Go: Makes strand grouping optional via the --paired flag

4. **Performance Optimization**:
   - R: Relies on data.table's optimized operations
   - Go: Implements parallel processing, memory-efficient chunking, and performance profiling

5. **Output Formats**:
   - R: Generates TSV files and a PDF visualization
   - Go: Generates TSV files and performance metrics, but no visualization

## How Go's --r-algorithm Option Mimics R Implementation

The Go implementation with --r-algorithm flag closely follows the R algorithm by:

1. **Using identical score calculation**: 
   ```go
   // R algorithm: align_score = ANI * AF
   ani := float64(entry.NumMatches) / float64(entry.AlignLength)
   af := float64(entry.AlignLength) / float64(entry.QLength)
   alignScore = ani * af
   ```

2. **Implementing the same read assignment logic**:
   - Takes top 2 scores
   - Uses the same 0.003 threshold for score difference
   - Applies the same conditions for serotype assignment vs. "ambiguous"

3. **Producing compatible output formats** that match the R implementation's file structure

## Key Parameters and Their Effects

1. **minAlignmentScore / score_thresh**:
   - R: Filters reads below the threshold score
   - Go: Same functionality with --min-score flag
   - Effect: Higher values increase specificity but reduce sensitivity

2. **paired / strand grouping**:
   - R: Always groups by strand
   - Go: Optional with --paired flag
   - Effect: When enabled, treats reads from different strands as separate entities

3. **useRAlgorithm / --r-algorithm**:
   - Go-specific parameter to switch between algorithms
   - Effect: Changes both score calculation and read assignment logic

4. **numWorkers / --workers**:
   - Go-specific parameter for parallel processing
   - Effect: Higher values can improve performance on multi-core systems

5. **chunkSize / --chunk-size**:
   - Go-specific parameter for memory management
   - Effect: Smaller chunks use less memory but may increase processing overhead

## Output Formats and Data Structures

### R Implementation
1. **Read Summary TSV**: Contains read name, serotype, segment, scores, and assignment
2. **Serotype-specific files**: Simple text files with read names for each serotype
3. **PDF Visualization**: Bar plot of read assignments

### Go Implementation
1. **Read Summary TSV**: Similar to R but with additional fields for all strands and segments
2. **Serotype-specific files**: Identical format to R implementation
3. **Performance metrics**: TSV and text summary of processing times

### Data Structures
- Both implementations use similar conceptual data structures:
  - PAF entries with alignment information
  - Mapping entries with serotype information
  - Summary entries with assignment results

- Go implementation adds explicit structures for:
  - Grouped entries (intermediate processing)
  - Performance metrics
  - Logging configuration

## Conclusion

The Go implementation provides a more scalable and performance-optimized approach to PAF processing for influenza A serotyping, while maintaining compatibility with the R implementation through the --r-algorithm option. The key algorithmic difference lies in the alignment score calculation and read assignment logic, which can be toggled between the Go-specific approach and the R-compatible approach.

For testing and comparison purposes, the parameter mapping between implementations is straightforward, with the Go version offering additional parameters for performance optimization. Output formats are compatible between implementations, making it possible to directly compare results.
