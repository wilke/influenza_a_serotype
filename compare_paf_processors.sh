#!/bin/bash

# compare_paf_processors.sh
# A comprehensive script to compare R and Go implementations of PAF processors
# for influenza A serotyping.

# Exit on any error, undefined variable, or pipe failure
set -euo pipefail

# conda activate iav

# Function to clean up temporary files on exit
function cleanup {
    # Remove temporary files if they exist
    if [[ -n "${PAF_FILES_LIST:-}" && -f "$PAF_FILES_LIST" ]]; then
        rm -f "$PAF_FILES_LIST"
    fi
    
    # Kill any background processes that might still be running
    jobs -p | xargs -r kill 2>/dev/null || true
}

# Register cleanup function to run on exit
trap cleanup EXIT INT TERM

# Display usage information
function show_usage {
    echo "Usage: $0 [options] <output_dir> <workers> <threshold>"
    echo ""
    echo "Required Arguments:"
    echo "  output_dir        Directory to store all output files and comparison results"
    echo "  workers           Number of worker goroutines for Go implementation (positive integer)"
    echo "  threshold         Minimum alignment score threshold (0.0-1.0)"
    echo ""
    echo "Options:"
    echo "  -h, --help        Show this help message and exit"
    echo "  -m, --mapping     Path to the mapping TSV file (default: DBs/v1.25/Influenza_A_segment_info1.tsv)"
    echo "  -d, --data        Path to PAF file to process (can be specified multiple times)"
    echo "  -v, --verbose     Enable verbose output"
    echo ""
    exit 1
}

# Function to check if a value is a valid number
function is_valid_number {
    local value="$1"
    [[ "$value" =~ ^[0-9]+(\.[0-9]+)?$ ]] && return 0 || return 1
}

# Function to check if a command exists
function command_exists {
    command -v "$1" >/dev/null 2>&1
}

# Check required commands
for cmd in bc Rscript /usr/bin/time; do
    if ! command_exists "$cmd"; then
        echo "Error: Required command '$cmd' not found. Please install it before running this script."
        exit 1
    fi
done

# Check if Go implementation is available
if [[ ! -f "./src/paf2serotypes/paf2serotypes" ]]; then
    echo "Error: Go implementation not found at ./src/paf2serotypes/paf2serotypes"
    echo "Please compile the Go implementation before running this script."
    exit 1
fi

# Check if R script is available
if [[ ! -f "src/iav_serotype/parse_pafs_influenza_A.R" ]]; then
    echo "Error: R script not found at src/iav_serotype/parse_pafs_influenza_A.R"
    exit 1
fi

# Parse command line options
VERBOSE=false
MAPPING_FILE="DBs/v1.25/Influenza_A_segment_info1.tsv"
PAF_FILES=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)
            show_usage
            ;;
        -m|--mapping)
            if [[ -z "${2:-}" ]]; then
                echo "Error: Missing argument for $1 option"
                show_usage
            fi
            MAPPING_FILE="$2"
            shift 2
            ;;
        -d|--data)
            if [[ -z "${2:-}" ]]; then
                echo "Error: Missing argument for $1 option"
                show_usage
            fi
            PAF_FILES+=("$2")
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -*)
            echo "Error: Unknown option: $1"
            show_usage
            ;;
        *)
            break
            ;;
    esac
done

# Check required arguments
if [[ $# -ne 3 ]]; then
    echo "Error: Missing required arguments"
    show_usage
fi

# Check if at least one PAF file was provided
if [[ ${#PAF_FILES[@]} -eq 0 ]]; then
    echo "Error: No PAF files specified. Use -d/--data option to specify PAF files"
    show_usage
fi

OUTPUT_DIR="$1"
WORKERS="$2"
THRESHOLD="$3"

# Validate arguments
if [[ ! -f "$MAPPING_FILE" ]]; then
    echo "Error: Mapping file '$MAPPING_FILE' not found"
    exit 1
fi

# Validate workers is a positive integer
if ! [[ "$WORKERS" =~ ^[0-9]+$ ]] || [[ "$WORKERS" -lt 1 ]]; then
    echo "Error: Workers must be a positive integer"
    exit 1
fi

# Validate threshold is between 0.0 and 1.0
if ! is_valid_number "$THRESHOLD" || (( $(echo "$THRESHOLD < 0.0 || $THRESHOLD > 1.0" | bc -l) )); then
    echo "Error: Threshold must be a number between 0.0 and 1.0"
    exit 1
fi

# Create temporary PAF files list with proper error handling
PAF_FILES_LIST=$(mktemp) || { echo "Error: Failed to create temporary file"; exit 1; }
for file in "${PAF_FILES[@]}"; do
    if [[ ! -f "$file" ]]; then
        echo "Error: PAF file '$file' not found"
        exit 1
    fi
    echo "$file" >> "$PAF_FILES_LIST" || { echo "Error: Failed to write to temporary file"; exit 1; }
done

# Check if we can create the output directory
if [[ -d "$OUTPUT_DIR" ]]; then
    if [[ ! -w "$OUTPUT_DIR" ]]; then
        echo "Error: Output directory '$OUTPUT_DIR' exists but is not writable"
        exit 1
    fi
else
    mkdir -p "$OUTPUT_DIR" || { echo "Error: Failed to create output directory '$OUTPUT_DIR'"; exit 1; }
fi

# Create output directory structure with error handling
for dir in "$OUTPUT_DIR/r_implementation" "$OUTPUT_DIR/go_default" "$OUTPUT_DIR/go_r_algorithm" \
           "$OUTPUT_DIR/comparison" "$OUTPUT_DIR/comparison/runtime" "$OUTPUT_DIR/comparison/memory" \
           "$OUTPUT_DIR/comparison/outputs" "$OUTPUT_DIR/reports" "$OUTPUT_DIR/reports/images"; do
    mkdir -p "$dir" || { echo "Error: Failed to create directory '$dir'"; exit 1; }
done

# Log file
LOG_FILE="$OUTPUT_DIR/comparison.log"
SUMMARY_FILE="$OUTPUT_DIR/summary.tsv"

# Initialize summary file
echo -e "file\timplementation\truntime_seconds\tmax_memory_kb\texit_code\tH1N1_reads\tH3N2_reads\tambiguous_reads" > "$SUMMARY_FILE" || { 
    echo "Error: Failed to write to summary file '$SUMMARY_FILE'"; 
    exit 1; 
}

# Function to log messages
function log {
    local message="$1"
    local timestamp=$(date "+%Y-%m-%d %H:%M:%S")
    # Write to log file and stderr instead of stdout
    echo "[$timestamp] $message" | tee -a "$LOG_FILE" >&2 || true
    
    if [[ "$VERBOSE" == "true" ]]; then
        echo "[$timestamp] $message" >&2
    fi
}

# Function to log debug information
function debug {
    if [[ "$VERBOSE" == "true" ]]; then
        local message="$1"
        local timestamp=$(date "+%Y-%m-%d %H:%M:%S")
        # Write to log file and stderr instead of stdout
        echo "[DEBUG][$timestamp] $message" | tee -a "$LOG_FILE" >&2 || true
    fi
}

log "Starting PAF processor comparison"
log "PAF files list: $PAF_FILES_LIST"
log "Output directory: $OUTPUT_DIR"
log "Workers: $WORKERS"
log "Threshold: $THRESHOLD"
log "Mapping file: $MAPPING_FILE"

# Check if required files and directories exist
log "Checking required files and directories:"
if [[ ! -f "$MAPPING_FILE" ]]; then
    log "ERROR: Mapping file not found at $MAPPING_FILE"
fi

if [[ ! -f "src/iav_serotype/parse_pafs_influenza_A.R" ]]; then
    log "ERROR: R script not found at src/iav_serotype/parse_pafs_influenza_A.R"
fi

if [[ ! -f "./src/paf2serotypes/paf2serotypes" ]]; then
    log "ERROR: Go binary not found at ./src/paf2serotypes/paf2serotypes"
fi

# Check if the DBs directory exists
if [[ ! -d "DBs" ]]; then
    log "ERROR: DBs directory not found"
fi

# Function to measure memory usage of a process
function measure_memory {
    local pid=$1
    local output_file=$2
    
    # Check that process exists before monitoring
    if ! kill -0 "$pid" 2>/dev/null; then
        echo "Warning: Process $pid doesn't exist, cannot measure memory" >> "$LOG_FILE"
        return 1
    fi
    
    while kill -0 "$pid" 2>/dev/null; do
        ps -o rss= -p "$pid" >> "$output_file" 2>/dev/null || break
        sleep 0.1
    done
}

# Function to run and monitor a process
function run_process {
    # The first argument is NOT the output directory - it's part of the command
    local command=("$@")
    local name="${command[0]##*/}"
    
    # Extract output directory from command context
    local output_dir=""
    # Look for --output parameter in the command arguments
    for ((i=0; i<${#command[@]}; i++)); do
        if [[ "${command[$i]}" == "--output" && $((i+1)) -lt ${#command[@]} ]]; then
            output_dir="${command[$i+1]}"
            break
        fi
    done
    
    # If no output directory found in the command, use current directory
    if [[ -z "$output_dir" ]]; then
        output_dir="."
    fi
    
    local memory_file="$output_dir/memory_usage.txt"
    local time_file="$output_dir/time_output.txt"
    local stdout_file="$output_dir/stdout.log"
    local stderr_file="$output_dir/stderr.log"
    
    # Ensure output directory exists
    if [[ ! -d "$output_dir" ]]; then
        log "Creating directory: $output_dir"
        mkdir -p "$output_dir" || { log "Error: Failed to create directory '$output_dir'"; return 1; }
    fi
    
    # Print command for debugging
    debug "Running command: ${command[*]}"
    
    # Measure runtime and memory usage
    local start_time=$(date +%s.%N)
    
    "${command[@]}" > "$stdout_file" 2> "$stderr_file" &
    local process_pid=$!
    
    debug "Started process with PID: $process_pid"
    
    # Monitor memory usage
    measure_memory "$process_pid" "$memory_file" &
    local monitor_pid=$!
    
    # Wait for process to complete
    wait "$process_pid" || true
    local exit_code=$?
    
    # Kill memory monitor
    kill "$monitor_pid" 2>/dev/null || true
    wait "$monitor_pid" 2>/dev/null || true
    
    local end_time=$(date +%s.%N)
    local runtime=$(echo "$end_time - $start_time" | bc)
    
    # Get max memory - handle empty file case
    local max_memory=0
    if [[ -f "$memory_file" && -s "$memory_file" ]]; then
        max_memory=$(sort -n "$memory_file" 2>/dev/null | tail -n 1) || max_memory=0
    fi
    
    # Display exit code information with different formatting based on success/failure
    if [[ $exit_code -eq 0 ]]; then
        log "$name completed successfully with exit code $exit_code in $runtime seconds"
        
        # Check if any output was produced
        log "Checking output files in $output_dir:"
        ls -la "$output_dir" 2>/dev/null | while read -r line; do
            log "  | $line"
        done
        
        # Print stdout for debugging
        if [[ -s "$stdout_file" ]]; then
            log "Stdout from $name:"
            head -n 20 "$stdout_file" | while read -r line; do
                log "  | $line"
            done
            if [[ $(wc -l < "$stdout_file") -gt 20 ]]; then
                log "  | ... (truncated, see $stdout_file for full output)"
            fi
        else
            log "No stdout output from $name"
        fi
    else
        log "ERROR: $name failed with exit code $exit_code after $runtime seconds"
        # Print error output if process failed
        if [[ -s "$stderr_file" ]]; then
            log "Error output from $name:"
            cat "$stderr_file" | while read -r line; do
                log "  | $line"
            done
        else
            log "No stderr output from $name despite failure"
        fi
    fi
    
    # Return values as a string, caller will parse
    # Use stderr for logs and stdout only for the return value
    echo "$exit_code $runtime $max_memory"
}

# Process each PAF file
while read -r paf_file; do
    if [[ ! -f "$paf_file" ]]; then
        log "Warning: PAF file '$paf_file' not found, skipping"
        continue
    fi
    
    # Extract filename without path and extension
    filename=$(basename "$paf_file")
    sample_name="${filename%.*}"
    
    # Create input-specific directories 
    input_dir_name="${filename}"
    
    log "Processing file: $paf_file (sample: $sample_name, input: $input_dir_name)"
    
    # 1. Run R implementation
    log "Running R implementation..."
    r_output_dir="$OUTPUT_DIR/r_implementation/$input_dir_name"
    mkdir -p "$r_output_dir" || { log "Error: Failed to create directory '$r_output_dir'"; continue; }
    
    # Log the full command for debugging
    log "R command: /usr/bin/time -l Rscript src/iav_serotype/parse_pafs_influenza_A.R \"$MAPPING_FILE\" \"$paf_file\" \"$sample_name\" \"$r_output_dir\" \"$THRESHOLD\""
    
    # Check if the R script exists
    if [[ ! -f "src/iav_serotype/parse_pafs_influenza_A.R" ]]; then
        log "ERROR: R script not found at src/iav_serotype/parse_pafs_influenza_A.R"
    fi
    
    r_result=$(run_process /usr/bin/time -l Rscript src/iav_serotype/parse_pafs_influenza_A.R \
        "$MAPPING_FILE" \
        "$paf_file" \
        "$sample_name" \
        "$r_output_dir" \
        "$THRESHOLD")
    
    read -r r_exit_code r_runtime r_max_memory <<< "$r_result"
    
    if (( r_exit_code != 0 )); then
        log "CRITICAL: R implementation failed with exit code $r_exit_code"
        continue
    fi
    
    r_h1n1_count=0
    r_h3n2_count=0
    r_ambiguous_count=0
    
    if [[ -f "$r_output_dir/${sample_name}_H1N1.txt" ]]; then
        r_h1n1_count=$(wc -l < "$r_output_dir/${sample_name}_H1N1.txt" 2>/dev/null || echo 0)
    fi
    
    if [[ -f "$r_output_dir/${sample_name}_H3N2.txt" ]]; then
        r_h3n2_count=$(wc -l < "$r_output_dir/${sample_name}_H3N2.txt" 2>/dev/null || echo 0)
    fi
    
    if [[ -f "$r_output_dir/${sample_name}_ambiguous.txt" ]]; then
        r_ambiguous_count=$(wc -l < "$r_output_dir/${sample_name}_ambiguous.txt" 2>/dev/null || echo 0)
    fi
    
    echo -e "$filename\tr_implementation\t$r_runtime\t$r_max_memory\t$r_exit_code\t$r_h1n1_count\t$r_h3n2_count\t$r_ambiguous_count" >> "$SUMMARY_FILE" || log "Warning: Failed to update summary file"
    
    # 2. Run Go implementation with default algorithm
    log "Running Go implementation with default algorithm..."
    go_default_output_dir="$OUTPUT_DIR/go_default/$input_dir_name"
    mkdir -p "$go_default_output_dir" || { log "Error: Failed to create directory '$go_default_output_dir'"; continue; }
    
    log "Running Go implementation with default algorithm..."
    go_default_output_dir="$OUTPUT_DIR/go_default/$sample_name"
    mkdir -p "$go_default_output_dir" || { log "Error: Failed to create directory '$go_default_output_dir'"; continue; }
    
    # Log the full command for debugging
    log "Go default command: /usr/bin/time -l ./src/paf2serotypes/paf2serotypes --mapping \"$MAPPING_FILE\" --paf \"$paf_file\" --min-score \"$THRESHOLD\" --workers \"$WORKERS\" --output \"$go_default_output_dir\" --sample \"$sample_name\" --log-level \"info\""
    
    # Check if the Go binary exists
    if [[ ! -f "./src/paf2serotypes/paf2serotypes" ]]; then
        log "ERROR: Go binary not found at ./src/paf2serotypes/paf2serotypes"
    fi
    
    go_default_result=$(run_process /usr/bin/time -l ./src/paf2serotypes/paf2serotypes \
        --mapping "$MAPPING_FILE" \
        --paf "$paf_file" \
        --min-score "$THRESHOLD" \
        --workers "$WORKERS" \
        --output "$go_default_output_dir" \
        --sample "$sample_name" \
        --log-level "info")
    
    read -r go_default_exit_code go_default_runtime go_default_max_memory <<< "$go_default_result"
    
    # Check Go default implementation exit code explicitly - fix integer comparison issues
    if (( go_default_exit_code != 0 )); then
        log "CRITICAL: Go default implementation failed with exit code $go_default_exit_code"
        # Optional: decide if you want to continue processing or abort
        # continue
    else
        debug "Go default implementation exited with code 0 (success)"
    fi
    
    # Count reads in each category
    go_default_h1n1_count=0
    go_default_h3n2_count=0
    go_default_ambiguous_count=0
    
    if [[ -f "$go_default_output_dir/${sample_name}_H1N1.txt" ]]; then
        go_default_h1n1_count=$(wc -l < "$go_default_output_dir/${sample_name}_H1N1.txt" 2>/dev/null || echo 0)
    fi
    
    if [[ -f "$go_default_output_dir/${sample_name}_H3N2.txt" ]]; then
        go_default_h3n2_count=$(wc -l < "$go_default_output_dir/${sample_name}_H3N2.txt" 2>/dev/null || echo 0)
    fi
    
    if [[ -f "$go_default_output_dir/${sample_name}_ambiguous.txt" ]]; then
        go_default_ambiguous_count=$(wc -l < "$go_default_output_dir/${sample_name}_ambiguous.txt" 2>/dev/null || echo 0)
    fi
    
    # Add to summary
    echo -e "$filename\tgo_default\t$go_default_runtime\t$go_default_max_memory\t$go_default_exit_code\t$go_default_h1n1_count\t$go_default_h3n2_count\t$go_default_ambiguous_count" >> "$SUMMARY_FILE" || log "Warning: Failed to update summary file"
    
    # 3. Run Go implementation with R algorithm
    log "Running Go implementation with R algorithm..."
    go_r_algorithm_output_dir="$OUTPUT_DIR/go_r_algorithm/$sample_name"
    mkdir -p "$go_r_algorithm_output_dir" || { log "Error: Failed to create directory '$go_r_algorithm_output_dir'"; continue; }
    
    # Log the full command for debugging
    log "Go R-algorithm command: /usr/bin/time -l ./src/paf2serotypes/paf2serotypes --mapping \"$MAPPING_FILE\" --paf \"$paf_file\" --min-score \"$THRESHOLD\" --workers \"$WORKERS\" --output \"$go_r_algorithm_output_dir\" --sample \"$sample_name\" --log-level \"info\" --r-algorithm"
    
    # Check if the Go binary exists (redundant check, but keeping for completeness)
    if [[ ! -f "./src/paf2serotypes/paf2serotypes" ]]; then
        log "ERROR: Go binary not found at ./src/paf2serotypes/paf2serotypes"
    fi
    
    go_r_algorithm_result=$(run_process /usr/bin/time -l ./src/paf2serotypes/paf2serotypes \
        --mapping "$MAPPING_FILE" \
        --paf "$paf_file" \
        --min-score "$THRESHOLD" \
        --workers "$WORKERS" \
        --output "$go_r_algorithm_output_dir" \
        --sample "$sample_name" \
        --log-level "info" \
        --r-algorithm)
    
    read -r go_r_algorithm_exit_code go_r_algorithm_runtime go_r_algorithm_max_memory <<< "$go_r_algorithm_result"
    
    # Check Go R-algorithm implementation exit code explicitly - fix integer comparison issues
    if (( go_r_algorithm_exit_code != 0 )); then
        log "CRITICAL: Go R-algorithm implementation failed with exit code $go_r_algorithm_exit_code"
        # Optional: decide if you want to continue processing or abort
        # continue
    else
        debug "Go R-algorithm implementation exited with code 0 (success)"
    fi
    
    # Count reads in each category
    go_r_algorithm_h1n1_count=0
    go_r_algorithm_h3n2_count=0
    go_r_algorithm_ambiguous_count=0
    
    if [[ -f "$go_r_algorithm_output_dir/${sample_name}_H1N1.txt" ]]; then
        go_r_algorithm_h1n1_count=$(wc -l < "$go_r_algorithm_output_dir/${sample_name}_H1N1.txt" 2>/dev/null || echo 0)
    fi
    
    if [[ -f "$go_r_algorithm_output_dir/${sample_name}_H3N2.txt" ]]; then
        go_r_algorithm_h3n2_count=$(wc -l < "$go_r_algorithm_output_dir/${sample_name}_H3N2.txt" 2>/dev/null || echo 0)
    fi
    
    if [[ -f "$go_r_algorithm_output_dir/${sample_name}_ambiguous.txt" ]]; then
        go_r_algorithm_ambiguous_count=$(wc -l < "$go_r_algorithm_output_dir/${sample_name}_ambiguous.txt" 2>/dev/null || echo 0)
    fi
    
    # Add to summary
    echo -e "$filename\tgo_r_algorithm\t$go_r_algorithm_runtime\t$go_r_algorithm_max_memory\t$go_r_algorithm_exit_code\t$go_r_algorithm_h1n1_count\t$go_r_algorithm_h3n2_count\t$go_r_algorithm_ambiguous_count" >> "$SUMMARY_FILE" || log "Warning: Failed to update summary file"
    
    # Compare outputs
    log "Comparing outputs for $sample_name..."
    
    # Create comparison directory if it doesn't exist
    mkdir -p "$OUTPUT_DIR/comparison/outputs" || { log "Error: Failed to create comparison directory"; continue; }
    
    # Function to safely compare files
    function safe_diff {
        local file1="$1"
        local file2="$2"
        local output="$3"
        
        if [[ -f "$file1" && -f "$file2" ]]; then
            sort "$file1" > "${file1}.sorted"
            sort "$file2" > "${file2}.sorted"
            diff -u "${file1}.sorted" "${file2}.sorted" > "$output" 2>/dev/null || true
            rm -f "${file1}.sorted" "${file2}.sorted"

        elif [[ -f "$file1" ]]; then
            echo "Only $file1 exists, no comparison made" > "$output"
        elif [[ -f "$file2" ]]; then
            echo "Only $file2 exists, no comparison made" > "$output"
        else
            echo "One or both files missing: $file1 $file2" > "$output"
        fi
    }
    
    # Compare R vs Go default
    safe_diff "$r_output_dir/${sample_name}_H1N1.txt" "$go_default_output_dir/${sample_name}_H1N1.txt" "$OUTPUT_DIR/comparison/outputs/${sample_name}_H1N1_r_vs_go_default.diff"
    safe_diff "$r_output_dir/${sample_name}_H3N2.txt" "$go_default_output_dir/${sample_name}_H3N2.txt" "$OUTPUT_DIR/comparison/outputs/${sample_name}_H3N2_r_vs_go_default.diff"
    safe_diff "$r_output_dir/${sample_name}_ambiguous.txt" "$go_default_output_dir/${sample_name}_ambiguous.txt" "$OUTPUT_DIR/comparison/outputs/${sample_name}_ambiguous_r_vs_go_default.diff"
    
    # Compare R vs Go R-algorithm
    safe_diff "$r_output_dir/${sample_name}_H1N1.txt" "$go_r_algorithm_output_dir/${sample_name}_H1N1.txt" "$OUTPUT_DIR/comparison/outputs/${sample_name}_H1N1_r_vs_go_r_algorithm.diff"
    safe_diff "$r_output_dir/${sample_name}_H3N2.txt" "$go_r_algorithm_output_dir/${sample_name}_H3N2.txt" "$OUTPUT_DIR/comparison/outputs/${sample_name}_H3N2_r_vs_go_r_algorithm.diff"
    safe_diff "$r_output_dir/${sample_name}_ambiguous.txt" "$go_r_algorithm_output_dir/${sample_name}_ambiguous.txt" "$OUTPUT_DIR/comparison/outputs/${sample_name}_ambiguous_r_vs_go_r_algorithm.diff"
    
    # Compare Go default vs Go R-algorithm
    safe_diff "$go_default_output_dir/${sample_name}_H1N1.txt" "$go_r_algorithm_output_dir/${sample_name}_H1N1.txt" "$OUTPUT_DIR/comparison/outputs/${sample_name}_H1N1_go_default_vs_go_r_algorithm.diff"
    safe_diff "$go_default_output_dir/${sample_name}_H3N2.txt" "$go_r_algorithm_output_dir/${sample_name}_H3N2.txt" "$OUTPUT_DIR/comparison/outputs/${sample_name}_H3N2_go_default_vs_go_r_algorithm.diff"
    safe_diff "$go_default_output_dir/${sample_name}_ambiguous.txt" "$go_r_algorithm_output_dir/${sample_name}_ambiguous.txt" "$OUTPUT_DIR/comparison/outputs/${sample_name}_ambiguous_go_default_vs_go_r_algorithm.diff"
    
    log "Completed processing for $paf_file"
    echo "----------------------------------------"
    
done < "$PAF_FILES_LIST"

log "All PAF files processed. Generating final summary..."

# Create a final summary file
if ! cat > "$OUTPUT_DIR/final_summary.txt" << EOF
PAF Processor Comparison Summary
===============================

Date: $(date)
PAF Files: $(wc -l < "$PAF_FILES_LIST" 2>/dev/null || echo "unknown")
Workers: $WORKERS
Threshold: $THRESHOLD
Mapping File: $MAPPING_FILE

Runtime Comparison (seconds):
$(awk -F'\t' 'NR>1 {sum[$2]+=$3; count[$2]++} END {for (imp in sum) printf "%-15s %10.3f (avg)\n", imp, sum[imp]/count[imp]}' "$SUMMARY_FILE" 2>/dev/null || echo "Error calculating runtime comparison")

Memory Usage Comparison (KB):
$(awk -F'\t' 'NR>1 {if ($4 > max[$2]) max[$2]=$4} END {for (imp in max) printf "%-15s %10d (max)\n", imp, max[imp]}' "$SUMMARY_FILE" 2>/dev/null || echo "Error calculating memory usage comparison")

Read Assignment Comparison:
$(awk -F'\t' 'NR>1 {h1n1[$2]+=$6; h3n2[$2]+=$7; amb[$2]+=$8; total[$2]+=$6+$7+$8} 
  END {
    for (imp in total) {
      if (total[imp] > 0) {
        printf "%-15s H1N1: %7d (%5.1f%%), H3N2: %7d (%5.1f%%), Ambiguous: %7d (%5.1f%%)\n", 
          imp, 
          h1n1[imp], 
          h1n1[imp]/total[imp]*100, 
          h3n2[imp], 
          h3n2[imp]/total[imp]*100, 
          amb[imp], 
          amb[imp]/total[imp]*100
      } else {
        printf "%-15s No reads processed\n", imp
      }
    }
  }' "$SUMMARY_FILE" 2>/dev/null || echo "Error calculating read assignment comparison")

Exit Code Summary:
$(awk -F'\t' 'NR>1 {codes[$2","$5]++} 
  END {
    for (code_pair in codes) {
      split(code_pair, parts, ",")
      printf "%-15s Exit code %s: %d occurrences\n", parts[1], parts[2], codes[code_pair]
    }
  }' "$SUMMARY_FILE" 2>/dev/null || echo "Error calculating exit code summary")

EOF
then
    log "Warning: Failed to create final summary file"
fi

log "Final summary created at $OUTPUT_DIR/final_summary.txt"
log "Comparison completed successfully"

# Check if visualization script exists before suggesting it
if [[ -f "$(dirname "$0")/visualize_comparison.py" ]]; then 
    echo "Run the visualization script to generate reports:"
    echo "python $(dirname "$0")/visualize_comparison.py $OUTPUT_DIR"
else 
    echo "Visualization script not found. Comparison complete."
fi

exit 0