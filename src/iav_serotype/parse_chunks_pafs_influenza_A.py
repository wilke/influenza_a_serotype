#!/usr/bin/env python3

import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import os
import argparse
import logging

def process_qname_group(group_df, flu_info_df, score_thresh):
    """Process a DataFrame containing rows with the same qname."""
    logging.debug(f"Processing group with {len(group_df)} rows.")

    # Ensure numeric columns are properly converted
    numeric_cols = ["qlength", "qstart", "qend", "tlength", "tstart", "tend", "num_matches", "align_length", "mapq"]
    for col in numeric_cols:
        group_df[col] = pd.to_numeric(group_df[col], errors="coerce")

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

    logging.debug(f"Finished processing group with {len(assignment_df)} rows.")
    return assignment_df

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
    args = parser.parse_args()

    flu_info_db = args.flu_info_db
    paf_file = args.paf_file
    sample_name = args.sample_name
    out_dir = args.out_dir
    score_thresh = args.score_threshold
    buffer_size = args.buffer_size
    write_individual_assignments = args.write_individual_assignments

    logging.info("Reading flu info database.")
    flu_info_df = pd.read_csv(flu_info_db, sep="\t", header=0)
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
                assignment_dfs.append(process_qname_group(group_df, flu_info_df, score_thresh))
                buffer = []  # Clear the buffer
                unique_qnames = set()  # Reset the unique qname tracker

        # Process the last buffer
        if buffer:
            logging.info(f"Processing final buffer with {len(buffer)} rows and {len(unique_qnames)} unique qnames.")
            group_df = pd.DataFrame(buffer, columns=paf_cols)
            assignment_dfs.append(process_qname_group(group_df, flu_info_df, score_thresh))

    # Combine all processed groups
    logging.info("Combining all processed groups.")
    assignment_df = pd.concat(assignment_dfs)

    # Summarize scores
    logging.info("Summarizing scores.")
    sum_df = (assignment_df.groupby(["qname", "serotype", "segment"])
              .agg(n=("align_score", "size"),
                   top_score=("align_score", "max"),
                   avg_score=("align_score", "mean"))
              .reset_index())

    sum_df = (sum_df.groupby("qname", group_keys=False)
              .apply(lambda x: x.nlargest(2, "top_score"))
              .assign(read_assignment=lambda x: np.where(
                  (x.groupby("qname")["top_score"].transform("max") - 0.003 >= x["top_score"].min()) |
                  (x.groupby("qname")["serotype"].transform("nunique") == 1),
                  x["serotype"].iloc[0],
                  "ambiguous"))
              .query("top_score >= @score_thresh"))

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