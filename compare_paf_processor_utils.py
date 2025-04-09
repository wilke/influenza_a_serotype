#!/usr/bin/env pythom

import os
import numpy as np
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns
from sklearn.metrics import confusion_matrix, cohen_kappa_score
import itertools
from collections import defaultdict

def calculate_agreement_metrics(assignment_df, implementations, verbose=False):
    """Calculate agreement metrics between implementations."""
    metrics = {}
    
    # Calculate metrics for each pair of implementations
    for impl1, impl2 in itertools.combinations(implementations, 2):
        key = f"{impl1}_vs_{impl2}"
        metrics[key] = {}
        
        # Filter out rows where either implementation has 'unknown' assignment
        valid_df = assignment_df[~((assignment_df[impl1] == 'unknown') | (assignment_df[impl2] == 'unknown'))]
        
        if len(valid_df) > 0:
            # Calculate percentage agreement
            agreement = (valid_df[impl1] == valid_df[impl2]).mean() * 100
            metrics[key]['percentage_agreement'] = agreement
            
            # Calculate Cohen's Kappa
            kappa = cohen_kappa_score(valid_df[impl1], valid_df[impl2])
            metrics[key]['cohens_kappa'] = kappa
            
            # Calculate confusion matrix
            labels = sorted(set(valid_df[impl1].unique()) | set(valid_df[impl2].unique()))
            cm = confusion_matrix(valid_df[impl1], valid_df[impl2], labels=labels)
            metrics[key]['confusion_matrix'] = cm.tolist()
            metrics[key]['confusion_matrix_labels'] = labels
            
            # Calculate per-serotype agreement
            serotype_metrics = {}
            for serotype in labels:
                serotype_df = valid_df[valid_df[impl1] == serotype]
                if len(serotype_df) > 0:
                    serotype_agreement = (serotype_df[impl1] == serotype_df[impl2]).mean() * 100
                    serotype_metrics[serotype] = {
                        'count': len(serotype_df),
                        'agreement_percentage': serotype_agreement
                    }
            
            metrics[key]['serotype_metrics'] = serotype_metrics
            
            if verbose:
                print(f"Calculated agreement metrics for {impl1} vs {impl2}: {agreement:.2f}% agreement, kappa={kappa:.3f}")
        else:
            if verbose:
                print(f"No valid data for {impl1} vs {impl2}")
    
    return metrics

def plot_confusion_matrices(metrics, report_dir, verbose=False):
    """Plot confusion matrices for each pair of implementations."""
    os.makedirs(os.path.join(report_dir, 'images'), exist_ok=True)
    
    # Set the style
    sns.set(style="white")
    
    for key, data in metrics.items():
        if 'confusion_matrix' in data and 'confusion_matrix_labels' in data:
            cm = np.array(data['confusion_matrix'])
            labels = data['confusion_matrix_labels']
            
            # Create a normalized confusion matrix
            cm_normalized = cm.astype('float') / cm.sum(axis=1)[:, np.newaxis]
            
            # Plot the confusion matrix
            plt.figure(figsize=(10, 8))
            plt.imshow(cm_normalized, interpolation='nearest', cmap=plt.cm.Blues)
            plt.title(f'Normalized Confusion Matrix - {key}')
            plt.colorbar()
            tick_marks = np.arange(len(labels))
            plt.xticks(tick_marks, labels, rotation=45)
            plt.yticks(tick_marks, labels)
            
            # Add text annotations
            fmt = '.2f'
            thresh = cm_normalized.max() / 2.
            for i, j in itertools.product(range(cm.shape[0]), range(cm.shape[1])):
                plt.text(j, i, format(cm_normalized[i, j], fmt),
                        horizontalalignment="center",
                        color="white" if cm_normalized[i, j] > thresh else "black")
            
            plt.tight_layout()
            plt.ylabel('Implementation 1 Assignment')
            plt.xlabel('Implementation 2 Assignment')
            plt.savefig(os.path.join(report_dir, 'images', f'confusion_matrix_{key}.png'))
            plt.close()
            
            if verbose:
                print(f"Generated confusion matrix plot for {key}")

def plot_agreement_metrics(metrics, report_dir, verbose=False):
    """Plot agreement metrics for each pair of implementations."""
    os.makedirs(os.path.join(report_dir, 'images'), exist_ok=True)
    
    # Set the style
    sns.set(style="whitegrid")
    
    # Extract percentage agreement and Cohen's Kappa for each pair
    pairs = []
    percentage_agreements = []
    kappas = []
    
    for key, data in metrics.items():
        if 'percentage_agreement' in data and 'cohens_kappa' in data:
            pairs.append(key)
            percentage_agreements.append(data['percentage_agreement'])
            kappas.append(data['cohens_kappa'])
    
    # Plot percentage agreement
    plt.figure(figsize=(10, 6))
    bars = plt.bar(pairs, percentage_agreements)
    plt.title('Percentage Agreement Between Implementations')
    plt.ylabel('Agreement (%)')
    plt.xlabel('Implementation Pair')
    plt.xticks(rotation=45)
    
    # Add value labels on top of bars
    for bar in bars:
        height = bar.get_height()
        plt.text(bar.get_x() + bar.get_width()/2., height + 1,
                f'{height:.1f}%', ha='center', va='bottom')
    
    plt.tight_layout()
    plt.savefig(os.path.join(report_dir, 'images', 'percentage_agreement.png'))
    plt.close()
    
    # Plot Cohen's Kappa
    plt.figure(figsize=(10, 6))
    bars = plt.bar(pairs, kappas)
    plt.title('Cohen\'s Kappa Between Implementations')
    plt.ylabel('Kappa')
    plt.xlabel('Implementation Pair')
    plt.xticks(rotation=45)
    
    # Add value labels on top of bars
    for bar in bars:
        height = bar.get_height()
        plt.text(bar.get_x() + bar.get_width()/2., height + 0.02,
                f'{height:.3f}', ha='center', va='bottom')
    
    plt.tight_layout()
    plt.savefig(os.path.join(report_dir, 'images', 'cohens_kappa.png'))
    plt.close()
    
    if verbose:
        print("Generated agreement metrics plots")

def plot_serotype_distribution(assignment_df, implementations, report_dir, verbose=False):
    """Plot serotype distribution for each implementation."""
    os.makedirs(os.path.join(report_dir, 'images'), exist_ok=True)
    
    # Set the style
    sns.set(style="whitegrid")
    
    # Get all serotypes
    all_serotypes = set()
    for impl in implementations:
        all_serotypes.update(assignment_df[impl].unique())
    
    if 'unknown' in all_serotypes:
        all_serotypes.remove('unknown')
    
    # Count serotypes for each implementation
    serotype_counts = {}
    for impl in implementations:
        counts = assignment_df[impl].value_counts()
        serotype_counts[impl] = {serotype: counts.get(serotype, 0) for serotype in all_serotypes}
    
    # Create DataFrame for plotting
    plot_data = []
    for impl in implementations:
        for serotype, count in serotype_counts[impl].items():
            plot_data.append({
                'implementation': impl,
                'serotype': serotype,
                'count': count
            })
    
    plot_df = pd.DataFrame(plot_data)
    
    # Plot serotype distribution
    plt.figure(figsize=(12, 8))
    ax = sns.barplot(x='serotype', y='count', hue='implementation', data=plot_df)
    plt.title('Serotype Distribution by Implementation')
    plt.ylabel('Number of Reads')
    plt.xlabel('Serotype')
    plt.xticks(rotation=45)
    plt.legend(title='Implementation')
    
    # Add value labels on top of bars
    for container in ax.containers:
        ax.bar_label(container, fmt='%d')
    
    plt.tight_layout()
    plt.savefig(os.path.join(report_dir, 'images', 'serotype_distribution.png'))
    plt.close()
    
    # Plot serotype distribution as percentage
    plt.figure(figsize=(12, 8))
    
    # Calculate percentages
    percentage_data = []
    for impl in implementations:
        total = sum(serotype_counts[impl].values())
        if total > 0:
            for serotype, count in serotype_counts[impl].items():
                percentage_data.append({
                    'implementation': impl,
                    'serotype': serotype,
                    'percentage': (count / total) * 100
                })
    
    percentage_df = pd.DataFrame(percentage_data)
    
    ax = sns.barplot(x='serotype', y='percentage', hue='implementation', data=percentage_df)
    plt.title('Serotype Distribution by Implementation (Percentage)')
    plt.ylabel('Percentage of Reads')
    plt.xlabel('Serotype')
    plt.xticks(rotation=45)
    plt.legend(title='Implementation')
    
    # Add value labels on top of bars
    for container in ax.containers:
        ax.bar_label(container, fmt='%.1f%%')
    
    plt.tight_layout()
    plt.savefig(os.path.join(report_dir, 'images', 'serotype_distribution_percentage.png'))
    plt.close()
    
    if verbose:
        print("Generated serotype distribution plots")

def plot_read_assignment_changes(assignment_df, implementations, report_dir, verbose=False):
    """Plot read assignment changes between implementations."""
    os.makedirs(os.path.join(report_dir, 'images'), exist_ok=True)
    
    # Set the style
    sns.set(style="whitegrid")
    
    # Calculate changes for each pair of implementations
    for impl1, impl2 in itertools.combinations(implementations, 2):
        key = f"{impl1}_vs_{impl2}"
        
        # Filter out rows where either implementation has 'unknown' assignment
        valid_df = assignment_df[~((assignment_df[impl1] == 'unknown') | (assignment_df[impl2] == 'unknown'))]
        
        if len(valid_df) > 0:
            # Create a DataFrame with changes
            changes_df = valid_df[valid_df[impl1] != valid_df[impl2]]
            
            if len(changes_df) > 0:
                # Count changes by serotype
                change_counts = defaultdict(int)
                for _, row in changes_df.iterrows():
                    change_key = f"{row[impl1]}_to_{row[impl2]}"
                    change_counts[change_key] += 1
                
                # Create DataFrame for plotting
                plot_data = []
                for change, count in change_counts.items():
                    from_serotype, to_serotype = change.split('_to_')
                    plot_data.append({
                        'from_serotype': from_serotype,
                        'to_serotype': to_serotype,
                        'count': count
                    })
                
                plot_df = pd.DataFrame(plot_data)
                
                # Plot changes as a heatmap
                plt.figure(figsize=(12, 10))
                
                # Create a pivot table
                pivot_df = plot_df.pivot_table(
                    values='count', 
                    index='from_serotype', 
                    columns='to_serotype', 
                    fill_value=0
                )
                
                # Plot heatmap
                sns.heatmap(pivot_df, annot=True, fmt='d', cmap='YlGnBu')
                plt.title(f'Read Assignment Changes: {impl1} → {impl2}')
                plt.ylabel(f'{impl1} Assignment')
                plt.xlabel(f'{impl2} Assignment')
                plt.tight_layout()
                plt.savefig(os.path.join(report_dir, 'images', f'assignment_changes_{key}.png'))
                plt.close()
                
                if verbose:
                    print(f"Generated read assignment changes plot for {key}")
            else:
                if verbose:
                    print(f"No assignment changes between {impl1} and {impl2}")

def generate_markdown_report(assignment_df, metrics, implementations, report_dir, verbose=False):
    """Generate a markdown report with output comparison results."""
    os.makedirs(report_dir, exist_ok=True)
    
    report_path = os.path.join(report_dir, 'output_comparison_report.md')
    
    with open(report_path, 'w') as f:
        f.write("# PAF Processor Output Comparison Report\n\n")
        
        f.write("## Overview\n\n")
        f.write("This report compares the outputs of different PAF processor implementations:\n\n")
        for impl in implementations:
            f.write(f"- **{impl}**\n")
        f.write("\n")
        
        # Summary statistics
        f.write("## Summary Statistics\n\n")
        f.write("### Read Counts by Implementation\n\n")
        f.write("| Implementation | Total Reads | H1N1 | H3N2 | Ambiguous | Unknown |\n")
        f.write("|----------------|-------------|------|------|-----------|--------|\n")
        
        for impl in implementations:
            total_reads = len(assignment_df)
            h1n1_count = len(assignment_df[assignment_df[impl] == 'H1N1'])
            h3n2_count = len(assignment_df[assignment_df[impl] == 'H3N2'])
            ambiguous_count = len(assignment_df[assignment_df[impl] == 'ambiguous'])
            unknown_count = len(assignment_df[assignment_df[impl] == 'unknown'])
            
            f.write(f"| {impl} | {total_reads} | {h1n1_count} | {h3n2_count} | {ambiguous_count} | {unknown_count} |\n")
        
        f.write("\n### Serotype Distribution\n\n")
        f.write("![Serotype Distribution](images/serotype_distribution.png)\n\n")
        f.write("![Serotype Distribution Percentage](images/serotype_distribution_percentage.png)\n\n")
        
        # Agreement metrics
        f.write("## Agreement Between Implementations\n\n")
        f.write("### Overall Agreement\n\n")
        f.write("![Percentage Agreement](images/percentage_agreement.png)\n\n")
        f.write("![Cohen's Kappa](images/cohens_kappa.png)\n\n")
        
        f.write("### Agreement Metrics\n\n")
        f.write("| Implementation Pair | Percentage Agreement | Cohen's Kappa |\n")
        f.write("|---------------------|----------------------|--------------|\n")
        
        for key, data in metrics.items():
            if 'percentage_agreement' in data and 'cohens_kappa' in data:
                f.write(f"| {key} | {data['percentage_agreement']:.2f}% | {data['cohens_kappa']:.3f} |\n")
        
        # Confusion matrices
        f.write("\n## Confusion Matrices\n\n")
        
        for key in metrics.keys():
            f.write(f"### {key}\n\n")
            f.write(f"![Confusion Matrix](images/confusion_matrix_{key}.png)\n\n")
        
        # Read assignment changes
        f.write("## Read Assignment Changes\n\n")
        
        for impl1, impl2 in itertools.combinations(implementations, 2):
            key = f"{impl1}_vs_{impl2}"
            f.write(f"### {key}\n\n")
            f.write(f"![Assignment Changes](images/assignment_changes_{key}.png)\n\n")
            
            # Calculate disagreement statistics
            valid_df = assignment_df[~((assignment_df[impl1] == 'unknown') | (assignment_df[impl2] == 'unknown'))]
            changes_df = valid_df[valid_df[impl1] != valid_df[impl2]]
            
            if len(valid_df) > 0:
                disagreement_pct = (len(changes_df) / len(valid_df)) * 100
                f.write(f"**Disagreement rate:** {disagreement_pct:.2f}% ({len(changes_df)} out of {len(valid_df)} reads)\n\n")
                
                # Most common changes
                if len(changes_df) > 0:
                    change_counts = defaultdict(int)
                    for _, row in changes_df.iterrows():
                        change_key = f"{row[impl1]}_to_{row[impl2]}"
                        change_counts[change_key] += 1
                    
                    sorted_changes = sorted(change_counts.items(), key=lambda x: x[1], reverse=True)
                    
                    f.write("**Most common changes:**\n\n")
                    for change, count in sorted_changes[:5]:  # Top 5 changes
                        from_serotype, to_serotype = change.split('_to_')
                        pct = (count / len(changes_df)) * 100
                        f.write(f"- {from_serotype} → {to_serotype}: {count} reads ({pct:.1f}% of changes)\n")
                    f.write("\n")
            else:
                f.write("No valid data for comparison.\n\n")
        
        # Conclusion
        f.write("## Conclusion\n\n")
        
        # Calculate overall agreement across all implementation pairs
        agreement_values = [data['percentage_agreement'] for key, data in metrics.items() 
                           if 'percentage_agreement' in data]
        kappa_values = [data['cohens_kappa'] for key, data in metrics.items() 
                       if 'cohens_kappa' in data]
        
        if agreement_values and kappa_values:
            avg_agreement = sum(agreement_values) / len(agreement_values)
            avg_kappa = sum(kappa_values) / len(kappa_values)
            
            f.write(f"The average agreement between implementations is **{avg_agreement:.2f}%** with an average Cohen's Kappa of **{avg_kappa:.3f}**.\n\n")
            
            # Interpret Kappa value
            if avg_kappa < 0.2:
                kappa_interpretation = "slight"
            elif avg_kappa < 0.4:
                kappa_interpretation = "fair"
            elif avg_kappa < 0.6:
                kappa_interpretation = "moderate"
            elif avg_kappa < 0.8:
                kappa_interpretation = "substantial"
            else:
                kappa_interpretation = "almost perfect"
            
            f.write(f"This indicates **{kappa_interpretation}** agreement between the implementations.\n\n")
            
            # Find the most similar pair
            if len(agreement_values) > 1:
                most_similar_idx = agreement_values.index(max(agreement_values))
                most_similar_pair = list(metrics.keys())[most_similar_idx]
                most_similar_agreement = agreement_values[most_similar_idx]
                
                f.write(f"The most similar implementations are **{most_similar_pair}** with {most_similar_agreement:.2f}% agreement.\n\n")
            
            # Find the most different pair
            if len(agreement_values) > 1:
                most_different_idx = agreement_values.index(min(agreement_values))
                most_different_pair = list(metrics.keys())[most_different_idx]
                most_different_agreement = agreement_values[most_different_idx]
                
                f.write(f"The most different implementations are **{most_different_pair}** with {most_different_agreement:.2f}% agreement.\n\n")
    
    if verbose:
        print(f"Generated markdown report at {report_path}")
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
            <title>PAF Processor Output Comparison Report</title>
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
        
        if verbose:
            print(f"Generated HTML report at {html_path}")
        return html_path
    except ImportError:
        if verbose:
            print("Warning: Could not generate HTML report. Python markdown package not installed.")
            print("Install with: pip install markdown")
        return None