#!/usr/bin/env python3

import argparse

def extract_chunks_streaming(input_file, output_file, rows_per_chunk, max_chunks):
    current_chunk_id = None
    current_chunk_count = 0
    rows_written = 0
    chunk_tracker = {}

    with open(input_file, 'r') as infile, open(output_file, 'w') as out:
        for line in infile:
            first_col = line.split('\t', 1)[0]

            if first_col not in chunk_tracker:
                if len(chunk_tracker) >= max_chunks:
                    continue  # skip new chunks after limit is reached
                chunk_tracker[first_col] = 0

            if chunk_tracker[first_col] < rows_per_chunk:
                out.write(line)
                chunk_tracker[first_col] += 1

def main():
    parser = argparse.ArgumentParser(description="Stream and extract rows per chunk from a large PAF file.")
    parser.add_argument('-i', '--input', required=True, help="Path to the input PAF file.")
    parser.add_argument('-o', '--output', default="output.paf", help="Path to the output file. [default: output.paf]")
    parser.add_argument('-n', '--rows-per-chunk', type=int, required=True, help="Number of rows to keep per chunk.")
    parser.add_argument('-c', '--max-chunks', type=int, required=True, help="Maximum number of chunks to process.")

    args = parser.parse_args()
    extract_chunks_streaming(args.input, args.output, args.rows_per_chunk, args.max_chunks)

if __name__ == "__main__":
    main()

