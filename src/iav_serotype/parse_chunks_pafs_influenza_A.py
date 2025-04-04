#!/usr/bin/env python3

import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import os
import argparse
import logging

def process_qname_group(group_df, flu_info_df, score_thresh, paired_reads=False, debug=False):
    """Process a DataFrame containing rows with the same qname."""
    logging.debug(f"Processing group with {len(group_df)} rows.")

    # Ensure numeric columns are properly converted
    numeric_cols = ["qlength", "qstart", "qend", "tlength", "tstart", "tend", "num_matches", "align_length", "mapq"]
    for col in numeric_cols:
        group_df[col] = pd.to_numeric(group_df[col], errors="coerce")

    flu_info_df["segment"] = pd.to_numeric(flu_info_df["segment"], errors="coerce")

    # Drop rows with NaN values
    group_df.dropna(inplace=True)

    # Merge with flu info database
    merge_df = pd.merge(group_df, flu_info_df, left_on="tname", right_on="accession")

    # Perform group-by operations
    assignment_df = (merge_df.groupby(["qname", "tname", "serotype", "segment", "strand"])
                     .agg(tot_read_length=("qlength", "sum"),
                          tot_align=("align_length", "sum"),
                          tot_match=("num_matches", "sum"))
                     .assign(ANI=lambda x: x["tot_match"] / x["tot_align"],
                             AF=lambda x: x["tot_align"] / x["tot_read_length"],
                             align_score=lambda x: x["ANI"] * x["AF"])
                     .reset_index())
    
    # Add debug information for assignment_df
    logging.debug(f"Assignment DataFrame:\n{assignment_df.head()}")

    if paired_reads:
        # Handle forward and reverse reads separately
        group = ["qname", "serotype", "segment", "strand"]
    else:
        # Handle reads by qname only - no strand information ; forward and reverse reads are lumped together
        group = ["qname", "serotype", "segment"]
       
    # Summarize scores
    logging.info("Summarizing scores.")
    qss_df = (assignment_df.groupby(group)
              .agg(n=("align_score", "size"),
                   top_score=("align_score", "max"),
                   avg_score=("align_score", "mean"))
              .reset_index())

    # Add debug information for sum_df
    logging.debug(f"Summary DataFrame:\n{qss_df.head()}")

    if paired_reads:
        group = ["qname", "strand"]

        # Get max top_score in each group
        qs_max_top_score_df = (qss_df.groupby(group)
                          .agg(max_top_score=("top_score", "max"))
                          .reset_index())
        
        # Get all entries in group_df within 0.003 of max top_score
        top_df = (qss_df.merge(qs_max_top_score_df, on=group)
                         .query("abs(top_score - max_top_score) <= 0.003"))
        
        logging.debug(f"Max top score DataFrame:\n{qs_max_top_score_df.head()}")
        logging.debug(f"All entries within 0.003 DataFrame:\n{top_df.head()}")

        # First, get the serotype and segment for each qname
        qname_serotype_segment = top_df.groupby("qname").agg({
            "serotype": lambda x: x.iloc[0] if len(set(x)) == 1 else x.iloc[0],
            "segment": "first"
        }).reset_index()
        
        # Store original counts for each qname to ensure accurate n values
        qname_counts = {}
        for qname in top_df['qname'].unique():
            qname_counts[qname] = len(group_df[group_df['qname'] == qname])
        
        # Group by qname and calculate distinct serotypes and segments
        sum_df = (top_df.groupby(["qname"])
            .agg(n=("n", "sum"),
                 top_score=("top_score", "max"),
                 avg_score=("top_score", "mean"),
                 distinct_serotypes=("serotype", lambda x: len(set(x))),
                 distinct_segments=("segment", lambda x: len(set(x)))))
            
        # Create a dictionary to store segment lists for each qname
        segment_lists = {}
        for qname in top_df['qname'].unique():
            segments = sorted(list(set(top_df[top_df['qname'] == qname]['segment'])))
            segment_lists[qname] = str(segments) if len(segments) > 1 else segments[0] if segments else "ambiguous"
        
        # Debug the segment lists
        for qname, segments in segment_lists.items():
            logging.debug(f"Segments for {qname}: {segments}")
        
        # Create a dictionary to store serotypes for each qname
        serotype_dict = {}
        for qname in top_df['qname'].unique():
            serotypes = list(set(top_df[top_df['qname'] == qname]['serotype']))
            serotype_dict[qname] = serotypes[0] if len(serotypes) == 1 else "ambiguous"
        
        # Assign read_assignment based on distinct_serotypes and distinct_segments
        sum_df = (sum_df.assign(read_assignment=lambda x: np.where(
                  (x["distinct_serotypes"] == 1) & (x["distinct_segments"] == 1),
                  x.index.map(lambda idx: serotype_dict[idx]),
                  x.index.map(lambda idx: segment_lists[idx])))
              .query("top_score >= @score_thresh")
              .drop(columns=["distinct_serotypes", "distinct_segments"]))  # Remove distinct columns from output
            
        # Merge with serotype and segment information
        sum_df = sum_df.reset_index().merge(qname_serotype_segment, on="qname")
        
        # Fix the n values for all qnames to match their original counts
        for qname, original_count in qname_counts.items():
            if qname in sum_df['qname'].values:
                current_n = sum_df.loc[sum_df['qname'] == qname, 'n'].iloc[0]
                if original_count != current_n:
                    sum_df.loc[sum_df['qname'] == qname, 'n'] = original_count
      
        logging.debug(f"Assignment DataFrame after merging top scores:\n{sum_df.head()}")
        
        return sum_df
    else:
        # First create a summary DataFrame with the same structure as in parse_pafs_influenza_A.py
        sum_df = (assignment_df.groupby(["qname", "serotype", "segment"])
              .agg(n=("align_score", "size"),
                   top_score=("align_score", "max"),
                   avg_score=("align_score", "mean"))
              .reset_index()
              .sort_values(by=["qname", "top_score"], ascending=[True, False]))
        
        # Then filter and assign serotype
        # First, get the top 2 scores for each qname
        top_scores_df = sum_df.groupby("qname", group_keys=False).apply(
            lambda x: x.nlargest(2, "top_score"), include_groups=False
        ).reset_index()
        
        # Calculate max score and number of unique serotypes for each qname
        qname_stats = top_scores_df.groupby("qname").agg(
            max_score=("top_score", "max"),
            min_score=("top_score", "min"),
            num_unique_serotypes=("serotype", "nunique")
        )
        
        # Merge the stats back to the top scores DataFrame
        top_scores_df = top_scores_df.merge(qname_stats, on="qname")
        
        # Then assign read_assignment
        sum_df = (top_scores_df
              .assign(read_assignment=lambda x: np.where(
                  (x["max_score"] - 0.003 >= x["min_score"]) | (x["num_unique_serotypes"] == 1),
                  x["serotype"],
                  "ambiguous"))
              .query("top_score >= @score_thresh"))
        
        # Add debug information for final assignment_df
        logging.debug(f"Final assignment DataFrame:\n{assignment_df.head()}")

        # Log debug information but don't exit
        if debug:
            logging.debug("Debug mode is enabled. Continuing execution.")

        logging.debug(f"Finished processing group with {len(assignment_df)} rows.")
        return sum_df

def main():
    # Set up logging
    logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")

    parser = argparse.ArgumentParser(description="Parse PAFs for Influenza A serotyping.")
    parser.add_argument("flu_info_db", help="Path to the flu info database file")
    parser.add_argument("paf_file", help="Path to the PAF file")
    parser.add_argument("sample_name", help="Sample name")
    parser.add_argument("out_dir", help="Output directory")
    parser.add_argument("score_threshold", type=float, help="Score threshold")
    parser.add_argument("--buffer_size", type=int, default=1000, help="Buffer size for unique qname values")
    parser.add_argument("--write_individual_assignments", action="store_true", help="Write individual read assignments")
    parser.add_argument("--paired_reads", action="store_true", help="Flag for paired reads")
    parser.add_argument("--debug", action="store_true", help="Enable debug logging")
    args = parser.parse_args()

    flu_info_db = args.flu_info_db
    paf_file = args.paf_file
    sample_name = args.sample_name
    out_dir = args.out_dir
    score_thresh = args.score_threshold
    buffer_size = args.buffer_size
    write_individual_assignments = args.write_individual_assignments

    if args.debug:
        logging.getLogger().setLevel(logging.DEBUG)

    logging.info("Reading flu info database.")
    # either int or str
    
    flu_info_df = pd.read_csv(flu_info_db, sep="\t", header=0)
    # flu_info_df = pd.read_csv(flu_info_db, sep="\t", header=0)
    logging.info(f"Flu info database loaded with {len(flu_info_df)} rows.")


    # Create output directory
    os.makedirs(out_dir, exist_ok=True)
    logging.info(f"Output directory created: {out_dir}")

    # Initialize variables
    buffer = []
    unique_qnames = set()
    assignment_dfs = []

    paf_cols = ["qname", "qlength", "qstart", "qend", "strand",
                "tname", "tlength", "tstart", "tend",
                "num_matches", "align_length", "mapq"]

    # Process the PAF file line by line
    logging.info(f"Processing PAF file: {paf_file}")
    with open(paf_file, "r") as paf:
        for line in paf:
            row = line.strip().split("\t")
            qname = row[0]

            # Add the row to the buffer
            buffer.append(row)
            unique_qnames.add(qname)

            # If buffer contains buffer_size unique qnames, process it
            if len(unique_qnames) >= buffer_size:
                logging.info(f"Processing buffer with {len(buffer)} rows and {len(unique_qnames)} unique qnames.")
                group_df = pd.DataFrame(buffer, columns=paf_cols)
                assignment_dfs.append(process_qname_group(
                    group_df, 
                    flu_info_df, 
                    score_thresh, 
                    args.paired_reads, 
                    args.debug))
                buffer = []  # Clear the buffer
                unique_qnames = set()  # Reset the unique qname tracker

        # Process the last buffer
        if buffer:
            logging.info(f"Processing final buffer with {len(buffer)} rows and {len(unique_qnames)} unique qnames.")
            group_df = pd.DataFrame(buffer, columns=paf_cols)
            assignment_dfs.append(process_qname_group(group_df, flu_info_df, score_thresh, args.paired_reads, args.debug))

    # Combine all processed groups
    logging.info("Combining all processed groups.")
    sum_df = pd.concat(assignment_dfs)

    # Write summary table
    summary_file = f"{out_dir}/{sample_name}_read_summary.tsv"
    logging.info(f"Writing summary table to {summary_file}.")
    sum_df.to_csv(summary_file, sep="\t", index=False)

    # Plot serotype assignment
    logging.info("Plotting serotype assignment.")
    assignp = (sum_df.groupby("read_assignment").size()
               .reset_index(name="count")
               .sort_values(by="count", ascending=False))

    plt.figure(figsize=(10, 6))
    plt.bar(assignp["read_assignment"], assignp["count"])
    plt.xticks(rotation=90)
    plt.xlabel("Read Assignment")
    plt.ylabel("Count")
    plt.title("Serotype Assignment")
    plt.tight_layout()
    plot_file = f"{out_dir}/{sample_name}_read_serotype_assignment.pdf"
    plt.savefig(plot_file)
    logging.info(f"Serotype assignment plot saved to {plot_file}.")

    # Write individual read assignments
    if write_individual_assignments:
        logging.info("Writing individual read assignments.")
        for read_assignment, group in sum_df.groupby("read_assignment"):
            assignment_file = f"{out_dir}/{sample_name}_{read_assignment}.txt"
            group[["qname"]].to_csv(assignment_file, sep="\t", index=False, header=False)
            logging.info(f"Individual assignment written to {assignment_file}.")

    logging.info("Processing complete.")

if __name__ == "__main__":
    main()