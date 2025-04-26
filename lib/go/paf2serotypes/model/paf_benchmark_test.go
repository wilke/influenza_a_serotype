package model

import (
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

// BenchmarkNewPafRecord measures the performance of PAF record parsing
func BenchmarkNewPafRecord(b *testing.B) {
	fields := []string{
		"read1", // qname
		"1000",  // qlength
		"0",     // qstart
		"1000",  // qend
		"+",     // strand
		"ref1",  // tname
		"2000",  // tlength
		"500",   // tstart
		"1500",  // tend
		"950",   // num_matches
		"1000",  // align_length
		"60",    // mapq
	}

	// Report memory statistics before the benchmark
	reportMemStats(b, "Before")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = NewPafRecord(fields)
	}

	b.StopTimer()
	// Report memory statistics after the benchmark
	reportMemStats(b, "After")
}

// reportMemStats reports memory statistics for benchmarking
func reportMemStats(b *testing.B, label string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	b.Logf("%s: Alloc = %v MiB, TotalAlloc = %v MiB, Sys = %v MiB, NumGC = %v",
		label,
		m.Alloc/1024/1024,
		m.TotalAlloc/1024/1024,
		m.Sys/1024/1024,
		m.NumGC)
}

// BenchmarkCalculateScores_Small measures performance with a small dataset
func BenchmarkCalculateScores_Small(b *testing.B) {
	benchmarkCalculateScores(b, 1000)
}

// BenchmarkCalculateScores_Medium measures performance with a medium dataset
func BenchmarkCalculateScores_Medium(b *testing.B) {
	benchmarkCalculateScores(b, 10000)
}

// BenchmarkCalculateScores_Large measures performance with a large dataset
func BenchmarkCalculateScores_Large(b *testing.B) {
	benchmarkCalculateScores(b, 50000)
}

// benchmarkCalculateScores is a helper function for benchmarking CalculateScores with different dataset sizes
func benchmarkCalculateScores(b *testing.B, numRecords int) {
	b.Logf("Benchmarking with %d records", numRecords)

	// Create test data
	records := make([]PafRecord, numRecords)

	for i := 0; i < numRecords; i++ {
		records[i] = PafRecord{
			Qname:       "read1",
			Qlength:     1000,
			Qstart:      0,
			Qend:        1000,
			Strand:      "+",
			Tname:       "ref1",
			Tlength:     2000,
			Tstart:      500,
			Tend:        1500,
			NumMatches:  950,
			AlignLength: 1000,
			Mapq:        60,
		}
	}

	// Create test mapping database
	db := &MappingDatabase{
		Entries: map[string]MappingEntry{
			"ref1": {
				Accession: "ref1",
				Serotype:  "H1N1",
				Segment:   1,
			},
		},
	}

	// Report memory statistics before the benchmark
	reportMemStats(b, "Before")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CalculateScores(records, db)
	}

	b.StopTimer()
	// Report memory statistics after the benchmark
	reportMemStats(b, "After")
}

// BenchmarkAssignSerotypes_Small measures performance with a small dataset
func BenchmarkAssignSerotypes_Small(b *testing.B) {
	benchmarkAssignSerotypes(b, 1000)
}

// BenchmarkAssignSerotypes_Medium measures performance with a medium dataset
func BenchmarkAssignSerotypes_Medium(b *testing.B) {
	benchmarkAssignSerotypes(b, 10000)
}

// BenchmarkAssignSerotypes_Large measures performance with a large dataset
func BenchmarkAssignSerotypes_Large(b *testing.B) {
	benchmarkAssignSerotypes(b, 50000)
}

// benchmarkAssignSerotypes is a helper function for benchmarking AssignSerotypes with different dataset sizes
func benchmarkAssignSerotypes(b *testing.B, numScores int) {
	b.Logf("Benchmarking with %d scores", numScores)

	// Create test data
	scores := make([]AlignmentScore, numScores)

	// Create a mix of different reads and serotypes to test real-world scenarios
	serotypes := []string{"H1N1", "H3N2", "H5N1", "H7N9"}

	for i := 0; i < numScores; i++ {
		readName := fmt.Sprintf("read%d", i%100) // Create 100 unique reads
		serotype := serotypes[i%len(serotypes)]  // Cycle through serotypes

		scores[i] = AlignmentScore{
			Qname:       readName,
			Tname:       fmt.Sprintf("ref%d", i%10),
			Serotype:    serotype,
			Segment:     (i % 8) + 1, // Segments 1-8
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  950,
			ANI:         0.95,
			AF:          1.0,
			AlignScore:  0.95 - (float64(i%10) * 0.01), // Vary scores slightly
		}
	}

	// Report memory statistics before the benchmark
	reportMemStats(b, "Before")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AssignSerotypes(scores, 0.8, 0.003)
	}

	b.StopTimer()
	// Report memory statistics after the benchmark
	reportMemStats(b, "After")
}

// BenchmarkWriteSummary measures the performance of writing summaries
func BenchmarkWriteSummary(b *testing.B) {
	// Create test data
	numSummaries := 1000
	summaries := make([]SerotypeSummary, numSummaries)

	for i := 0; i < numSummaries; i++ {
		summaries[i] = SerotypeSummary{
			Qname:          fmt.Sprintf("read%d", i%100),
			Serotype:       fmt.Sprintf("H%dN%d", (i%5)+1, (i%3)+1),
			Segment:        (i % 8) + 1,
			Count:          1,
			TopScore:       0.95,
			AvgScore:       0.95,
			ReadAssignment: fmt.Sprintf("H%dN%d", (i%5)+1, (i%3)+1),
		}
	}

	// Skip actual file writing in benchmark
	b.Skip("Skipping benchmark that writes to disk")

	// Report memory statistics before the benchmark
	reportMemStats(b, "Before")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WriteSummary(summaries, "/tmp", "benchmark")
	}

	b.StopTimer()
	// Report memory statistics after the benchmark
	reportMemStats(b, "After")
}

// BenchmarkPafRecordCreation_WithPool measures the performance of PAF record creation with object pooling
func BenchmarkPafRecordCreation_WithPool(b *testing.B) {
	// Create a pool of PAF records
	pool := sync.Pool{
		New: func() interface{} {
			return new(PafRecord)
		},
	}

	fields := []string{
		"read1", // qname
		"1000",  // qlength
		"0",     // qstart
		"1000",  // qend
		"+",     // strand
		"ref1",  // tname
		"2000",  // tlength
		"500",   // tstart
		"1500",  // tend
		"950",   // num_matches
		"1000",  // align_length
		"60",    // mapq
	}

	// Report memory statistics before the benchmark
	reportMemStats(b, "Before")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Get a record from the pool
		record := pool.Get().(*PafRecord)

		// Parse the fields
		record.Qname = fields[0]
		record.Qlength, _ = strconv.Atoi(fields[1])
		record.Qstart, _ = strconv.Atoi(fields[2])
		record.Qend, _ = strconv.Atoi(fields[3])
		record.Strand = fields[4]
		record.Tname = fields[5]
		record.Tlength, _ = strconv.Atoi(fields[6])
		record.Tstart, _ = strconv.Atoi(fields[7])
		record.Tend, _ = strconv.Atoi(fields[8])
		record.NumMatches, _ = strconv.Atoi(fields[9])
		record.AlignLength, _ = strconv.Atoi(fields[10])
		record.Mapq, _ = strconv.Atoi(fields[11])

		// Return the record to the pool
		pool.Put(record)
	}

	b.StopTimer()
	// Report memory statistics after the benchmark
	reportMemStats(b, "After")
}
