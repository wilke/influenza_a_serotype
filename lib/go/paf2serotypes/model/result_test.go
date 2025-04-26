package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAssignSerotypes_BasicAssignment(t *testing.T) {
	// Create test alignment scores
	scores := []AlignmentScore{
		{
			Qname:       "read1",
			Tname:       "ref1",
			Serotype:    "H1N1",
			Segment:     1,
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  950,
			ANI:         0.95,
			AF:          1.0,
			AlignScore:  0.95,
		},
	}

	// Test with default thresholds
	summaries := AssignSerotypes(scores, 0.8, 0.003)

	// Check results
	if len(summaries) != 1 {
		t.Errorf("Expected 1 summary, got %d", len(summaries))
	}

	if summaries[0].Qname != "read1" {
		t.Errorf("Expected Qname to be 'read1', got '%s'", summaries[0].Qname)
	}

	if summaries[0].Serotype != "H1N1" {
		t.Errorf("Expected Serotype to be 'H1N1', got '%s'", summaries[0].Serotype)
	}

	if summaries[0].ReadAssignment != "H1N1" {
		t.Errorf("Expected ReadAssignment to be 'H1N1', got '%s'", summaries[0].ReadAssignment)
	}
}

func TestAssignSerotypes_ScoreThreshold(t *testing.T) {
	// Create test alignment scores with different scores
	scores := []AlignmentScore{
		{
			Qname:       "read1",
			Tname:       "ref1",
			Serotype:    "H1N1",
			Segment:     1,
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  950,
			ANI:         0.95,
			AF:          1.0,
			AlignScore:  0.95, // Above threshold
		},
		{
			Qname:       "read2",
			Tname:       "ref2",
			Serotype:    "H3N2",
			Segment:     1,
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  700,
			ANI:         0.7,
			AF:          1.0,
			AlignScore:  0.7, // Below threshold
		},
	}

	// Test with threshold of 0.8
	summaries := AssignSerotypes(scores, 0.8, 0.003)

	// Check results - only read1 should be included
	if len(summaries) != 1 {
		t.Errorf("Expected 1 summary, got %d", len(summaries))
	}

	if summaries[0].Qname != "read1" {
		t.Errorf("Expected Qname to be 'read1', got '%s'", summaries[0].Qname)
	}

	// Test with lower threshold of 0.6
	summaries = AssignSerotypes(scores, 0.6, 0.003)

	// Check results - both reads should be included
	if len(summaries) != 2 {
		t.Errorf("Expected 2 summaries, got %d", len(summaries))
	}
}

func TestAssignSerotypes_AmbiguityThreshold(t *testing.T) {
	// Create test alignment scores with close scores
	scores := []AlignmentScore{
		{
			Qname:       "read1",
			Tname:       "ref1",
			Serotype:    "H1N1",
			Segment:     1,
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  950,
			ANI:         0.95,
			AF:          1.0,
			AlignScore:  0.95,
		},
		{
			Qname:       "read1",
			Tname:       "ref2",
			Serotype:    "H3N2",
			Segment:     1,
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  948,
			ANI:         0.948,
			AF:          1.0,
			AlignScore:  0.948, // Very close to the first score (difference: 0.002)
		},
	}

	// Test with default ambiguity threshold (0.003)
	// Difference is 0.002, which is less than 0.003, so should be ambiguous
	summaries := AssignSerotypes(scores, 0.8, 0.003)

	// Check results
	if len(summaries) != 2 {
		t.Errorf("Expected 2 summaries, got %d", len(summaries))
	}

	// Both should have "ambiguous" as ReadAssignment
	for _, summary := range summaries {
		if summary.ReadAssignment != "ambiguous" {
			t.Errorf("Expected ReadAssignment to be 'ambiguous', got '%s'", summary.ReadAssignment)
		}
	}

	// Test with smaller ambiguity threshold (0.001)
	// Difference is 0.002, which is greater than 0.001, so should not be ambiguous
	summaries = AssignSerotypes(scores, 0.8, 0.001)

	// Check results
	if len(summaries) != 2 {
		t.Errorf("Expected 2 summaries, got %d", len(summaries))
	}

	// Both should have H1N1 as ReadAssignment (the highest score)
	for _, summary := range summaries {
		if summary.ReadAssignment != "H1N1" {
			t.Errorf("Expected ReadAssignment to be 'H1N1', got '%s'", summary.ReadAssignment)
		}
	}
}

func TestAssignSerotypes_IdenticalScores(t *testing.T) {
	// Create test alignment scores with identical scores but different serotypes
	scores := []AlignmentScore{
		{
			Qname:       "read1",
			Tname:       "ref1",
			Serotype:    "H1N1",
			Segment:     1,
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  950,
			ANI:         0.95,
			AF:          1.0,
			AlignScore:  0.95,
		},
		{
			Qname:       "read1",
			Tname:       "ref2",
			Serotype:    "H3N2",
			Segment:     1,
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  950,
			ANI:         0.95,
			AF:          1.0,
			AlignScore:  0.95, // Identical to the first score
		},
	}

	// Test with any ambiguity threshold
	summaries := AssignSerotypes(scores, 0.8, 0.003)

	// Check results
	if len(summaries) != 2 {
		t.Errorf("Expected 2 summaries, got %d", len(summaries))
	}

	// Both should have "ambiguous" as ReadAssignment
	for _, summary := range summaries {
		if summary.ReadAssignment != "ambiguous" {
			t.Errorf("Expected ReadAssignment to be 'ambiguous', got '%s'", summary.ReadAssignment)
		}
	}
}

func TestAssignSerotypes_SameSerotype(t *testing.T) {
	// Create test alignment scores with different scores but same serotype
	scores := []AlignmentScore{
		{
			Qname:       "read1",
			Tname:       "ref1",
			Serotype:    "H1N1",
			Segment:     1,
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  950,
			ANI:         0.95,
			AF:          1.0,
			AlignScore:  0.95,
		},
		{
			Qname:       "read1",
			Tname:       "ref2",
			Serotype:    "H1N1", // Same serotype
			Segment:     2,      // Different segment
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  900,
			ANI:         0.9,
			AF:          1.0,
			AlignScore:  0.9, // Lower score
		},
	}

	// Test with default thresholds
	summaries := AssignSerotypes(scores, 0.8, 0.003)

	// Check results
	if len(summaries) != 2 {
		t.Errorf("Expected 2 summaries, got %d", len(summaries))
	}

	// Both should have H1N1 as ReadAssignment (same serotype)
	for _, summary := range summaries {
		if summary.ReadAssignment != "H1N1" {
			t.Errorf("Expected ReadAssignment to be 'H1N1', got '%s'", summary.ReadAssignment)
		}
	}
}

func TestAssignSerotypes_MultipleReads(t *testing.T) {
	// Create test alignment scores for multiple reads
	scores := []AlignmentScore{
		// Read 1 - Clear winner
		{
			Qname:       "read1",
			Tname:       "ref1",
			Serotype:    "H1N1",
			Segment:     1,
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  950,
			ANI:         0.95,
			AF:          1.0,
			AlignScore:  0.95,
		},
		{
			Qname:       "read1",
			Tname:       "ref2",
			Serotype:    "H3N2",
			Segment:     1,
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  900,
			ANI:         0.9,
			AF:          1.0,
			AlignScore:  0.9,
		},
		// Read 2 - Ambiguous
		{
			Qname:       "read2",
			Tname:       "ref3",
			Serotype:    "H5N1",
			Segment:     1,
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  950,
			ANI:         0.95,
			AF:          1.0,
			AlignScore:  0.95,
		},
		{
			Qname:       "read2",
			Tname:       "ref4",
			Serotype:    "H7N9",
			Segment:     1,
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  948,
			ANI:         0.948,
			AF:          1.0,
			AlignScore:  0.948,
		},
		// Read 3 - Same serotype
		{
			Qname:       "read3",
			Tname:       "ref5",
			Serotype:    "H9N2",
			Segment:     1,
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  950,
			ANI:         0.95,
			AF:          1.0,
			AlignScore:  0.95,
		},
		{
			Qname:       "read3",
			Tname:       "ref6",
			Serotype:    "H9N2",
			Segment:     2,
			Strand:      "+",
			ReadLength:  1000,
			AlignLength: 1000,
			NumMatches:  900,
			ANI:         0.9,
			AF:          1.0,
			AlignScore:  0.9,
		},
	}

	// Test with default thresholds
	summaries := AssignSerotypes(scores, 0.8, 0.003)

	// Check results
	if len(summaries) != 6 {
		t.Errorf("Expected 6 summaries, got %d", len(summaries))
	}

	// Group summaries by read
	readSummaries := make(map[string][]SerotypeSummary)
	for _, summary := range summaries {
		readSummaries[summary.Qname] = append(readSummaries[summary.Qname], summary)
	}

	// Check read1 - should be H1N1
	for _, summary := range readSummaries["read1"] {
		if summary.ReadAssignment != "H1N1" {
			t.Errorf("Expected read1 assignment to be 'H1N1', got '%s'", summary.ReadAssignment)
		}
	}

	// Check read2 - should be ambiguous
	for _, summary := range readSummaries["read2"] {
		if summary.ReadAssignment != "ambiguous" {
			t.Errorf("Expected read2 assignment to be 'ambiguous', got '%s'", summary.ReadAssignment)
		}
	}

	// Check read3 - should be H9N2
	for _, summary := range readSummaries["read3"] {
		if summary.ReadAssignment != "H9N2" {
			t.Errorf("Expected read3 assignment to be 'H9N2', got '%s'", summary.ReadAssignment)
		}
	}
}

func TestWriteSummary(t *testing.T) {
	// Create test summaries
	summaries := []SerotypeSummary{
		{
			Qname:          "read1",
			Serotype:       "H1N1",
			Segment:        1,
			Count:          1,
			TopScore:       0.95,
			AvgScore:       0.95,
			ReadAssignment: "H1N1",
		},
		{
			Qname:          "read2",
			Serotype:       "H3N2",
			Segment:        1,
			Count:          1,
			TopScore:       0.9,
			AvgScore:       0.9,
			ReadAssignment: "H3N2",
		},
	}

	// Create temporary directory for test output
	tempDir, err := os.MkdirTemp("", "test_write_summary")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write summary
	err = WriteSummary(summaries, tempDir, "test")
	if err != nil {
		t.Errorf("WriteSummary returned error: %v", err)
	}

	// Check if file exists
	outPath := filepath.Join(tempDir, "test_read_summary.tsv")
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Errorf("Expected summary file to exist at %s", outPath)
	}
}

func TestWriteReadLists(t *testing.T) {
	// Create test summaries
	summaries := []SerotypeSummary{
		{
			Qname:          "read1",
			Serotype:       "H1N1",
			Segment:        1,
			Count:          1,
			TopScore:       0.95,
			AvgScore:       0.95,
			ReadAssignment: "H1N1",
		},
		{
			Qname:          "read2",
			Serotype:       "H3N2",
			Segment:        1,
			Count:          1,
			TopScore:       0.9,
			AvgScore:       0.9,
			ReadAssignment: "H3N2",
		},
		{
			Qname:          "read3",
			Serotype:       "H5N1",
			Segment:        1,
			Count:          1,
			TopScore:       0.95,
			AvgScore:       0.95,
			ReadAssignment: "ambiguous",
		},
		{
			Qname:          "read3",
			Serotype:       "H7N9",
			Segment:        1,
			Count:          1,
			TopScore:       0.948,
			AvgScore:       0.948,
			ReadAssignment: "ambiguous",
		},
	}

	// Create temporary directory for test output
	tempDir, err := os.MkdirTemp("", "test_write_read_lists")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Write read lists
	err = WriteReadLists(summaries, tempDir, "test")
	if err != nil {
		t.Errorf("WriteReadLists returned error: %v", err)
	}

	// Check if files exist
	h1n1Path := filepath.Join(tempDir, "test_H1N1.txt")
	if _, err := os.Stat(h1n1Path); os.IsNotExist(err) {
		t.Errorf("Expected H1N1 read list file to exist at %s", h1n1Path)
	}

	h3n2Path := filepath.Join(tempDir, "test_H3N2.txt")
	if _, err := os.Stat(h3n2Path); os.IsNotExist(err) {
		t.Errorf("Expected H3N2 read list file to exist at %s", h3n2Path)
	}

	ambiguousPath := filepath.Join(tempDir, "test_ambiguous.txt")
	if _, err := os.Stat(ambiguousPath); os.IsNotExist(err) {
		t.Errorf("Expected ambiguous read list file to exist at %s", ambiguousPath)
	}
}
