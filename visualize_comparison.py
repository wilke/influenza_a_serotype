#!/usr/bin/env python

import os
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
import sys
from pathlib import Path

def main():
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <output_directory>")
        sys.exit(1)
    
    output_dir = sys.argv[1]
    summary_file = os.path.join(output_dir, "summary.tsv")
    
    if not os.path.exists(summary_file):
        print(f"Error: Summary file not found at {summary_file}")
        sys.exit(1)
    
    # Read the summary data
    df = pd.read_csv(summary_file, sep='\t')
    
    # Create output directories for images
    images_dir = os.path.join(output_dir, "reports", "images")
    os.makedirs(images_dir, exist_ok=True)
    
    # Set the style
    sns.set(style="whitegrid")
    
    # 1. Runtime comparison
    plt.figure(figsize=(10, 6))
    runtime_plot = sns.barplot(x='implementation', y='runtime_seconds', data=df)
    plt.title('Runtime Comparison')
    plt.ylabel('Runtime (seconds)')
    plt.xlabel('Implementation')
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig(os.path.join(images_dir, 'runtime_comparison.png'))
    plt.close()
    
    # 2. Memory usage comparison
    plt.figure(figsize=(10, 6))
    memory_plot = sns.barplot(x='implementation', y='max_memory_kb', data=df)
    plt.title('Memory Usage Comparison')
    plt.ylabel('Max Memory (KB)')
    plt.xlabel('Implementation')
    plt.xticks(rotation=45)
    plt.tight_layout()
    plt.savefig(os.path.join(images_dir, 'memory_comparison.png'))
    plt.close()
    
    # 3. Read assignment comparison
    df['total_reads'] = df['H1N1_reads'] + df['H3N2_reads'] + df['ambiguous_reads']
    
    # Calculate percentages
    df['H1N1_pct'] = df['H1N1_reads'] / df['total_reads'] * 100
    df['H3N2_pct'] = df['H3N2_reads'] / df['total_reads'] * 100
    df['ambiguous_pct'] = df['ambiguous_reads'] / df['total_reads'] * 100
    
    # Prepare data for stacked bar chart
    read_data = []
    for impl in df['implementation'].unique():
        impl_data = df[df['implementation'] == impl]
        read_data.append({
            'implementation': impl,
            'H1N1': impl_data['H1N1_reads'].sum(),
            'H3N2': impl_data['H3N2_reads'].sum(),
            'Ambiguous': impl_data['ambiguous_reads'].sum()
        })
    
    read_df = pd.DataFrame(read_data)
    read_df = read_df.set_index('implementation')
    
    # Create stacked bar chart
    plt.figure(figsize=(10, 6))
    read_df.plot(kind='bar', stacked=True, colormap='viridis')
    plt.title('Read Assignment Comparison')
    plt.ylabel('Number of Reads')
    plt.xlabel('Implementation')
    plt.xticks(rotation=45)
    plt.legend(title='Read Type')
    plt.tight_layout()
    plt.savefig(os.path.join(images_dir, 'read_assignment_comparison.png'))
    plt.close()
    
    # 4. Read assignment percentage comparison
    pct_data = []
    for impl in df['implementation'].unique():
        impl_data = df[df['implementation'] == impl]
        total_reads = impl_data['total_reads'].sum()
        if total_reads > 0:
            pct_data.append({
                'implementation': impl,
                'H1N1': impl_data['H1N1_reads'].sum() / total_reads * 100,
                'H3N2': impl_data['H3N2_reads'].sum() / total_reads * 100,
                'Ambiguous': impl_data['ambiguous_reads'].sum() / total_reads * 100
            })
    
    pct_df = pd.DataFrame(pct_data)
    pct_df = pct_df.set_index('implementation')
    
    # Create stacked bar chart for percentages
    plt.figure(figsize=(10, 6))
    pct_df.plot(kind='bar', stacked=True, colormap='viridis')
    plt.title('Read Assignment Percentage Comparison')
    plt.ylabel('Percentage of Reads')
    plt.xlabel('Implementation')
    plt.xticks(rotation=45)
    plt.legend(title='Read Type')
    plt.tight_layout()
    plt.savefig(os.path.join(images_dir, 'read_assignment_percentage.png'))
    plt.close()
    
    # 5. File-specific runtime comparison
    plt.figure(figsize=(12, 8))
    file_runtime_plot = sns.barplot(x='file', y='runtime_seconds', hue='implementation', data=df)
    plt.title('Runtime Comparison by File')
    plt.ylabel('Runtime (seconds)')
    plt.xlabel('File')
    plt.xticks(rotation=90)
    plt.legend(title='Implementation')
    plt.tight_layout()
    plt.savefig(os.path.join(images_dir, 'file_runtime_comparison.png'))
    plt.close()
    
    # 6. File-specific memory comparison
    plt.figure(figsize=(12, 8))
    file_memory_plot = sns.barplot(x='file', y='max_memory_kb', hue='implementation', data=df)
    plt.title('Memory Usage Comparison by File')
    plt.ylabel('Max Memory (KB)')
    plt.xlabel('File')
    plt.xticks(rotation=90)
    plt.legend(title='Implementation')
    plt.tight_layout()
    plt.savefig(os.path.join(images_dir, 'file_memory_comparison.png'))
    plt.close()
    
    # Generate a markdown report
    # Get read assignments by sample
    read_assignments = get_read_assignments_by_sample(output_dir)
    
    # Analyze input files
    input_stats = analyze_input_files(output_dir, df)
    
    # Generate the markdown report
    generate_markdown_report(output_dir, df, read_assignments, input_stats)
    
    # Generate an HTML report
    generate_html_report(output_dir)
    
    print(f"Visualization completed. Images saved to {images_dir}")
    
def get_read_assignments_by_sample(output_dir):
    """
    Find and parse all read assignment files for each implementation and sample.
    Dynamically extracts serotypes from file names following the pattern ${sample_name}_${serotype}.txt
    and ${sample_name}_ambiguous.txt.
    Returns a dictionary with counts of reads assigned to each category per sample and implementation.
    """
    # Find all implementation directories
    implementations = []
    for item in os.listdir(output_dir):
        if os.path.isdir(os.path.join(output_dir, item)) and not item.startswith('.'):
            implementations.append(item)
    
    assignments = {}
    all_categories = set()  # Track all categories found across all samples
    
    for impl in implementations:
        impl_dir = os.path.join(output_dir, impl)
        if not os.path.exists(impl_dir):
            continue
            
        # Find all output files
        for root, _, files in os.walk(impl_dir):
            for file in files:
                # Check if file matches the pattern ${sample_name}_*.txt
                if file.endswith(".txt") and "_" in file:
                    parts = file.rsplit("_", 1)
                    if len(parts) == 2:
                        sample_name = parts[0]
                        category_with_ext = parts[1]
                        
                        # Skip performance samples
                        if "_performance" in sample_name:
                            continue
                        
                        # Extract category (remove .txt extension)
                        if category_with_ext.endswith(".txt"):
                            category = category_with_ext[:-4]  # Remove ".txt"
                            
                            # Skip summary and usage categories
                            if category.lower() in ["summary", "usage"]:
                                continue
                            
                            # Count lines in the file to get read count
                            file_path = os.path.join(root, file)
                            try:
                                with open(file_path, 'r') as f:
                                    read_count = sum(1 for _ in f)
                            except Exception as e:
                                print(f"Error reading {file_path}: {e}")
                                read_count = 0
                            
                            # Add this category to our set of all categories
                            all_categories.add(category)
                            
                            # Initialize nested dictionaries if they don't exist
                            if sample_name not in assignments:
                                assignments[sample_name] = {}
                            if impl not in assignments[sample_name]:
                                assignments[sample_name][impl] = {"Total": 0}
                            if category not in assignments[sample_name][impl]:
                                assignments[sample_name][impl][category] = 0
                            
                            # Update read count
                            assignments[sample_name][impl][category] = read_count
                            assignments[sample_name][impl]["Total"] += read_count
    
    return assignments

def analyze_input_files(output_dir, df):
    """
    Analyze input PAF files to extract statistics about unique qnames and hits per qname.
    Returns a dictionary with input file statistics.
    """
    input_stats = {}
    
    # Get unique file names from the summary dataframe
    unique_files = df['file'].unique()
    
    for file_name in unique_files:
        # Look for the input file in the test_data directory or other common locations
        potential_paths = [
            os.path.join(output_dir, "..", "test_data", file_name),
            os.path.join(output_dir, "..", file_name),
            os.path.join("test_data", file_name)
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
        qnames = set()
        qname_hits = {}
        
        try:
            with open(file_path, 'r') as f:
                for line in f:
                    parts = line.strip().split('\t')
                    if len(parts) >= 1:
                        qname = parts[0]
                        qnames.add(qname)
                        qname_hits[qname] = qname_hits.get(qname, 0) + 1
        except Exception as e:
            print(f"Error analyzing input file {file_path}: {e}")
            continue
        
        # Calculate statistics
        unique_qnames = len(qnames)
        total_hits = sum(qname_hits.values())
        avg_hits_per_qname = total_hits / unique_qnames if unique_qnames > 0 else 0
        max_hits = max(qname_hits.values()) if qname_hits else 0
        min_hits = min(qname_hits.values()) if qname_hits else 0
        
        # Store statistics
        input_stats[file_name] = {
            "unique_qnames": unique_qnames,
            "total_hits": total_hits,
            "avg_hits_per_qname": avg_hits_per_qname,
            "max_hits": max_hits,
            "min_hits": min_hits
        }
    
    return input_stats

def generate_markdown_report(output_dir, df, read_assignments, input_stats):
    """Generate a markdown report with the comparison results."""
    images_dir = os.path.join("images")
    report_path = os.path.join(output_dir, "reports", "comparison_report.md")
    
    # Read the final summary
    with open(os.path.join(output_dir, "final_summary.txt"), 'r') as f:
        final_summary = f.read()
    
    # Create the markdown report
    with open(report_path, 'w') as f:
        f.write("# PAF Processor Implementation Comparison\n\n")
        
        f.write("## Summary\n\n")
        f.write("```\n")
        f.write(final_summary)
        f.write("```\n\n")
        
        f.write("## Runtime Comparison\n\n")
        f.write(f"![Runtime Comparison]({images_dir}/runtime_comparison.png)\n\n")
        
        f.write("## Memory Usage Comparison\n\n")
        f.write(f"![Memory Usage Comparison]({images_dir}/memory_comparison.png)\n\n")
        
        f.write("## Read Assignment Comparison\n\n")
        f.write(f"![Read Assignment Comparison]({images_dir}/read_assignment_comparison.png)\n\n")
        f.write(f"![Read Assignment Percentage]({images_dir}/read_assignment_percentage.png)\n\n")
        
        f.write("## File-specific Comparisons\n\n")
        f.write(f"![File Runtime Comparison]({images_dir}/file_runtime_comparison.png)\n\n")
        f.write(f"![File Memory Comparison]({images_dir}/file_memory_comparison.png)\n\n")
        
        f.write("## Output Differences\n\n")
        
        # Add information about output differences
        comparison_dir = os.path.join(output_dir, "comparison", "outputs")
        if os.path.exists(comparison_dir):
            diff_files = [f for f in os.listdir(comparison_dir) if f.endswith('.diff')]
            
            if diff_files:
                f.write("The following output differences were found:\n\n")
                
                for diff_file in sorted(diff_files):
                    diff_path = os.path.join(comparison_dir, diff_file)
                    with open(diff_path, 'r') as diff_f:
                        diff_content = diff_f.read().strip()
                    
                    if diff_content:
                        f.write(f"### {diff_file}\n\n")
                        f.write("```diff\n")
                        f.write(diff_content[:1000])  # Limit to first 1000 chars
                        if len(diff_content) > 1000:
                            f.write("\n... (truncated)")
                        f.write("\n```\n\n")
                    else:
                        f.write(f"### {diff_file}\n\n")
                        f.write("No differences found.\n\n")
            else:
                f.write("No output differences were found.\n\n")
        
        f.write("## Read Assignments Per Sample\n\n")
        
        if read_assignments:
            # Get all categories across all samples and implementations
            all_categories = set()
            for sample_data in read_assignments.values():
                for impl_data in sample_data.values():
                    for category in impl_data.keys():
                        if category != "Total" and category.lower() not in ["summary", "usage"]:
                            all_categories.add(category)
            
            # Sort categories for consistent display
            sorted_categories = sorted(all_categories)
            
            # Create a table header
            f.write("| Sample | Implementation | ")
            for category in sorted_categories:
                f.write(f"{category} Reads | ")
            f.write("Total Reads |\n")
            
            # Create the header separator line
            f.write("|--------|---------------|")
            for _ in sorted_categories:
                f.write("------------|")
            f.write("-------------|\n")
            
            # Sort samples for consistent display
            for sample_name in sorted(read_assignments.keys()):
                # Skip performance samples
                if "_performance" in sample_name:
                    continue
                    
                sample_data = read_assignments[sample_name]
                
                # Sort implementations for consistent display
                for impl in sorted(sample_data.keys()):
                    impl_data = sample_data[impl]
                    
                    # Write a row for each sample and implementation
                    f.write(f"| {sample_name} | {impl} | ")
                    
                    # Add data for each category
                    for category in sorted_categories:
                        read_count = impl_data.get(category, 0)  # Default to 0 if category not present
                        f.write(f"{read_count} | ")
                    
                    # Add total
                    f.write(f"{impl_data['Total']} |\n")
                
                # Add a separator between samples for better readability
                if len([s for s in read_assignments.keys() if "_performance" not in s]) > 1:
                    f.write("|--------|---------------|")
                    for _ in sorted_categories:
                        f.write("------------|")
                    f.write("-------------|\n")
            
            # Add a section for comparison between implementations
            f.write("\n### Read Assignment Differences\n\n")
            
            # For each sample, compare read assignments between implementations
            for sample_name in sorted(read_assignments.keys()):
                # Skip performance samples
                if "_performance" in sample_name:
                    continue
                    
                sample_data = read_assignments[sample_name]
                
                # Only compare if we have more than one implementation for this sample
                if len(sample_data) > 1:
                    f.write(f"#### {sample_name}\n\n")
                    
                    # Create a table for percentage differences
                    f.write("| Category | ")
                    for impl in sorted(sample_data.keys()):
                        f.write(f"{impl} | ")
                    f.write("\n")
                    
                    f.write("|----------|")
                    for _ in range(len(sample_data)):
                        f.write("------------|")
                    f.write("\n")
                    
                    # Add rows for each category
                    for category in sorted_categories:
                        f.write(f"| {category} | ")
                        for impl in sorted(sample_data.keys()):
                            total = sample_data[impl]["Total"]
                            if total > 0 and category in sample_data[impl]:
                                percentage = (sample_data[impl][category] / total) * 100
                                f.write(f"{percentage:.2f}% | ")
                            else:
                                f.write("0.00% | ")
                        f.write("\n")
                    
                    f.write("\n")
        else:
            f.write("No read assignment data available.\n\n")
        
        # Add Serotype Statistics section
        f.write("## Serotype Statistics\n\n")
        
        if read_assignments:
            # Calculate total reads per serotype across all samples and implementations
            serotype_totals = {}
            total_reads_all = 0
            
            for sample_data in read_assignments.values():
                for impl_data in sample_data.values():
                    for category, count in impl_data.items():
                        if category != "Total" and category.lower() not in ["summary", "usage"]:
                            serotype_totals[category] = serotype_totals.get(category, 0) + count
                            total_reads_all += count
            
            # Sort serotypes by total reads (descending)
            sorted_serotypes = sorted(serotype_totals.items(), key=lambda x: x[1], reverse=True)
            
            # Create a table for serotype statistics
            f.write("### Total Reads Per Serotype\n\n")
            f.write("| Serotype | Total Reads | Percentage |\n")
            f.write("|----------|-------------|------------|\n")
            
            for serotype, count in sorted_serotypes:
                percentage = (count / total_reads_all) * 100 if total_reads_all > 0 else 0
                f.write(f"| {serotype} | {count} | {percentage:.2f}% |\n")
            
            # Add total row
            f.write(f"| **Total** | **{total_reads_all}** | **100.00%** |\n\n")
        else:
            f.write("No serotype statistics available.\n\n")
        
        # Add Runtime and Memory Usage section
        f.write("## Runtime and Memory Usage\n\n")
        
        # Calculate runtime statistics per implementation
        runtime_stats = df.groupby('implementation')['runtime_seconds'].agg(['min', 'max', 'mean'])
        memory_stats = df.groupby('implementation')['max_memory_kb'].agg(['min', 'max', 'mean'])
        
        # Create runtime statistics table
        f.write("### Runtime Statistics (seconds)\n\n")
        f.write("| Implementation | Minimum | Maximum | Average |\n")
        f.write("|----------------|---------|---------|--------|\n")
        
        for impl, stats in runtime_stats.iterrows():
            f.write(f"| {impl} | {stats['min']:.2f} | {stats['max']:.2f} | {stats['mean']:.2f} |\n")
        
        f.write("\n")
        
        # Create memory statistics table
        f.write("### Memory Usage Statistics (KB)\n\n")
        f.write("| Implementation | Minimum | Maximum | Average |\n")
        f.write("|----------------|---------|---------|--------|\n")
        
        for impl, stats in memory_stats.iterrows():
            f.write(f"| {impl} | {stats['min']:.2f} | {stats['max']:.2f} | {stats['mean']:.2f} |\n")
        
        f.write("\n")
        
        # Add performance comparison table
        f.write("### Performance Comparison\n\n")
        
        # Calculate relative performance (normalized to the slowest implementation)
        slowest_impl = runtime_stats['mean'].idxmax()
        highest_memory_impl = memory_stats['mean'].idxmax()
        
        max_runtime = runtime_stats.loc[slowest_impl, 'mean']
        max_memory = memory_stats.loc[highest_memory_impl, 'mean']
        
        f.write("| Implementation | Relative Speed | Relative Memory Efficiency |\n")
        f.write("|----------------|---------------|---------------------------|\n")
        
        for impl in runtime_stats.index:
            rel_speed = max_runtime / runtime_stats.loc[impl, 'mean']
            rel_memory = max_memory / memory_stats.loc[impl, 'mean']
            f.write(f"| {impl} | {rel_speed:.2f}x | {rel_memory:.2f}x |\n")
        
        f.write("\n")
        
        # Add Input File Statistics section
        f.write("## Input File Statistics\n\n")
        
        if input_stats:
            f.write("| File | Unique QNames | Total Hits | Avg Hits/QName | Max Hits | Min Hits |\n")
            f.write("|------|--------------|------------|----------------|----------|----------|\n")
            
            for file_name, stats in sorted(input_stats.items()):
                f.write(f"| {file_name} | {stats['unique_qnames']} | {stats['total_hits']} | ")
                f.write(f"{stats['avg_hits_per_qname']:.2f} | {stats['max_hits']} | {stats['min_hits']} |\n")
        else:
            f.write("No input file statistics available.\n\n")
        
        f.write("## Conclusion\n\n")
        f.write("This report compares three implementations of the PAF processor for influenza A serotyping:\n\n")
        f.write("1. **R Implementation**: The original implementation in R, which serves as the reference for output validation\n")
        f.write("2. **Go Default**: The Go implementation with its optimized algorithm designed for performance\n")
        f.write("3. **Go R-Algorithm**: The Go implementation using the R-compatible algorithm to ensure consistent results\n\n")
        
        f.write("The comparison evaluates these implementations across three key dimensions:\n\n")
        f.write("- **Performance**: Runtime efficiency across different file sizes and types\n")
        f.write("- **Resource Usage**: Memory consumption during processing\n")
        f.write("- **Output Consistency**: Differences in read assignments between implementations\n\n")
        
        # Add detailed conclusions based on the data
        runtime_by_impl = df.groupby('implementation')['runtime_seconds'].mean()
        memory_by_impl = df.groupby('implementation')['max_memory_kb'].max()
        
        fastest_impl = runtime_by_impl.idxmin()
        slowest_impl = runtime_by_impl.idxmax()
        lowest_memory_impl = memory_by_impl.idxmin()
        highest_memory_impl = memory_by_impl.idxmax()
        
        # Calculate performance differences
        speed_improvement = (runtime_by_impl[slowest_impl] / runtime_by_impl[fastest_impl])
        memory_improvement = (memory_by_impl[highest_memory_impl] / memory_by_impl[lowest_memory_impl])
        
        f.write("### Key Findings\n\n")
        
        f.write(f"**Performance**: The **{fastest_impl}** implementation demonstrates superior runtime performance, ")
        f.write(f"approximately {speed_improvement:.1f}x faster than the **{slowest_impl}** implementation. ")
        f.write(f"This makes it particularly suitable for processing large datasets or when time efficiency is critical.\n\n")
        
        f.write(f"**Resource Efficiency**: The **{lowest_memory_impl}** implementation shows the most efficient memory usage, ")
        f.write(f"using approximately {1/memory_improvement:.1f}x less memory than the **{highest_memory_impl}** implementation. ")
        f.write(f"This makes it advantageous for environments with limited resources or when processing very large files.\n\n")
        
        # Add information about output consistency
        f.write("**Output Consistency**: ")
        
        # Check if comparison directory exists to determine if there are differences
        comparison_dir = os.path.join(output_dir, "comparison", "outputs")
        if os.path.exists(comparison_dir):
            diff_files = [f for f in os.listdir(comparison_dir) if f.endswith('.diff')]
            if diff_files and any(os.path.getsize(os.path.join(comparison_dir, f)) > 0 for f in diff_files):
                f.write("There are notable differences in read assignments between implementations, as detailed in the diff files above. ")
                f.write("These differences primarily stem from algorithmic variations in how reads are assigned to serotypes. ")
                f.write("The Go R-Algorithm implementation aims to match the R implementation's output as closely as possible, ")
                f.write("while the Go Default implementation prioritizes performance optimizations.\n\n")
            else:
                f.write("The implementations show high consistency in their outputs, with minimal differences in read assignments. ")
                f.write("This suggests that the algorithmic differences between implementations have limited impact on the final results.\n\n")
        else:
            f.write("Output consistency analysis was not performed or no comparison data is available.\n\n")
        
        f.write("### Recommendations\n\n")
        
        f.write("Based on this comprehensive analysis, we recommend:\n\n")
        
        f.write(f"- **For Maximum Performance**: Use the **{fastest_impl}** implementation when processing speed is the primary concern, ")
        f.write("particularly for large-scale analyses or time-sensitive applications.\n\n")
        
        f.write(f"- **For Resource-Constrained Environments**: The **{lowest_memory_impl}** implementation is ideal when working with ")
        f.write("limited computational resources or when processing extremely large datasets that might cause memory issues.\n\n")
        
        f.write("- **For Output Consistency with Legacy Systems**: If maintaining exact compatibility with the original R implementation ")
        f.write("is critical, the **Go R-Algorithm** implementation offers the best balance of improved performance while preserving ")
        f.write("output consistency.\n\n")
        
        f.write("For detailed analysis of specific differences in read assignments, refer to the Output Differences section above.\n")
    
    print(f"Markdown report generated at {report_path}")

def generate_html_report(output_dir):
    """Generate an HTML report from the markdown report."""
    try:
        import markdown
        from markdown.extensions.toc import TocExtension
        from markdown.extensions.tables import TableExtension
        
        md_path = os.path.join(output_dir, "reports", "comparison_report.md")
        html_path = os.path.join(output_dir, "reports", "comparison_report.html")
        
        if not os.path.exists(md_path):
            print(f"Error: Markdown report not found at {md_path}")
            return
        
        with open(md_path, 'r') as f:
            md_content = f.read()
        
        # Convert markdown to HTML
        html = markdown.markdown(
            md_content,
            extensions=[
                'markdown.extensions.extra',
                'markdown.extensions.codehilite',
                TocExtension(baselevel=1),
                TableExtension()
            ]
        )
        
        # Add CSS styling
        html_content = f"""
        <!DOCTYPE html>
        <html>
        <head>
            <meta charset="utf-8">
            <title>PAF Processor Implementation Comparison</title>
            <style>
                body {{
                    font-family: Arial, sans-serif;
                    line-height: 1.6;
                    max-width: 1200px;
                    margin: 0 auto;
                    padding: 20px;
                    color: #333;
                }}
                h1, h2, h3 {{
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
                    font-size: 14px;
                }}
                th, td {{
                    border: 1px solid #ddd;
                    padding: 8px;
                    text-align: left;
                }}
                th {{
                    background-color: #f2f2f2;
                    font-weight: bold;
                }}
                tr:nth-child(even) {{
                    background-color: #f9f9f9;
                }}
                tr:hover {{
                    background-color: #f1f1f1;
                }}
                pre {{
                    background-color: #f8f8f8;
                    border: 1px solid #ddd;
                    border-radius: 3px;
                    padding: 10px;
                    overflow: auto;
                }}
                code {{
                    background-color: #f8f8f8;
                    padding: 2px 4px;
                    border-radius: 3px;
                }}
                .diff {{
                    font-family: monospace;
                }}
                .diff-added {{
                    background-color: #e6ffed;
                    color: #22863a;
                }}
                .diff-removed {{
                    background-color: #ffeef0;
                    color: #cb2431;
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
        
        print(f"HTML report generated at {html_path}")
    except ImportError:
        print("Warning: Could not generate HTML report. Python markdown package not installed.")
        print("Install with: pip install markdown")

if __name__ == "__main__":
    main()