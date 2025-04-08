#!/usr/bin/env pythom

import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import os
import argparse

def main():
    parser = argparse.ArgumentParser(description="Parse PAFs for Influenza A serotyping.")
    parser.add_argument("flu_info_db", help="Path to the flu info database file")
    parser.add_argument("paf_file", help="Path to the PAF file")
    parser.add_argument("sample_name", help="Sample name")
    parser.add_argument("out_dir", help="Output directory")
    parser.add_argument("score_threshold", type=float, help="Score threshold")
    args = parser.parse_args()

    flu_info_db = args.flu_info_db
    paf_file = args.paf_file
    sample_name = args.sample_name
    out_dir = args.out_dir
    score_thresh = args.score_threshold

    # Read input files
    paf_cols = ["qname", "qlength", "qstart", "qend", "strand",
                "tname", "tlength", "tstart", "tend",
                "num_matches", "align_length", "mapq"]
    paf_df = pd.read_csv(paf_file, sep="\t", header=None, names=paf_cols, usecols=range(12))
    flu_info_df = pd.read_csv(flu_info_db, sep="\t", header=0)

    # Merge dataframes
    merge_df = pd.merge(paf_df, flu_info_df, left_on="tname", right_on="accession")

    # Group and summarize
    assignment_df = (merge_df.groupby(["qname", "tname", "serotype", "segment", "strand"])
                     .agg(tot_read_length=("qlength", "sum"),
                          tot_align=("align_length", "sum"),
                          tot_match=("num_matches", "sum"))
                     .assign(ANI=lambda x: x["tot_match"] / x["tot_align"],
                             AF=lambda x: x["tot_align"] / x["tot_read_length"],
                             align_score=lambda x: x["ANI"] * x["AF"])
                     .reset_index()
                     .sort_values(by=["qname", "align_score"], ascending=[True, False]))

    # Summarize scores
    sum_df = (assignment_df.groupby(["qname", "serotype", "segment"])
              .agg(n=("align_score", "size"),
                   top_score=("align_score", "max"),
                   avg_score=("align_score", "mean"))
              .reset_index()
              .sort_values(by=["qname", "top_score"], ascending=[True, False]))

    sum_df = (sum_df.groupby("qname", group_keys=False)
              .apply(lambda x: x.nlargest(2, "top_score"))
              .assign(read_assignment=lambda x: np.where(
                  (x.groupby("qname")["top_score"].transform("max") - 0.003 >= x["top_score"].min()) |
                  (x.groupby("qname")["serotype"].transform("nunique") == 1),
                  x["serotype"].iloc[0],
                  "ambiguous"))
              .query("top_score >= @score_thresh"))

    # Write summary table
    os.makedirs(out_dir, exist_ok=True)
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
    for read_assignment, group in sum_df.groupby("read_assignment"):
        group[["qname"]].to_csv(f"{out_dir}/{sample_name}_{read_assignment}.txt",
                                sep="\t", index=False, header=False)

if __name__ == "__main__":
    main()