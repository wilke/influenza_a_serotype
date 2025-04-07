#!/usr/bin/env python
"""
Comparison Script for Influenza A Serotyping Implementations

This script compares the R and Go implementations for Influenza A serotyping
by running both on the same datasets and analyzing the results.
"""

import os
import sys
import time
import subprocess
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
from datetime import datetime
import argparse
import resource
import tempfile
import shutil
import json
from pathlib import Path

# Define paths
PROJECT_ROOT = os.path.dirname(os.path.abspath(__file__))
R_SCRIPT_PATH = os.path.join(PROJECT_ROOT, "src/iav_serotype/parse_pafs_influenza_A.R")
GO_BINARY_PATH = os.path.join(PROJECT_ROOT, "src/pafprocessor/pafprocessor")
DB_PATH = os.path.join(PROJECT_ROOT, "DBs/v1.25/Influenza_A_segment_info1.tsv")
TEST_DATA_DIR = os.path.join(PROJECT_ROOT, "test_data")
RESULTS_DIR = os.path.join(PROJECT_ROOT, "comparison_results")

# Define test datasets
TEST_DATASETS = {
    "small": os.path.join(TEST_DATA_DIR, "small_test.200.paf"),
    "medium": os.path.join(TEST_DATA_DIR, "medium_test_20000.paf"),
    "large": os.path.join(TEST_DATA_DIR, "xHYB004_fastp_107678_iav_influenza_A.cigar.paf")
}

# Ensure the results directory exists
os.makedirs(RESULTS_DIR, exist_ok=True)

def measure_execution(cmd, output_dir):
    """
    Execute a command and measure its execution time and resource usage.
    
    Args:
        cmd (list): Command to execute as a list of strings
        output_dir (str): Directory to store output files
        
    Returns:
        dict: Dictionary containing execution metrics
    """
    start_time = time.time()
    start_resources = resource.getrusage(resource.RUSAGE_CHILDREN)
    
    # Execute the command
    process = subprocess.Popen(
        cmd, 
        stdout=subprocess.PIPE, 
        stderr=subprocess.PIPE
    )
    stdout, stderr = process.communicate()
    
    end_time = time.time()
    end_resources = resource.getrusage(resource.RUSAGE_CHILDREN)
    
    # Calculate resource usage
    user_time = end_resources.ru_utime - start_resources.ru_utime
    system_time = end_resources.ru_stime - start_resources.ru_stime
    max_rss = end_resources.ru_maxrss  # Maximum resident set size in KB
    
    # Save stdout and stderr
    with open(os.path.join(output_dir, "stdout.log"), "wb") as f:
        f.write(stdout)
    with open(os.path.join(output_dir, "stderr.log"), "wb") as f:
        f.write(stderr)
    
    return {
        "execution_time": end_time - start_time,
        "user_time": user_time,
        "system_time": system_time,
        "max_memory_kb": max_rss,
        "exit_code": process.returncode
    }

def run_r_implementation(paf_file, output_dir, sample_name, score_threshold=0.8):
    """
    Run the R implementation of the Influenza A serotyping algorithm.
    
    Args:
        paf_file (str): Path to the PAF file
        output_dir (str): Directory to store output files
        sample_name (str): Sample name
        score_threshold (float): Score threshold
        
    Returns:
        dict: Dictionary containing execution metrics
    """
    os.makedirs(output_dir, exist_ok=True)
    
    cmd = [
        "Rscript",
        R_SCRIPT_PATH,
        DB_PATH,
        paf_file,
        sample_name,
        output_dir,
        str(score_threshold)
    ]
    
    print(f"Running R implementation with command: {' '.join(cmd)}")
    metrics = measure_execution(cmd, output_dir)
    
    return metrics

def run_go_implementation(paf_file, output_dir, sample_name, score_threshold=0.8, num_workers=4, chunk_size=1000):
    """
    Run the Go implementation of the Influenza A serotyping algorithm.
    
    Args:
        paf_file (str): Path to the PAF file
        output_dir (str): Directory to store output files
        sample_name (str): Sample name
        score_threshold (float): Score threshold
        num_workers (int): Number of worker goroutines
        chunk_size (int): Number of PAF entries per chunk
        
    Returns:
        dict: Dictionary containing execution metrics
    """
    os.makedirs(output_dir, exist_ok=True)
    
    cmd = [
        GO_BINARY_PATH,
        "--mapping", DB_PATH,
        "--paf", paf_file,
        "--min-score", str(score_threshold),
        "--output", output_dir,
        "--sample", sample_name,
        "--workers", str(num_workers),
        "--chunk-size", str(chunk_size)
    ]
    
    print(f"Running Go implementation with command: {' '.join(cmd)}")
    metrics = measure_execution(cmd, output_dir)
    
    return metrics

def compare_results(r_output_dir, go_output_dir, sample_name, comparison_dir):
    """
    Compare the results of the R and Go implementations.
    
    Args:
        r_output_dir (str): Directory containing R output files
        go_output_dir (str): Directory containing Go output files
        sample_name (str): Sample name
        comparison_dir (str): Directory to store comparison results
        
    Returns:
        dict: Dictionary containing comparison metrics
    """
    os.makedirs(comparison_dir, exist_ok=True)
    
    # Load summary files
    r_summary_file = os.path.join(r_output_dir, f"{sample_name}_read_summary.tsv")
    go_summary_file = os.path.join(go_output_dir, f"{sample_name}_read_summary.tsv")
    
    if not os.path.exists(r_summary_file) or not os.path.exists(go_summary_file):
        print(f"Warning: Summary files not found. R: {os.path.exists(r_summary_file)}, Go: {os.path.exists(go_summary_file)}")
        return {"error": "Summary files not found"}
    
    r_summary = pd.read_csv(r_summary_file, sep='\t')
    go_summary = pd.read_csv(go_summary_file, sep='\t')
    
    # Compare read assignments
    r_assignments = r_summary.groupby('read_assignment').size().reset_index(name='r_count')
    go_assignments = go_summary.groupby('read_assignment').size().reset_index(name='go_count')
    
    # Merge the assignment counts
    merged_assignments = pd.merge(r_assignments, go_assignments, on='read_assignment', how='outer').fillna(0)
    merged_assignments['difference'] = merged_assignments['r_count'] - merged_assignments['go_count']
    merged_assignments['percent_diff'] = (merged_assignments['difference'] / merged_assignments['r_count'].replace(0, 1)) * 100
    
    # Save the comparison results
    merged_assignments.to_csv(os.path.join(comparison_dir, f"{sample_name}_assignment_comparison.csv"), index=False)
    
    # Create a bar chart of the assignments
    plt.figure(figsize=(12, 8))
    
    # Plot the data
    bar_width = 0.35
    index = np.arange(len(merged_assignments))
    
    plt.bar(index, merged_assignments['r_count'], bar_width, label='R Implementation')
    plt.bar(index + bar_width, merged_assignments['go_count'], bar_width, label='Go Implementation')
    
    plt.xlabel('Read Assignment')
    plt.ylabel('Count')
    plt.title(f'Comparison of Read Assignments: {sample_name}')
    plt.xticks(index + bar_width / 2, merged_assignments['read_assignment'], rotation=45)
    plt.legend()
    plt.tight_layout()
    
    # Save the plot
    plt.savefig(os.path.join(comparison_dir, f"{sample_name}_assignment_comparison.png"))
    plt.close()
    
    # Compare individual read assignments
    r_reads = r_summary[['qname', 'read_assignment']].set_index('qname')
    go_reads = go_summary[['qname', 'read_assignment']].set_index('qname')
    
    # Merge the read assignments
    merged_reads = pd.merge(r_reads, go_reads, left_index=True, right_index=True, 
                           how='outer', suffixes=('_r', '_go')).fillna('missing')
    
    # Count the number of matching and differing assignments
    merged_reads['match'] = merged_reads['read_assignment_r'] == merged_reads['read_assignment_go']
    match_count = merged_reads['match'].sum()
    diff_count = len(merged_reads) - match_count
    
    # Save the differing assignments
    diff_reads = merged_reads[~merged_reads['match']]
    diff_reads.to_csv(os.path.join(comparison_dir, f"{sample_name}_differing_assignments.csv"))
    
    # Calculate agreement percentage
    agreement_pct = (match_count / len(merged_reads)) * 100 if len(merged_reads) > 0 else 0
    
    # Create a pie chart of agreement
    plt.figure(figsize=(8, 8))
    plt.pie([match_count, diff_count], labels=['Matching', 'Differing'], 
            autopct='%1.1f%%', startangle=90, colors=['#4CAF50', '#F44336'])
    plt.title(f'Read Assignment Agreement: {sample_name}\n{match_count}/{len(merged_reads)} ({agreement_pct:.2f}%)')
    plt.axis('equal')
    
    # Save the plot
    plt.savefig(os.path.join(comparison_dir, f"{sample_name}_assignment_agreement.png"))
    plt.close()
    
    # Return comparison metrics
    return {
        "total_reads": len(merged_reads),
        "matching_assignments": int(match_count),
        "differing_assignments": int(diff_count),
        "agreement_percentage": float(agreement_pct),
        "assignment_counts": merged_assignments.to_dict(orient='records')
    }

def compare_performance(r_metrics, go_metrics, dataset_name, comparison_dir):
    """
    Compare the performance metrics of the R and Go implementations.
    
    Args:
        r_metrics (dict): Performance metrics for the R implementation
        go_metrics (dict): Performance metrics for the Go implementation
        dataset_name (str): Name of the dataset
        comparison_dir (str): Directory to store comparison results
        
    Returns:
        dict: Dictionary containing performance comparison metrics
    """
    os.makedirs(comparison_dir, exist_ok=True)
    
    # Create a comparison dictionary
    comparison = {
        "dataset": dataset_name,
        "r_implementation": r_metrics,
        "go_implementation": go_metrics,
        "execution_time_ratio": r_metrics["execution_time"] / go_metrics["execution_time"] if go_metrics["execution_time"] > 0 else float('inf'),
        "memory_usage_ratio": r_metrics["max_memory_kb"] / go_metrics["max_memory_kb"] if go_metrics["max_memory_kb"] > 0 else float('inf')
    }
    
    # Save the comparison as JSON
    with open(os.path.join(comparison_dir, f"{dataset_name}_performance_comparison.json"), "w") as f:
        json.dump(comparison, f, indent=2)
    
    # Create a bar chart comparing execution times
    plt.figure(figsize=(10, 6))
    
    metrics = ['execution_time', 'user_time', 'system_time']
    r_values = [r_metrics[m] for m in metrics]
    go_values = [go_metrics[m] for m in metrics]
    
    x = np.arange(len(metrics))
    width = 0.35
    
    plt.bar(x - width/2, r_values, width, label='R Implementation')
    plt.bar(x + width/2, go_values, width, label='Go Implementation')
    
    plt.ylabel('Time (seconds)')
    plt.title(f'Performance Comparison: {dataset_name}')
    plt.xticks(x, ['Total Execution Time', 'User CPU Time', 'System CPU Time'])
    plt.legend()
    
    # Add values on top of bars
    for i, v in enumerate(r_values):
        plt.text(i - width/2, v + 0.1, f'{v:.2f}s', ha='center')
    for i, v in enumerate(go_values):
        plt.text(i + width/2, v + 0.1, f'{v:.2f}s', ha='center')
    
    plt.tight_layout()
    plt.savefig(os.path.join(comparison_dir, f"{dataset_name}_time_comparison.png"))
    plt.close()
    
    # Create a bar chart comparing memory usage
    plt.figure(figsize=(8, 6))
    
    labels = ['R Implementation', 'Go Implementation']
    memory_values = [r_metrics["max_memory_kb"] / 1024, go_metrics["max_memory_kb"] / 1024]  # Convert to MB
    
    plt.bar(labels, memory_values, color=['#3498db', '#2ecc71'])
    plt.ylabel('Memory Usage (MB)')
    plt.title(f'Memory Usage Comparison: {dataset_name}')
    
    # Add values on top of bars
    for i, v in enumerate(memory_values):
        plt.text(i, v + 0.1, f'{v:.2f} MB', ha='center')
    
    plt.tight_layout()
    plt.savefig(os.path.join(comparison_dir, f"{dataset_name}_memory_comparison.png"))
    plt.close()
    
    return comparison

def generate_report(all_comparisons, output_dir):
    """
    Generate a comprehensive report of the comparison results.
    
    Args:
        all_comparisons (dict): Dictionary containing all comparison results
        output_dir (str): Directory to store the report
        
    Returns:
        str: Path to the generated report
    """
    os.makedirs(output_dir, exist_ok=True)
    
    report_path = os.path.join(output_dir, "comparison_report.md")
    
    with open(report_path, "w") as f:
        f.write("# Comparison Report: R vs Go Implementation for Influenza A Serotyping\n\n")
        f.write(f"Generated on: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n\n")
        
        f.write("## Summary\n\n")
        
        # Create a summary table
        f.write("| Dataset | R Execution Time | Go Execution Time | Speedup | R Memory (MB) | Go Memory (MB) | Memory Ratio | Agreement |\n")
        f.write("|---------|------------------|-------------------|---------|---------------|----------------|--------------|----------|\n")
        
        for dataset, comparison in all_comparisons.items():
            perf_comparison = comparison.get("performance", {})
            result_comparison = comparison.get("results", {})
            
            r_time = perf_comparison.get("r_implementation", {}).get("execution_time", 0)
            go_time = perf_comparison.get("go_implementation", {}).get("execution_time", 0)
            speedup = perf_comparison.get("execution_time_ratio", 0)
            
            r_mem = perf_comparison.get("r_implementation", {}).get("max_memory_kb", 0) / 1024  # Convert to MB
            go_mem = perf_comparison.get("go_implementation", {}).get("max_memory_kb", 0) / 1024  # Convert to MB
            mem_ratio = perf_comparison.get("memory_usage_ratio", 0)
            
            agreement = result_comparison.get("agreement_percentage", 0)
            
            f.write(f"| {dataset} | {r_time:.2f}s | {go_time:.2f}s | {speedup:.2f}x | {r_mem:.2f} | {go_mem:.2f} | {mem_ratio:.2f}x | {agreement:.2f}% |\n")
        
        f.write("\n## Detailed Results\n\n")
        
        for dataset, comparison in all_comparisons.items():
            f.write(f"### Dataset: {dataset}\n\n")
            
            # Performance comparison
            perf_comparison = comparison.get("performance", {})
            f.write("#### Performance Comparison\n\n")
            f.write(f"- R Execution Time: {perf_comparison.get('r_implementation', {}).get('execution_time', 0):.2f} seconds\n")
            f.write(f"- Go Execution Time: {perf_comparison.get('go_implementation', {}).get('execution_time', 0):.2f} seconds\n")
            f.write(f"- Speedup: {perf_comparison.get('execution_time_ratio', 0):.2f}x\n\n")
            
            f.write(f"- R Memory Usage: {perf_comparison.get('r_implementation', {}).get('max_memory_kb', 0) / 1024:.2f} MB\n")
            f.write(f"- Go Memory Usage: {perf_comparison.get('go_implementation', {}).get('max_memory_kb', 0) / 1024:.2f} MB\n")
            f.write(f"- Memory Ratio: {perf_comparison.get('memory_usage_ratio', 0):.2f}x\n\n")
            
            f.write(f"![Time Comparison]({dataset}_time_comparison.png)\n\n")
            f.write(f"![Memory Comparison]({dataset}_memory_comparison.png)\n\n")
            
            # Results comparison
            result_comparison = comparison.get("results", {})
            f.write("#### Results Comparison\n\n")
            
            total_reads = result_comparison.get("total_reads", 0)
            matching = result_comparison.get("matching_assignments", 0)
            differing = result_comparison.get("differing_assignments", 0)
            agreement = result_comparison.get("agreement_percentage", 0)
            
            f.write(f"- Total Reads: {total_reads}\n")
            f.write(f"- Matching Assignments: {matching} ({agreement:.2f}%)\n")
            f.write(f"- Differing Assignments: {differing} ({100-agreement:.2f}%)\n\n")
            
            f.write(f"![Assignment Comparison]({dataset}_assignment_comparison.png)\n\n")
            f.write(f"![Assignment Agreement]({dataset}_assignment_agreement.png)\n\n")
            
            # Assignment counts
            f.write("#### Assignment Counts\n\n")
            f.write("| Assignment | R Count | Go Count | Difference | % Difference |\n")
            f.write("|------------|---------|----------|------------|---------------|\n")
            
            for assignment in result_comparison.get("assignment_counts", []):
                f.write(f"| {assignment.get('read_assignment', '')} | {int(assignment.get('r_count', 0))} | {int(assignment.get('go_count', 0))} | {int(assignment.get('difference', 0))} | {assignment.get('percent_diff', 0):.2f}% |\n")
            
            f.write("\n")
        
        f.write("## Conclusion\n\n")
        f.write("Based on the comparison results, we can draw the following conclusions:\n\n")
        
        # Calculate average speedup and memory ratio
        avg_speedup = sum(comp.get("performance", {}).get("execution_time_ratio", 0) for comp in all_comparisons.values()) / len(all_comparisons)
        avg_mem_ratio = sum(comp.get("performance", {}).get("memory_usage_ratio", 0) for comp in all_comparisons.values()) / len(all_comparisons)
        avg_agreement = sum(comp.get("results", {}).get("agreement_percentage", 0) for comp in all_comparisons.values()) / len(all_comparisons)
        
        f.write(f"1. **Performance**: The Go implementation is on average {avg_speedup:.2f}x faster than the R implementation.\n")
        f.write(f"2. **Memory Usage**: The Go implementation uses on average {1/avg_mem_ratio:.2f}x less memory than the R implementation.\n")
        f.write(f"3. **Result Accuracy**: The two implementations agree on {avg_agreement:.2f}% of read assignments.\n\n")
        
        f.write("### Key Differences\n\n")
        f.write("1. **Parallelism**: The Go implementation uses goroutines for parallel processing, while the R script processes data sequentially.\n")
        f.write("2. **Memory Management**: The Go implementation processes data in chunks, which is more memory-efficient for large files.\n")
        f.write("3. **Algorithm Differences**:\n")
        f.write("   - The R script calculates alignment score as `ANI * AF`, while the Go program uses a different formula.\n")
        f.write("   - The read assignment logic differs between implementations, with the R script using a threshold of 0.003 for score differences.\n")
        f.write("4. **Output**: The R implementation generates a visualization PDF, while the Go implementation provides detailed performance metrics.\n\n")
        
        f.write("### Recommendations\n\n")
        f.write("1. **For Large Datasets**: The Go implementation is recommended due to its better performance and memory efficiency.\n")
        f.write("2. **For Consistency**: The differences in read assignments should be investigated further to ensure both implementations produce consistent results.\n")
        f.write("3. **For Future Development**: Consider standardizing the alignment score calculation and read assignment logic between the two implementations.\n")
    
    # Copy images to the report directory
    for dataset in all_comparisons.keys():
        for img_type in ["time_comparison", "memory_comparison", "assignment_comparison", "assignment_agreement"]:
            src = os.path.join(output_dir, dataset, f"{dataset}_{img_type}.png")
            if os.path.exists(src):
                shutil.copy(src, os.path.join(output_dir, f"{dataset}_{img_type}.png"))
    
    return report_path

def main():
    """
    Main function to run the comparison.
    """
    parser = argparse.ArgumentParser(description="Compare R and Go implementations for Influenza A serotyping")
    parser.add_argument("--datasets", nargs="+", choices=TEST_DATASETS.keys(), default=["small"],
                        help="Datasets to use for comparison")
    parser.add_argument("--score-threshold", type=float, default=0.8,
                        help="Score threshold for read assignment")
    parser.add_argument("--workers", type=int, default=4,
                        help="Number of worker goroutines for Go implementation")
    parser.add_argument("--chunk-size", type=int, default=1000,
                        help="Chunk size for Go implementation")
    args = parser.parse_args()
    
    # Create timestamp for this run
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    run_dir = os.path.join(RESULTS_DIR, timestamp)
    os.makedirs(run_dir, exist_ok=True)
    
    # Save the arguments
    with open(os.path.join(run_dir, "args.json"), "w") as f:
        json.dump(vars(args), f, indent=2)
    
    all_comparisons = {}
    
    for dataset_name in args.datasets:
        print(f"\n=== Processing dataset: {dataset_name} ===\n")
        
        dataset_path = TEST_DATASETS[dataset_name]
        dataset_dir = os.path.join(run_dir, dataset_name)
        os.makedirs(dataset_dir, exist_ok=True)
        
        # Create output directories
        r_output_dir = os.path.join(dataset_dir, "r_output")
        go_output_dir = os.path.join(dataset_dir, "go_output")
        comparison_dir = os.path.join(dataset_dir)
        
        # Run R implementation
        print(f"Running R implementation on {dataset_name}...")
        r_metrics = run_r_implementation(
            dataset_path, 
            r_output_dir, 
            f"{dataset_name}", 
            args.score_threshold
        )
        
        # Run Go implementation
        print(f"Running Go implementation on {dataset_name}...")
        go_metrics = run_go_implementation(
            dataset_path, 
            go_output_dir, 
            f"{dataset_name}", 
            args.score_threshold,
            args.workers,
            args.chunk_size
        )
        
        # Compare results
        print(f"Comparing results for {dataset_name}...")
        result_comparison = compare_results(
            r_output_dir, 
            go_output_dir, 
            f"{dataset_name}", 
            comparison_dir
        )
        
        # Compare performance
        print(f"Comparing performance for {dataset_name}...")
        perf_comparison = compare_performance(
            r_metrics, 
            go_metrics, 
            dataset_name, 
            comparison_dir
        )
        
        all_comparisons[dataset_name] = {
            "performance": perf_comparison,
            "results": result_comparison
        }
    
    # Generate report
    print("\nGenerating comparison report...")
    report_path = generate_report(all_comparisons, run_dir)
    
    print(f"\nComparison complete! Report saved to: {report_path}")
    
    # Save all comparison data
    with open(os.path.join(run_dir, "all_comparisons.json"), "w") as f:
        # Convert non-serializable objects to strings
        serializable_comparisons = json.loads(
            json.dumps(all_comparisons, default=lambda o: str(o))
        )
        json.dump(serializable_comparisons, f, indent=2)
    
    return 0

if __name__ == "__main__":
    sys.exit(main())