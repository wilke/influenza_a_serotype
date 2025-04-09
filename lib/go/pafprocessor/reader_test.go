package pafprocessor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPAFFileChunking(t *testing.T) {
	// Create a temporary test file
	tmpDir, err := os.MkdirTemp("", "pafprocessor-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test PAF file with entries with same and different QNames
	testPAFContent := `read1	1000	0	100	+	target1	2000	0	100	90	100	60
read1	1000	200	300	+	target2	2000	100	200	95	100	60
read2	1000	0	100	+	target1	2000	300	400	85	100	60
read2	1000	200	300	+	target3	2000	0	100	80	100	60
read3	1000	0	100	+	target1	2000	500	600	70	100	60
`
	pafFilePath := filepath.Join(tmpDir, "test.paf")
	if err := os.WriteFile(pafFilePath, []byte(testPAFContent), 0644); err != nil {
		t.Fatalf("Failed to write test PAF file: %v", err)
	}

	// Test with chunkSize=1 (one unique QName per chunk)
	t.Run("ChunkSize=1", func(t *testing.T) {
		chunks, err := ReadPAFFile(pafFilePath, 1)
		if err != nil {
			t.Fatalf("Failed to read PAF file: %v", err)
		}

		// First chunk should contain 2 entries for read1
		chunk1 := <-chunks
		if len(chunk1) != 2 {
			t.Errorf("Expected 2 entries in first chunk, got %d", len(chunk1))
		}
		if chunk1[0].QName != "read1" || chunk1[1].QName != "read1" {
			t.Errorf("Expected only read1 entries in first chunk, got %s and %s", chunk1[0].QName, chunk1[1].QName)
		}

		// Second chunk should contain 2 entries for read2
		chunk2 := <-chunks
		if len(chunk2) != 2 {
			t.Errorf("Expected 2 entries in second chunk, got %d", len(chunk2))
		}
		if chunk2[0].QName != "read2" || chunk2[1].QName != "read2" {
			t.Errorf("Expected only read2 entries in second chunk, got %s and %s", chunk2[0].QName, chunk2[1].QName)
		}

		// Third chunk should contain 1 entry for read3
		chunk3 := <-chunks
		if len(chunk3) != 1 {
			t.Errorf("Expected 1 entry in third chunk, got %d", len(chunk3))
		}
		if chunk3[0].QName != "read3" {
			t.Errorf("Expected read3 in third chunk, got %s", chunk3[0].QName)
		}
	})

	// Test with chunkSize=2 (two unique QNames per chunk)
	t.Run("ChunkSize=2", func(t *testing.T) {
		chunks, err := ReadPAFFile(pafFilePath, 2)
		if err != nil {
			t.Fatalf("Failed to read PAF file: %v", err)
		}

		// First chunk should contain entries for read1 and read2 (4 entries total)
		chunk1 := <-chunks
		if len(chunk1) != 4 {
			t.Errorf("Expected 4 entries in first chunk, got %d", len(chunk1))
		}

		// Count entries by QName
		qNameCount := make(map[string]int)
		for _, entry := range chunk1 {
			qNameCount[entry.QName]++
		}
		if qNameCount["read1"] != 2 || qNameCount["read2"] != 2 {
			t.Errorf("Expected 2 entries each for read1 and read2, got %v", qNameCount)
		}

		// Second chunk should contain only read3 (1 entry)
		chunk2 := <-chunks
		if len(chunk2) != 1 {
			t.Errorf("Expected 1 entry in second chunk, got %d", len(chunk2))
		}
		if chunk2[0].QName != "read3" {
			t.Errorf("Expected read3 in second chunk, got %s", chunk2[0].QName)
		}
	})

	// Test with chunkSize=3 (all three unique QNames in one chunk)
	t.Run("ChunkSize=3", func(t *testing.T) {
		chunks, err := ReadPAFFile(pafFilePath, 3)
		if err != nil {
			t.Fatalf("Failed to read PAF file: %v", err)
		}

		// Should get all entries in a single chunk (5 entries total)
		chunk := <-chunks
		if len(chunk) != 5 {
			t.Errorf("Expected 5 entries in chunk, got %d", len(chunk))
		}

		// Count entries by QName
		qNameCount := make(map[string]int)
		for _, entry := range chunk {
			qNameCount[entry.QName]++
		}
		if qNameCount["read1"] != 2 || qNameCount["read2"] != 2 || qNameCount["read3"] != 1 {
			t.Errorf("Expected 2 entries for read1, 2 for read2, and 1 for read3, got %v", qNameCount)
		}
	})
}
