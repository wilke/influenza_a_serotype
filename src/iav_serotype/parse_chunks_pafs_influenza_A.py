#!/usr/bin/env python3

import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import os
import argparse

def process_qname_group(group_df, flu_info_df, score_thresh):
    """Process a DataFrame containing rows with the same qname."""
    merge_df = pd.merge(group_df, flu_info_df, left_on="tname", right_on="accession")
    assignment_df = (merge_df.groupby(["qname", "tname", "serotype", "segment", "strand"])
                     .agg(tot_read_length=("qlength", "sum"),
                          tot_align=("align_length", "sum"),
                          tot_match=("num_matches", "sum"))
                     .assign(ANI=lambda x: x["tot_match"] / x["tot_align"],
                             AF=lambda x: x["tot_align"] / x["tot_read_length"],
                             align_score=lambda x: x["ANI"] * x["AF"])
                     .reset_index())
    return assignment_df

def main():
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

    # Read flu info database
    flu_info_df = pd.read_csv(flu_info_db, sep="\t", header=0)

    # Create output directory
    os.makedirs(out_dir, exist_ok=True)

    # Initialize variables
    buffer = []
    unique_qnames = set()
    assignment_dfs = []

    paf_cols = ["qname", "qlength", "qstart", "qend", "strand",
                "tname", "tlength", "tstart", "tend",
                "num_matches", "align_length", "mapq"]

    # Process the PAF file line by line
    with open(paf_file, "r") as paf:
        for line in paf:
            row = line.strip().split("\t")
            qname = row[0]

            # Add the row to the buffer
            buffer.append(row)
            unique_qnames.add(qname)

            # If buffer contains buffer_size unique qnames, process it
            if len(unique_qnames) >= buffer_size:
                group_df = pd.DataFrame(buffer, columns=paf_cols)
                assignment_dfs.append(process_qname_group(group_df, flu_info_df, score_thresh))
                buffer = []  # Clear the buffer
                unique_qnames = set()  # Reset the unique qname tracker

        # Process the last buffer
        if buffer:
            group_df = pd.DataFrame(buffer, columns=paf_cols)
            assignment_dfs.append(process_qname_group(group_df, flu_info_df, score_thresh))

    # Combine all processed groups
    assignment_df = pd.concat(assignment_dfs)

    # Summarize scores
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
    sum_df.to_csv(f"{out_dir}/{sample_name}_read_summary.tsv", sep="\t", index=False)

    # Plot serotype assignment
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
    plt.savefig(f"{out_dir}/{sample_name}_read_serotype_assignment.pdf")

    # Write individual read assignments
    if write_individual_assignments:
        for read_assignment, group in sum_df.groupby("read_assignment"):
            group[["qname"]].to_csv(f"{out_dir}/{sample_name}_{read_assignment}.txt",
                                    sep="\t", index=False, header=False)

if __name__ == "__main__":
    main()