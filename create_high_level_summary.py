#!/usr/bin/env python

import os
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import json
import sys
from pathlib import Path
import argparse
import numpy as np
from collections import defaultdict

def parse_args():
    parser = argparse.ArgumentParser(description='Generate a high-level summary of the PAF processor comparison results.')
    parser.add_argument('output_dir', help='Directory containing the comparison results')
    parser.add_argument('--summary-file', help='Output file for the summary report', default=None)
    return parser.parse_args()

def analyze_input_files(output_dir):
    """
    Analyze all input PAF files to extract statistics about reads and alignments.
    """
    summary_file = os.path.join(output_dir, "summary.tsv")
    if not os.path.exists(summary_file):
        print(f"Error: Summary file not found at {summary_file}")
        return {}
    
    # Read the summary data
    df = pd.read_csv(summary_file, sep='\t')
    
    # Get unique file names from the summary dataframe
    unique_files = df['file'].unique()
    
    input_stats = {}
    
    for file_name in unique_files:
        # Look for the input file in various potential locations
        potential_paths = [
            os.path.join(output_dir, "..", "test_data", file_name),
            os.path.join(output_dir, "..", file_name),
            os.path.join("test_data", file_name),
            # Add more potential paths if needed
        ]
        
        file_path = None
        for path in potential_paths:
            if os.path.exists(path):
                file_path = path
                break
        
        if file_path is None:
            print(f"Warning: Could not find input file {file_name} for analysis")
            continue
        
        # Analyze the PAF file
        try:
            print(f"Analyzing input file: {file_path}")
            qnames = set()
            qname_hits = {}
            total_lines = 0
            
            with open(file_path, 'r') as f:
                for line in f:
                    total_lines += 1
                    parts = line.strip().split('\t')
                    if len(parts) >= 1:
                        qname = parts[0]
                        qnames.add(qname)
                        qname_hits[qname] = qname_hits.get(qname, 0) + 1
            
            # Calculate statistics
            unique_qnames = len(qnames)
            total_hits = sum(qname_hits.values())
            avg_hits_per_qname = total_hits / unique_qnames if unique_qnames > 0 else 0
            max_hits = max(qname_hits.values()) if qname_hits else 0
            min_hits = min(qname_hits.values()) if qname_hits else 0
            
            # Calculate distribution of hits per qname
            hit_counts = list(qname_hits.values())
            hit_distribution = {}
            for i in range(1, max(10, max_hits) + 1):
                if i <= 10 or i == max_hits:  # Show details for 1-10 hits, plus the max
                    hit_distribution[i] = sum(1 for hits in hit_counts if hits == i)
            
            # Calculate percentiles
            percentiles = {}
            for p in [25, 50, 75, 90, 95, 99]:
                percentiles[f"p{p}"] = np.percentile(hit_counts, p) if hit_counts else 0
            
            # Store statistics
            input_stats[file_name] = {
                "file_size_bytes": os.path.getsize(file_path),
                "total_lines": total_lines,
                "unique_qnames": unique_qnames,
                "total_hits": total_hits,
                "avg_hits_per_qname": avg_hits_per_qname,
                "max_hits": max_hits,
                "min_hits": min_hits,
                "hit_distribution": hit_distribution,
                "percentiles": percentiles
            }
            
        except Exception as e:
            print(f"Error analyzing input file {file_path}: {e}")
            continue
    
    return input_stats

def get_serotype_assignments(output_dir):
    """
    Extract serotype assignments from all implementations.
    """
    implementations = ["r_implementation", "go_default", "go_r_algorithm"]
    
    assignments = {}
    all_serotypes = set()
    
    for impl in implementations:
        impl_dir = os.path.join(output_dir, impl)
        if not os.path.exists(impl_dir):
            print(f"Warning: Implementation directory {impl_dir} not found")
            continue
        
        assignments[impl] = {}
        
        # Walk through all files in the implementation directory
        for root, _, files in os.walk(impl_dir):
            for file in files:
                if file.endswith(".txt") and "_" in file and not file.endswith("_read_summary.txt") and not file.endswith("_usage.txt"):
                    file_path = os.path.join(root, file)
                    
                    # Extract sample and serotype from filename (sample_serotype.txt)
                    parts = file.rsplit("_", 1)
                    if len(parts) == 2:
                        sample = parts[0]
                        serotype_with_ext = parts[1]
                        serotype = serotype_with_ext.replace(".txt", "")
                        
                        # Skip non-serotype files
                        if serotype in ["read_summary", "usage"]:
                            continue
                        
                        # Count lines in the file to get read count
                        try:
                            with open(file_path, 'r') as f:
                                read_count = sum(1 for _ in f)
                            
                            if sample not in assignments[impl]:
                                assignments[impl][sample] = {}
                            
                            assignments[impl][sample][serotype] = read_count
                            
                            # Add to set of all serotypes
                            if serotype != "ambiguous":
                                all_serotypes.add(serotype)
                                
                        except Exception as e:
                            print(f"Error reading {file_path}: {e}")
    
    return assignments, sorted(list(all_serotypes))

def calculate_runtime_memory_stats(output_dir):
    """
    Calculate detailed runtime and memory usage statistics.
    """
    summary_file = os.path.join(output_dir, "summary.tsv")
    if not os.path.exists(summary_file):
        print(f"Error: Summary file not found at {summary_file}")
        return {}
    
    # Read the summary data
    df = pd.read_csv(summary_file, sep='\t')
    
    # Group by implementation
    runtime_by_impl = df.groupby('implementation')['runtime_seconds']
    memory_by_impl = df.groupby('implementation')['max_memory_kb']
    
    stats = {}
    
    for impl in df['implementation'].unique():
        stats[impl] = {
            "runtime": {
                "min": runtime_by_impl.min()[impl],
                "max": runtime_by_impl.max()[impl],
                "mean": runtime_by_impl.mean()[impl],
                "median": runtime_by_impl.median()[impl],
                "std": runtime_by_impl.std()[impl]
            },
            "memory": {
                "min": memory_by_impl.min()[impl],
                "max": memory_by_impl.max()[impl],
                "mean": memory_by_impl.mean()[impl],
                "median": memory_by_impl.median()[impl],
                "std": memory_by_impl.std()[impl]
            }
        }
    
    return stats

def generate_summary_report(output_dir, input_stats, assignments, serotypes, performance_stats):
    """
    Generate a comprehensive summary report.
    """
    # Create output directory
    report_dir = os.path.join(output_dir, "reports")
    os.makedirs(report_dir, exist_ok=True)
    
    # Summary report path
    report_path = os.path.join(report_dir, "high_level_summary.md")
    
    with open(report_path, 'w') as f:
        f.write("# High-Level Summary of PAF Processor Comparison\n\n")
        f.write(f"*Generated on: {pd.Timestamp.now().strftime('%Y-%m-%d %H:%M:%S')}*\n\n")
        
        # Input Files Summary
        f.write("## Input Files Summary\n\n")
        if input_stats:
            f.write("| File | Size (MB) | Total Lines | Unique QNames | Total Hits | Avg Hits/QName | Max Hits | Min Hits |\n")
            f.write("|------|-----------|-------------|--------------|------------|----------------|----------|----------|\n")
            
            for file_name, stats in sorted(input_stats.items()):
                size_mb = stats["file_size_bytes"] / (1024 * 1024)
                
                f.write(f"| {file_name} | {size_mb:.2f} | {stats['total_lines']} | {stats['unique_qnames']} | ")
                f.write(f"{stats['total_hits']} | {stats['avg_hits_per_qname']:.2f} | {stats['max_hits']} | {stats['min_hits']} |\n")
            
            # Add hit distribution
            f.write("\n### Hits Distribution\n\n")
            for file_name, stats in sorted(input_stats.items()):
                f.write(f"**{file_name}**:\n\n")
                
                # Format distribution
                f.write("| Hits per QName | Count | Percentage |\n")
                f.write("|---------------|-------|------------|\n")
                
                total_qnames = stats["unique_qnames"]
                dist = stats["hit_distribution"]
                
                for hits, count in sorted(dist.items()):
                    percentage = (count / total_qnames) * 100 if total_qnames > 0 else 0
                    f.write(f"| {hits} | {count} | {percentage:.2f}% |\n")
                
                # Add percentiles
                f.write("\nPercentiles:\n\n")
                percentiles = stats["percentiles"]
                for p, value in percentiles.items():
                    f.write(f"- {p}: {value:.1f} hits\n")
                
                f.write("\n")
        else:
            f.write("No input file statistics available.\n\n")
        
        # Performance Stats
        f.write("## Performance Statistics\n\n")
        if performance_stats:
            f.write("### Runtime (seconds)\n\n")
            f.write("| Implementation | Min | Max | Mean | Median | Std |\n")
            f.write("|---------------|-----|-----|------|--------|-----|\n")
            
            for impl, stats in sorted(performance_stats.items()):
                runtime = stats["runtime"]
                f.write(f"| {impl} | {runtime['min']:.2f} | {runtime['max']:.2f} | ")
                f.write(f"{runtime['mean']:.2f} | {runtime['median']:.2f} | {runtime['std']:.2f} |\n")
            
            f.write("\n### Memory Usage (KB)\n\n")
            f.write("| Implementation | Min | Max | Mean | Median | Std |\n")
            f.write("|---------------|-----|-----|------|--------|-----|\n")
            
            for impl, stats in sorted(performance_stats.items()):
                memory = stats["memory"]
                f.write(f"| {impl} | {memory['min']:.0f} | {memory['max']:.0f} | ")
                f.write(f"{memory['mean']:.0f} | {memory['median']:.0f} | {memory['std']:.0f} |\n")
            
            # Calculate relative performance
            f.write("\n### Relative Performance\n\n")
            
            # Find slowest and highest memory implementations
            slowest_impl = max(performance_stats.items(), key=lambda x: x[1]["runtime"]["mean"])[0]
            highest_mem_impl = max(performance_stats.items(), key=lambda x: x[1]["memory"]["max"])[0]
            
            max_runtime = performance_stats[slowest_impl]["runtime"]["mean"]
            max_memory = performance_stats[highest_mem_impl]["memory"]["max"]
            
            f.write("| Implementation | Speed Ratio | Memory Efficiency Ratio |\n")
            f.write("|---------------|------------|------------------------|\n")
            
            for impl, stats in sorted(performance_stats.items()):
                speed_ratio = max_runtime / stats["runtime"]["mean"]
                memory_ratio = max_memory / stats["memory"]["max"]
                f.write(f"| {impl} | {speed_ratio:.2f}x | {memory_ratio:.2f}x |\n")
        else:
            f.write("No performance statistics available.\n\n")
        
        # Serotype Assignments
        f.write("## Serotype Assignments\n\n")
        if assignments:
            implementations = sorted(assignments.keys())
            
            # Combine all samples
            all_samples = set()
            for impl_data in assignments.values():
                all_samples.update(impl_data.keys())
            
            # For each sample
            for sample in sorted(all_samples):
                f.write(f"### Sample: {sample}\n\n")
                
                # Create a table with counts for each serotype
                f.write("| Serotype | " + " | ".join(implementations) + " |\n")
                f.write("|----------|" + "|".join(["---------" for _ in implementations]) + "|\n")
                
                # Add ambiguous counts
                f.write("| ambiguous |")
                for impl in implementations:
                    count = assignments.get(impl, {}).get(sample, {}).get("ambiguous", 0)
                    f.write(f" {count} |")
                f.write("\n")
                
                # Add counts for each serotype
                for serotype in serotypes:
                    f.write(f"| {serotype} |")
                    for impl in implementations:
                        count = assignments.get(impl, {}).get(sample, {}).get(serotype, 0)
                        f.write(f" {count} |")
                    f.write("\n")
                
                # Add total row
                f.write("| **Total** |")
                for impl in implementations:
                    total = sum(assignments.get(impl, {}).get(sample, {}).values())
                    f.write(f" **{total}** |")
                f.write("\n\n")
                
                # Add percentage table
                f.write("#### Percentage Distribution\n\n")
                f.write("| Serotype | " + " | ".join(implementations) + " |\n")
                f.write("|----------|" + "|".join(["---------" for _ in implementations]) + "|\n")
                
                # Add percentages
                for serotype in ["ambiguous"] + serotypes:
                    f.write(f"| {serotype} |")
                    for impl in implementations:
                        impl_data = assignments.get(impl, {}).get(sample, {})
                        total = sum(impl_data.values())
                        count = impl_data.get(serotype, 0)
                        percentage = (count / total) * 100 if total > 0 else 0
                        f.write(f" {percentage:.2f}% |")
                    f.write("\n")
                
                f.write("\n")
            
            # Overall comparison across implementations
            f.write("### Implementation Comparison\n\n")
            
            # Calculate total assignments per serotype across all samples
            overall_counts = defaultdict(lambda: defaultdict(int))
            
            for impl in implementations:
                for sample, serotype_counts in assignments.get(impl, {}).items():
                    for serotype, count in serotype_counts.items():
                        overall_counts[impl][serotype] += count
            
            # Create table with total counts
            f.write("| Serotype | " + " | ".join(implementations) + " |\n")
            f.write("|----------|" + "|".join(["---------" for _ in implementations]) + "|\n")
            
            total_by_impl = defaultdict(int)
            
            # Add ambiguous row
            f.write("| ambiguous |")
            for impl in implementations:
                count = overall_counts[impl].get("ambiguous", 0)
                total_by_impl[impl] += count
                f.write(f" {count} |")
            f.write("\n")
            
            # Add serotype rows
            for serotype in serotypes:
                f.write(f"| {serotype} |")
                for impl in implementations:
                    count = overall_counts[impl].get(serotype, 0)
                    total_by_impl[impl] += count
                    f.write(f" {count} |")
                f.write("\n")
            
            # Add total row
            f.write("| **Total** |")
            for impl in implementations:
                f.write(f" **{total_by_impl[impl]}** |")
            f.write("\n\n")
            
            # Add percentage table
            f.write("#### Percentage Distribution\n\n")
            f.write("| Serotype | " + " | ".join(implementations) + " |\n")
            f.write("|----------|" + "|".join(["---------" for _ in implementations]) + "|\n")
            
            # Add percentages
            for serotype in ["ambiguous"] + serotypes:
                f.write(f"| {serotype} |")
                for impl in implementations:
                    count = overall_counts[impl].get(serotype, 0)
                    percentage = (count / total_by_impl[impl]) * 100 if total_by_impl[impl] > 0 else 0
                    f.write(f" {percentage:.2f}% |")
                f.write("\n")
            
            f.write("\n")
            
        else:
            f.write("No serotype assignment data available.\n\n")
        
        # Include a conclusion section
        f.write("## Conclusion\n\n")
        f.write("This report provides a high-level summary of the PAF processor comparison results.\n\n")
        
        if performance_stats:
            # Find fastest implementation
            fastest_impl = min(performance_stats.items(), key=lambda x: x[1]["runtime"]["mean"])[0]
            most_efficient_impl = min(performance_stats.items(), key=lambda x: x[1]["memory"]["max"])[0]
            
            f.write(f"Based on the performance statistics, the **{fastest_impl}** implementation ")
            f.write("is the fastest in terms of runtime, ")
            f.write(f"while the **{most_efficient_impl}** implementation ")
            f.write("is the most memory-efficient.\n\n")
        
        if assignments:
            f.write("The serotype assignments show ")
            
            # Check if the implementations agree
            agreement_level = "high agreement" if len(set(tuple(sorted(overall_counts[impl].items())) for impl in implementations)) == 1 else "some differences"
            
            f.write(f"{agreement_level} between the different implementations, ")
            f.write("which suggests that the algorithms are producing consistent results ")
            f.write("despite differences in implementation details.\n")
    
    print(f"High-level summary report generated at {report_path}")
    return report_path

def main():
    args = parse_args()
    output_dir = args.output_dir
    
    if not os.path.isdir(output_dir):
        print(f"Error: Output directory '{output_dir}' does not exist.")
        sys.exit(1)
    
    print(f"Analyzing comparison results in {output_dir}...")
    
    # Analyze input files
    print("Analyzing input files...")
    input_stats = analyze_input_files(output_dir)
    
    # Get serotype assignments
    print("Extracting serotype assignments...")
    assignments, serotypes = get_serotype_assignments(output_dir)
    
    # Calculate runtime and memory usage statistics
    print("Calculating performance statistics...")
    performance_stats = calculate_runtime_memory_stats(output_dir)
    
    # Generate summary report
    print("Generating summary report...")
    summary_path = generate_summary_report(output_dir, input_stats, assignments, serotypes, performance_stats)
    
    print(f"Summary report generated at {summary_path}")

if __name__ == "__main__":
    main()