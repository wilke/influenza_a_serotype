package io

import (
	"fmt"
	"os"
	"testing"

	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/model"
)

// BenchmarkReadChunk_SameQname benchmarks the ReadChunk method with records having the same Qname
func BenchmarkReadChunk_SameQname(b *testing.B) {
	// Create a temporary file with test data
	tmpfile, err := os.CreateTemp("", "bench_paf_same_qname_*.paf")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write test data with groups of records having the same Qname
	// This simulates a realistic PAF file where reads map to multiple references
	numGroups := 200
	recordsPerGroup := 5
	totalRecords := numGroups * recordsPerGroup

	b.Logf("Creating test file with %d groups of %d records each (total: %d records)",
		numGroups, recordsPerGroup, totalRecords)

	for i := 0; i < numGroups; i++ {
		qname := fmt.Sprintf("read%d", i)
		for j := 0; j < recordsPerGroup; j++ {
			fmt.Fprintf(tmpfile, "%s\t1000\t0\t1000\t+\tref%d\t2000\t500\t1500\t950\t1000\t60\n", qname, j)
		}
	}
	tmpfile.Close()

	// Run the benchmark with different chunk sizes
	for _, chunkSize := range []int{10, 50, 100, 500} {
		b.Run(fmt.Sprintf("ChunkSize_%d", chunkSize), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create a reader with the test file
				reader, err := NewPafReader(tmpfile.Name())
				if err != nil {
					b.Fatalf("Failed to create PAF reader: %v", err)
				}

				// Read chunks until EOF
				recordCount := 0
				chunkCount := 0
				for {
					chunk, err := reader.ReadChunk(chunkSize)
					if err != nil {
						b.Fatalf("Failed to read chunk: %v", err)
					}
					if chunk == nil {
						break // EOF
					}

					// Count records in the chunk
					records := chunk.([]model.PafRecord)
					recordCount += len(records)
					chunkCount++
				}

				// Verify that we read all records
				if recordCount != totalRecords {
					b.Fatalf("Expected to read %d records, got %d", totalRecords, recordCount)
				}

				reader.Close()
			}
		})
	}
}

// BenchmarkReadChunk_MixedQnames benchmarks the ReadChunk method with a mix of records having different Qnames
func BenchmarkReadChunk_MixedQnames(b *testing.B) {
	// Create a temporary file with test data
	tmpfile, err := os.CreateTemp("", "bench_paf_mixed_qnames_*.paf")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write test data with a mix of records having different Qnames
	totalRecords := 1000
	b.Logf("Creating test file with %d records with unique Qnames", totalRecords)

	for i := 0; i < totalRecords; i++ {
		fmt.Fprintf(tmpfile, "read%d\t1000\t0\t1000\t+\tref%d\t2000\t500\t1500\t950\t1000\t60\n", i, i%10)
	}
	tmpfile.Close()

	// Run the benchmark with different chunk sizes
	for _, chunkSize := range []int{10, 50, 100, 500} {
		b.Run(fmt.Sprintf("ChunkSize_%d", chunkSize), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create a reader with the test file
				reader, err := NewPafReader(tmpfile.Name())
				if err != nil {
					b.Fatalf("Failed to create PAF reader: %v", err)
				}

				// Read chunks until EOF
				recordCount := 0
				chunkCount := 0
				for {
					chunk, err := reader.ReadChunk(chunkSize)
					if err != nil {
						b.Fatalf("Failed to read chunk: %v", err)
					}
					if chunk == nil {
						break // EOF
					}

					// Count records in the chunk
					records := chunk.([]model.PafRecord)
					recordCount += len(records)
					chunkCount++
				}

				// Verify that we read all records
				if recordCount != totalRecords {
					b.Fatalf("Expected to read %d records, got %d", totalRecords, recordCount)
				}

				reader.Close()
			}
		})
	}
}

// BenchmarkReadChunk_LargeGroups benchmarks the ReadChunk method with large groups of records having the same Qname
func BenchmarkReadChunk_LargeGroups(b *testing.B) {
	// Create a temporary file with test data
	tmpfile, err := os.CreateTemp("", "bench_paf_large_groups_*.paf")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write test data with large groups of records having the same Qname
	numGroups := 10
	recordsPerGroup := 100
	totalRecords := numGroups * recordsPerGroup

	b.Logf("Creating test file with %d groups of %d records each (total: %d records)",
		numGroups, recordsPerGroup, totalRecords)

	for i := 0; i < numGroups; i++ {
		qname := fmt.Sprintf("read%d", i)
		for j := 0; j < recordsPerGroup; j++ {
			fmt.Fprintf(tmpfile, "%s\t1000\t0\t1000\t+\tref%d\t2000\t500\t1500\t950\t1000\t60\n", qname, j)
		}
	}
	tmpfile.Close()

	// Run the benchmark with different chunk sizes
	for _, chunkSize := range []int{10, 50, 100, 500} {
		b.Run(fmt.Sprintf("ChunkSize_%d", chunkSize), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Create a reader with the test file
				reader, err := NewPafReader(tmpfile.Name())
				if err != nil {
					b.Fatalf("Failed to create PAF reader: %v", err)
				}

				// Read chunks until EOF
				recordCount := 0
				chunkCount := 0
				for {
					chunk, err := reader.ReadChunk(chunkSize)
					if err != nil {
						b.Fatalf("Failed to read chunk: %v", err)
					}
					if chunk == nil {
						break // EOF
					}

					// Count records in the chunk
					records := chunk.([]model.PafRecord)
					recordCount += len(records)
					chunkCount++
				}

				// Verify that we read all records
				if recordCount != totalRecords {
					b.Fatalf("Expected to read %d records, got %d", totalRecords, recordCount)
				}

				reader.Close()
			}
		})
	}
}
