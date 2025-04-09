#!/usr/bin/env pythom

import os
import sys
import glob
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
from pathlib import Path
import argparse
from sklearn.metrics import confusion_matrix, cohen_kappa_score
import itertools
import json
from collections import defaultdict

def parse_args():
    parser = argparse.ArgumentParser(description='Compare outputs of PAF processor implementations')
    parser.add_argument('output_dir', help='Directory containing comparison results')
    parser.add_argument('--report-dir', help='Directory to save the output comparison report (default: output_dir/reports)')
    parser.add_argument('--verbose', '-v', action='store_true', help='Enable verbose output')
    return parser.parse_args()

def log(message, verbose=False):
    if verbose:
        print(message)

def find_serotype_files(output_dir, implementation, serotype, verbose=False):
    """Find all serotype files for a specific implementation and serotype."""
    pattern = os.path.join(output_dir, implementation, '*', f'*_{serotype}.txt')
    files = glob.glob(pattern)
    log(f"Found {len(files)} {serotype} files for {implementation}: {files}", verbose)
    return files

def parse_serotype_file(file_path, verbose=False):
    """Parse a serotype file and return a set of read names."""
    try:
        with open(file_path, 'r') as f:
            read_names = set(line.strip() for line in f if line.strip())
        
        sample_name = os.path.basename(os.path.dirname(file_path))
        serotype = os.path.basename(file_path).split('_')[-1].split('.')[0]
        
        log(f"Parsed {serotype} file {file_path}: {len(read_names)} reads", verbose)
        return sample_name, serotype, read_names
    except Exception as e:
        print(f"Error parsing serotype file {file_path}: {e}")
        return None, None, set()

def collect_read_assignments(output_dir, implementations, serotypes, verbose=False):
    """Collect read assignments for all implementations, samples, and serotypes."""
    read_assignments = {}
    
    for impl in implementations:
        read_assignments[impl] = {}
        
        for serotype in serotypes:
            files = find_serotype_files(output_dir, impl, serotype, verbose)
            
            for file in files:
                sample_name, serotype_name, read_names = parse_serotype_file(file, verbose)
                
                if sample_name and read_names:
                    if sample_name not in read_assignments[impl]:
                        read_assignments[impl][sample_name] = {}
                    
                    read_assignments[impl][sample_name][serotype_name] = read_names
    
    return read_assignments

def create_assignment_dataframe(read_assignments, implementations, verbose=False):
    """Create a DataFrame with read assignments for all implementations."""
    assignment_data = []
    
    # Collect all unique samples
    all_samples = set()
    for impl in implementations:
        all_samples.update(read_assignments[impl].keys())
    
    # Collect all unique reads across all implementations and samples
    all_reads = set()
    for impl in implementations:
        for sample in read_assignments[impl]:
            for serotype in read_assignments[impl][sample]:
                all_reads.update(read_assignments[impl][sample][serotype])
    
    log(f"Found {len(all_samples)} samples and {len(all_reads)} unique reads", verbose)
    
    # Create assignment data
    for sample in all_samples:
        for read in all_reads:
            row = {'sample': sample, 'read': read}
            
            for impl in implementations:
                if sample in read_assignments[impl]:
                    # Find which serotype this read belongs to in this implementation
                    assigned_serotype = 'unknown'
                    for serotype, reads in read_assignments[impl][sample].items():
                        if read in reads:
                            assigned_serotype = serotype
                            break
                    
                    row[impl] = assigned_serotype
                else:
                    row[impl] = 'unknown'
            
            assignment_data.append(row)
    
    # Create DataFrame
    df = pd.DataFrame(assignment_data)
    
    log(f"Created assignment DataFrame with {len(df)} rows", verbose)
    return df

def main():
    args = parse_args()
    output_dir = args.output_dir
    report_dir = args.report_dir or os.path.join(output_dir, 'reports')
    verbose = args.verbose
    
    # Create report directory
    os.makedirs(report_dir, exist_ok=True)
    
    # Define implementations and serotypes
    implementations = ['r_implementation', 'go_default', 'go_r_algorithm']
    serotypes = ['H1N1', 'H3N2', 'ambiguous']
    
    # Collect read assignments
    read_assignments = collect_read_assignments(output_dir, implementations, serotypes, verbose)
    
    # Create assignment DataFrame
    assignment_df = create_assignment_dataframe(read_assignments, implementations, verbose)
    
    # Import the utility functions
    try:
        from compare_paf_processor_utils import (
            calculate_agreement_metrics,
            plot_confusion_matrices,
            plot_agreement_metrics,
            plot_serotype_distribution,
            plot_read_assignment_changes,
            generate_markdown_report,
            generate_html_report
        )
        
        # Calculate agreement metrics
        metrics = calculate_agreement_metrics(assignment_df, implementations, verbose)
        
        # Generate visualizations
        plot_confusion_matrices(metrics, report_dir, verbose)
        plot_agreement_metrics(metrics, report_dir, verbose)
        plot_serotype_distribution(assignment_df, implementations, report_dir, verbose)
        plot_read_assignment_changes(assignment_df, implementations, report_dir, verbose)
        
        # Generate reports
        markdown_path = generate_markdown_report(assignment_df, metrics, implementations, report_dir, verbose)
        html_path = generate_html_report(markdown_path, verbose)
        
        print(f"Output comparison completed. Reports generated in {report_dir}")
        print(f"Markdown report: {markdown_path}")
        if html_path:
            print(f"HTML report: {html_path}")
    
    except ImportError:
        print("Error: Could not import utility functions from compare_paf_processor_utils.py")
        print("Please make sure the file exists in the same directory as this script.")
        sys.exit(1)

if __name__ == "__main__":
    main()
