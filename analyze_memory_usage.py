#!/usr/bin/env pythom

import os
import sys
import glob
import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import seaborn as sns
from pathlib import Path
import re
import argparse

def parse_args():
    parser = argparse.ArgumentParser(description='Analyze memory usage patterns in PAF processor implementations')
    parser.add_argument('output_dir', help='Directory containing comparison results')
    parser.add_argument('--report-dir', help='Directory to save the memory analysis report (default: output_dir/memory_analysis)')
    parser.add_argument('--verbose', '-v', action='store_true', help='Enable verbose output')
    return parser.parse_args()

def log(message, verbose=False):
    if verbose:
        print(message)

def find_memory_files(output_dir, implementation, verbose=False):
    """Find all memory usage files for a specific implementation."""
    pattern = os.path.join(output_dir, implementation, '*', 'memory_usage.txt')
    files = glob.glob(pattern)
    log(f"Found {len(files)} memory usage files for {implementation}: {files}", verbose)
    return files

def parse_memory_file(file_path, verbose=False):
    """Parse a memory usage file and return a DataFrame with memory usage over time."""
    try:
        # Read memory usage data (one value per line, in KB)
        with open(file_path, 'r') as f:
            memory_values = [int(line.strip()) for line in f if line.strip()]
        
        # Create a DataFrame with memory usage over time
        df = pd.DataFrame({
            'time_point': range(len(memory_values)),
            'memory_kb': memory_values
        })
        
        # Extract sample name from file path
        sample_name = os.path.basename(os.path.dirname(file_path))
        df['sample'] = sample_name
        
        log(f"Parsed memory file {file_path}: {len(memory_values)} data points", verbose)
        return df
    except Exception as e:
        print(f"Error parsing memory file {file_path}: {e}")
        return None

def calculate_memory_metrics(df, verbose=False):
    """Calculate memory usage metrics for a DataFrame of memory usage data."""
    if df.empty:
        return {}
    
    # Calculate basic metrics
    peak_memory = df['memory_kb'].max()
    avg_memory = df['memory_kb'].mean()
    min_memory = df['memory_kb'].min()
    
    # Calculate memory growth rate (linear regression)
    x = df['time_point'].values
    y = df['memory_kb'].values
    
    if len(x) > 1:
        # Use numpy's polyfit to get the slope (growth rate)
        slope, intercept = np.polyfit(x, y, 1)
        growth_rate = slope  # KB per time point
    else:
        growth_rate = 0
    
    # Calculate memory volatility (standard deviation)
    volatility = df['memory_kb'].std()
    
    # Calculate memory stability (coefficient of variation)
    if avg_memory > 0:
        stability = volatility / avg_memory
    else:
        stability = 0
    
    metrics = {
        'peak_memory_kb': peak_memory,
        'avg_memory_kb': avg_memory,
        'min_memory_kb': min_memory,
        'growth_rate_kb': growth_rate,
        'volatility_kb': volatility,
        'stability_ratio': stability
    }
    
    log(f"Calculated memory metrics: {metrics}", verbose)
    return metrics

def detect_memory_issues(metrics, verbose=False):
    """Detect potential memory issues based on metrics."""
    issues = []
    
    # Check for high growth rate (potential memory leak)
    if metrics['growth_rate_kb'] > 100:  # Arbitrary threshold, adjust as needed
        issues.append("High memory growth rate detected (potential memory leak)")
    
    # Check for high volatility (potential memory fragmentation)
    if metrics['stability_ratio'] > 0.5:  # Arbitrary threshold, adjust as needed
        issues.append("High memory volatility detected (potential memory fragmentation)")
    
    log(f"Detected memory issues: {issues}", verbose)
    return issues

def calculate_memory_efficiency(memory_metrics, summary_df, verbose=False):
    """Calculate memory efficiency metrics by combining memory metrics with summary data."""
    # Group summary data by implementation
    grouped = summary_df.groupby('implementation').agg({
        'H1N1_reads': 'sum',
        'H3N2_reads': 'sum',
        'ambiguous_reads': 'sum',
        'runtime_seconds': 'mean'
    }).reset_index()
    
    # Calculate total reads
    grouped['total_reads'] = grouped['H1N1_reads'] + grouped['H3N2_reads'] + grouped['ambiguous_reads']
    
    # Merge with memory metrics
    efficiency_df = pd.DataFrame(memory_metrics).reset_index().rename(columns={'index': 'implementation'})
    efficiency_df = pd.merge(efficiency_df, grouped, on='implementation')
    
    # Calculate efficiency metrics
    efficiency_df['reads_per_mb'] = efficiency_df['total_reads'] / (efficiency_df['peak_memory_kb'] / 1024)
    efficiency_df['reads_per_mb_per_second'] = efficiency_df['reads_per_mb'] / efficiency_df['runtime_seconds']
    
    log(f"Calculated memory efficiency metrics: {efficiency_df.to_dict('records')}", verbose)
    return efficiency_df

def plot_memory_usage_over_time(memory_data, report_dir, verbose=False):
    """Plot memory usage over time for each implementation and sample."""
    os.makedirs(os.path.join(report_dir, 'images'), exist_ok=True)
    
    # Set the style
    sns.set(style="whitegrid")
    
    # Plot memory usage over time for each implementation
    plt.figure(figsize=(12, 8))
    
    for impl, data in memory_data.items():
        if not data.empty:
            # Calculate the average memory usage at each time point across all samples
            avg_data = data.groupby('time_point')['memory_kb'].mean().reset_index()
            plt.plot(avg_data['time_point'], avg_data['memory_kb'], label=impl)
    
    plt.title('Memory Usage Over Time (Average Across Samples)')
    plt.xlabel('Time Point')
    plt.ylabel('Memory Usage (KB)')
    plt.legend()
    plt.tight_layout()
    plt.savefig(os.path.join(report_dir, 'images', 'memory_usage_over_time.png'))
    plt.close()
    
    # Plot memory usage over time for each sample and implementation
    for impl, data in memory_data.items():
        if not data.empty:
            plt.figure(figsize=(12, 8))
            
            for sample, sample_data in data.groupby('sample'):
                plt.plot(sample_data['time_point'], sample_data['memory_kb'], label=sample)
            
            plt.title(f'Memory Usage Over Time - {impl}')
            plt.xlabel('Time Point')
            plt.ylabel('Memory Usage (KB)')
            plt.legend()
            plt.tight_layout()
            plt.savefig(os.path.join(report_dir, 'images', f'memory_usage_{impl}.png'))
            plt.close()
    
    log("Generated memory usage over time plots", verbose)

def plot_memory_metrics(memory_metrics, report_dir, verbose=False):
    """Plot memory metrics for each implementation."""
    os.makedirs(os.path.join(report_dir, 'images'), exist_ok=True)
    
    # Set the style
    sns.set(style="whitegrid")
    
    # Convert metrics to DataFrame
    metrics_df = pd.DataFrame(memory_metrics).reset_index().rename(columns={'index': 'implementation'})
    
    # Plot peak memory usage
    plt.figure(figsize=(10, 6))
    sns.barplot(x='implementation', y='peak_memory_kb', data=metrics_df)
    plt.title('Peak Memory Usage')
    plt.ylabel('Memory (KB)')
    plt.xlabel('Implementation')
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig(os.path.join(report_dir, 'images', 'peak_memory.png'))
    plt.close()
    
    # Plot average memory usage
    plt.figure(figsize=(10, 6))
    sns.barplot(x='implementation', y='avg_memory_kb', data=metrics_df)
    plt.title('Average Memory Usage')
    plt.ylabel('Memory (KB)')
    plt.xlabel('Implementation')
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig(os.path.join(report_dir, 'images', 'avg_memory.png'))
    plt.close()
    
    # Plot memory growth rate
    plt.figure(figsize=(10, 6))
    sns.barplot(x='implementation', y='growth_rate_kb', data=metrics_df)
    plt.title('Memory Growth Rate')
    plt.ylabel('Growth Rate (KB per time point)')
    plt.xlabel('Implementation')
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig(os.path.join(report_dir, 'images', 'growth_rate.png'))
    plt.close()
    
    # Plot memory stability
    plt.figure(figsize=(10, 6))
    sns.barplot(x='implementation', y='stability_ratio', data=metrics_df)
    plt.title('Memory Stability (lower is better)')
    plt.ylabel('Stability Ratio (std/mean)')
    plt.xlabel('Implementation')
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig(os.path.join(report_dir, 'images', 'stability.png'))
    plt.close()
    
    log("Generated memory metrics plots", verbose)

def plot_memory_efficiency(efficiency_df, report_dir, verbose=False):
    """Plot memory efficiency metrics."""
    os.makedirs(os.path.join(report_dir, 'images'), exist_ok=True)
    
    # Set the style
    sns.set(style="whitegrid")
    
    # Plot reads per MB
    plt.figure(figsize=(10, 6))
    sns.barplot(x='implementation', y='reads_per_mb', data=efficiency_df)
    plt.title('Memory Efficiency (Reads per MB)')
    plt.ylabel('Reads per MB')
    plt.xlabel('Implementation')
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig(os.path.join(report_dir, 'images', 'reads_per_mb.png'))
    plt.close()
    
    # Plot reads per MB per second
    plt.figure(figsize=(10, 6))
    sns.barplot(x='implementation', y='reads_per_mb_per_second', data=efficiency_df)
    plt.title('Memory-Time Efficiency (Reads per MB per Second)')
    plt.ylabel('Reads per MB per Second')
    plt.xlabel('Implementation')
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig(os.path.join(report_dir, 'images', 'reads_per_mb_per_second.png'))
    plt.close()
    
    log("Generated memory efficiency plots", verbose)

def generate_markdown_report(memory_metrics, memory_issues, efficiency_df, report_dir, verbose=False):
    """Generate a markdown report with memory analysis results."""
    os.makedirs(report_dir, exist_ok=True)
    
    report_path = os.path.join(report_dir, 'memory_analysis_report.md')
    
    with open(report_path, 'w') as f:
        f.write("# Memory Usage Analysis Report\n\n")
        
        f.write("## Memory Usage Over Time\n\n")
        f.write("![Memory Usage Over Time](images/memory_usage_over_time.png)\n\n")
        
        f.write("### Implementation-specific Memory Usage\n\n")
        for impl in memory_metrics.keys():
            f.write(f"#### {impl}\n\n")
            f.write(f"![Memory Usage - {impl}](images/memory_usage_{impl}.png)\n\n")
        
        f.write("## Memory Metrics\n\n")
        f.write("![Peak Memory Usage](images/peak_memory.png)\n\n")
        f.write("![Average Memory Usage](images/avg_memory.png)\n\n")
        f.write("![Memory Growth Rate](images/growth_rate.png)\n\n")
        f.write("![Memory Stability](images/stability.png)\n\n")
        
        f.write("### Detailed Metrics\n\n")
        f.write("| Implementation | Peak Memory (KB) | Avg Memory (KB) | Growth Rate (KB/tp) | Stability Ratio |\n")
        f.write("|----------------|-----------------|-----------------|---------------------|----------------|\n")
        
        metrics_df = pd.DataFrame(memory_metrics).reset_index().rename(columns={'index': 'implementation'})
        for _, row in metrics_df.iterrows():
            f.write(f"| {row['implementation']} | {row['peak_memory_kb']:.0f} | {row['avg_memory_kb']:.0f} | {row['growth_rate_kb']:.2f} | {row['stability_ratio']:.3f} |\n")
        
        f.write("\n## Memory Efficiency\n\n")
        f.write("![Reads per MB](images/reads_per_mb.png)\n\n")
        f.write("![Reads per MB per Second](images/reads_per_mb_per_second.png)\n\n")
        
        f.write("### Detailed Efficiency Metrics\n\n")
        f.write("| Implementation | Total Reads | Peak Memory (MB) | Reads per MB | Reads per MB per Second |\n")
        f.write("|----------------|-------------|------------------|--------------|-------------------------|\n")
        
        for _, row in efficiency_df.iterrows():
            peak_memory_mb = row['peak_memory_kb'] / 1024
            f.write(f"| {row['implementation']} | {row['total_reads']:.0f} | {peak_memory_mb:.2f} | {row['reads_per_mb']:.2f} | {row['reads_per_mb_per_second']:.2f} |\n")
        
        f.write("\n## Potential Memory Issues\n\n")
        
        if any(memory_issues.values()):
            for impl, issues in memory_issues.items():
                if issues:
                    f.write(f"### {impl}\n\n")
                    for issue in issues:
                        f.write(f"- {issue}\n")
                    f.write("\n")
        else:
            f.write("No significant memory issues detected.\n\n")
        
        f.write("## Conclusion\n\n")
        
        # Determine the most memory-efficient implementation
        if not efficiency_df.empty:
            most_efficient = efficiency_df.loc[efficiency_df['reads_per_mb'].idxmax()]
            lowest_peak = efficiency_df.loc[efficiency_df['peak_memory_kb'].idxmin()]
            most_stable = metrics_df.loc[metrics_df['stability_ratio'].idxmin()]
            
            f.write(f"Based on the analysis, the **{most_efficient['implementation']}** implementation is the most memory-efficient, ")
            f.write(f"processing {most_efficient['reads_per_mb']:.2f} reads per MB of memory.\n\n")
            
            f.write(f"The **{lowest_peak['implementation']}** implementation has the lowest peak memory usage ")
            f.write(f"at {lowest_peak['peak_memory_kb']/1024:.2f} MB.\n\n")
            
            f.write(f"The **{most_stable['implementation']}** implementation has the most stable memory usage pattern ")
            f.write(f"with a stability ratio of {most_stable['stability_ratio']:.3f}.\n\n")
            
            # Overall recommendation
            f.write("### Overall Recommendation\n\n")
            
            # Simple scoring system
            efficiency_df['efficiency_score'] = (
                efficiency_df['reads_per_mb'] / efficiency_df['reads_per_mb'].max() +
                (1 - efficiency_df['peak_memory_kb'] / efficiency_df['peak_memory_kb'].max())
            ) / 2
            
            best_overall = efficiency_df.loc[efficiency_df['efficiency_score'].idxmax()]
            
            f.write(f"For the best balance of memory efficiency and usage, the **{best_overall['implementation']}** implementation is recommended.\n")
        else:
            f.write("Insufficient data to make a recommendation.\n")
    
    log(f"Generated markdown report at {report_path}", verbose)
    return report_path

def generate_html_report(markdown_path, verbose=False):
    """Generate an HTML report from the markdown report."""
    try:
        import markdown
        from markdown.extensions.toc import TocExtension
        
        html_path = markdown_path.replace('.md', '.html')
        
        with open(markdown_path, 'r') as f:
            md_content = f.read()
        
        # Convert markdown to HTML
        html = markdown.markdown(
            md_content,
            extensions=[
                'markdown.extensions.extra',
                'markdown.extensions.codehilite',
                TocExtension(baselevel=1)
            ]
        )
        
        # Add CSS styling
        html_content = f"""
        <!DOCTYPE html>
        <html>
        <head>
            <meta charset="utf-8">
            <title>Memory Usage Analysis Report</title>
            <style>
                body {{
                    font-family: Arial, sans-serif;
                    line-height: 1.6;
                    max-width: 1200px;
                    margin: 0 auto;
                    padding: 20px;
                    color: #333;
                }}
                h1, h2, h3, h4 {{
                    color: #2c3e50;
                }}
                h1 {{
                    border-bottom: 2px solid #eee;
                    padding-bottom: 10px;
                }}
                h2 {{
                    margin-top: 30px;
                    border-bottom: 1px solid #eee;
                    padding-bottom: 5px;
                }}
                img {{
                    max-width: 100%;
                    height: auto;
                    border: 1px solid #ddd;
                    border-radius: 4px;
                    padding: 5px;
                    margin: 10px 0;
                }}
                table {{
                    border-collapse: collapse;
                    width: 100%;
                    margin: 20px 0;
                }}
                th, td {{
                    border: 1px solid #ddd;
                    padding: 8px;
                    text-align: left;
                }}
                th {{
                    background-color: #f2f2f2;
                }}
                tr:nth-child(even) {{
                    background-color: #f9f9f9;
                }}
            </style>
        </head>
        <body>
            {html}
        </body>
        </html>
        """
        
        with open(html_path, 'w') as f:
            f.write(html_content)
        
        log(f"Generated HTML report at {html_path}", verbose)
        return html_path
    except ImportError:
        log("Warning: Could not generate HTML report. Python markdown package not installed.", verbose)
        log("Install with: pip install markdown", verbose)
        return None

def main():
    args = parse_args()
    output_dir = args.output_dir
    report_dir = args.report_dir or os.path.join(output_dir, 'memory_analysis')
    verbose = args.verbose
    
    # Create report directory
    os.makedirs(report_dir, exist_ok=True)
    
    # Find memory usage files for each implementation
    implementations = ['r_implementation', 'go_default', 'go_r_algorithm']
    memory_files = {}
    
    for impl in implementations:
        memory_files[impl] = find_memory_files(output_dir, impl, verbose)
    
    # Parse memory usage files
    memory_data = {}
    
    for impl, files in memory_files.items():
        dfs = []
        for file in files:
            df = parse_memory_file(file, verbose)
            if df is not None:
                dfs.append(df)
        
        if dfs:
            memory_data[impl] = pd.concat(dfs, ignore_index=True)
        else:
            memory_data[impl] = pd.DataFrame()
    
    # Calculate memory metrics
    memory_metrics = {}
    memory_issues = {}
    
    for impl, data in memory_data.items():
        if not data.empty:
            metrics = calculate_memory_metrics(data, verbose)
            memory_metrics[impl] = metrics
            memory_issues[impl] = detect_memory_issues(metrics, verbose)
    
    # Read summary data
    summary_file = os.path.join(output_dir, 'summary.tsv')
    if os.path.exists(summary_file):
        summary_df = pd.read_csv(summary_file, sep='\t')
        
        # Calculate memory efficiency metrics
        efficiency_df = calculate_memory_efficiency(memory_metrics, summary_df, verbose)
    else:
        print(f"Warning: Summary file not found at {summary_file}")
        efficiency_df = pd.DataFrame()
    
    # Generate visualizations
    plot_memory_usage_over_time(memory_data, report_dir, verbose)
    plot_memory_metrics(memory_metrics, report_dir, verbose)
    
    if not efficiency_df.empty:
        plot_memory_efficiency(efficiency_df, report_dir, verbose)
    
    # Generate reports
    markdown_path = generate_markdown_report(memory_metrics, memory_issues, efficiency_df, report_dir, verbose)
    html_path = generate_html_report(markdown_path, verbose)
    
    print(f"Memory analysis completed. Reports generated in {report_dir}")
    print(f"Markdown report: {markdown_path}")
    if html_path:
        print(f"HTML report: {html_path}")

if __name__ == "__main__":
    main()