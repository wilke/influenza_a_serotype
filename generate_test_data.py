#!/usr/bin/env pythom

import os
import numpy as np
import pandas as pd
import random
import argparse
from pathlib import Path

def parse_args():
    parser = argparse.ArgumentParser(description='Generate test data for memory usage analysis')
    parser.add_argument('output_dir', help='Directory to save the test data')
    parser.add_argument('--samples', type=int, default=3, help='Number of sample files to generate')
    parser.add_argument('--time-points', type=int, default=100, help='Number of time points per sample')
    parser.add_argument('--verbose', '-v', action='store_true', help='Enable verbose output')
    return parser.parse_args()

def generate_memory_pattern(pattern_type, time_points, base_memory=50000, noise_level=5000):
    """Generate a memory usage pattern of a specific type."""
    time = np.arange(time_points)
    noise = np.random.normal(0, noise_level, time_points)
    
    if pattern_type == 'stable':
        # Stable memory usage with small fluctuations
        memory = base_memory + noise
    
    elif pattern_type == 'growing':
        # Linearly growing memory usage (potential memory leak)
        growth_rate = random.uniform(100, 500)  # KB per time point
        memory = base_memory + growth_rate * time + noise
    
    elif pattern_type == 'fluctuating':
        # Memory usage with periodic fluctuations (e.g., garbage collection)
        period = random.randint(10, 30)
        amplitude = random.uniform(10000, 30000)
        memory = base_memory + amplitude * np.sin(2 * np.pi * time / period) + noise
    
    elif pattern_type == 'spiky':
        # Memory usage with occasional spikes
        memory = base_memory + noise
        num_spikes = random.randint(3, 8)
        for _ in range(num_spikes):
            spike_pos = random.randint(0, time_points - 1)
            spike_height = random.uniform(50000, 100000)
            spike_width = random.randint(1, 5)
            
            # Create a spike centered at spike_pos
            for i in range(max(0, spike_pos - spike_width), min(time_points, spike_pos + spike_width + 1)):
                distance = abs(i - spike_pos)
                memory[i] += spike_height * (1 - distance / (spike_width + 1))
    
    else:
        raise ValueError(f"Unknown pattern type: {pattern_type}")
    
    # Ensure memory values are positive
    memory = np.maximum(memory, 1000)
    
    return memory.astype(int)

def generate_sample_data(output_dir, sample_name, implementation, pattern_type, time_points, verbose=False):
    """Generate memory usage data for a sample and implementation."""
    # Create directory structure
    impl_dir = os.path.join(output_dir, implementation, sample_name)
    os.makedirs(impl_dir, exist_ok=True)
    
    # Generate memory usage pattern
    base_memory = random.randint(30000, 80000)  # Base memory in KB
    memory_values = generate_memory_pattern(pattern_type, time_points, base_memory)
    
    # Write memory usage data to file
    memory_file = os.path.join(impl_dir, 'memory_usage.txt')
    with open(memory_file, 'w') as f:
        for value in memory_values:
            f.write(f"{value}\n")
    
    if verbose:
        print(f"Generated {pattern_type} memory pattern for {implementation}/{sample_name}")
        print(f"  Base memory: {base_memory} KB")
        print(f"  Peak memory: {memory_values.max()} KB")
        print(f"  File: {memory_file}")
    
    return {
        'implementation': implementation,
        'sample': sample_name,
        'pattern_type': pattern_type,
        'base_memory_kb': base_memory,
        'peak_memory_kb': memory_values.max(),
        'avg_memory_kb': memory_values.mean(),
        'min_memory_kb': memory_values.min(),
        'time_points': time_points
    }

def generate_summary_data(output_dir, sample_data, verbose=False):
    """Generate a summary.tsv file with sample data."""
    summary_file = os.path.join(output_dir, 'summary.tsv')
    
    # Create summary data
    summary_rows = []
    
    for data in sample_data:
        impl = data['implementation']
        sample = data['sample']
        
        # Generate random read counts
        h1n1_reads = random.randint(1000, 5000)
        h3n2_reads = random.randint(1000, 5000)
        ambiguous_reads = random.randint(100, 1000)
        
        # Generate random runtime
        runtime = random.uniform(5, 30)
        
        # Add row to summary
        summary_rows.append({
            'file': f"{sample}.paf",
            'implementation': impl,
            'runtime_seconds': runtime,
            'max_memory_kb': data['peak_memory_kb'],
            'exit_code': 0,
            'H1N1_reads': h1n1_reads,
            'H3N2_reads': h3n2_reads,
            'ambiguous_reads': ambiguous_reads
        })
    
    # Create DataFrame and write to TSV
    summary_df = pd.DataFrame(summary_rows)
    summary_df.to_csv(summary_file, sep='\t', index=False)
    
    if verbose:
        print(f"Generated summary file: {summary_file}")
        print(summary_df)
    
    return summary_file

def main():
    args = parse_args()
    output_dir = args.output_dir
    num_samples = args.samples
    time_points = args.time_points
    verbose = args.verbose
    
    # Create output directory
    os.makedirs(output_dir, exist_ok=True)
    
    # Define implementations and pattern types
    implementations = ['r_implementation', 'go_default', 'go_r_algorithm']
    pattern_types = {
        'r_implementation': 'fluctuating',  # R tends to have garbage collection cycles
        'go_default': 'stable',             # Go with good memory management
        'go_r_algorithm': 'growing'         # Simulating a potential memory leak
    }
    
    # Generate sample names
    sample_names = [f"sample_{i+1}" for i in range(num_samples)]
    
    # Generate data for each sample and implementation
    sample_data = []
    
    for sample in sample_names:
        for impl in implementations:
            pattern = pattern_types[impl]
            data = generate_sample_data(output_dir, sample, impl, pattern, time_points, verbose)
            sample_data.append(data)
    
    # Generate summary file
    summary_file = generate_summary_data(output_dir, sample_data, verbose)
    
    print(f"Test data generation completed. Data saved to {output_dir}")
    print(f"Run the memory analysis script with: python analyze_memory_usage.py {output_dir}")

if __name__ == "__main__":
    main()