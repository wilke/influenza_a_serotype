package model

import (
	"testing"
)

func TestNewPafRecord(t *testing.T) {
	// Test case 1: Valid PAF record
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

	record, err := NewPafRecord(fields)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if record.Qname != "read1" {
		t.Errorf("Expected Qname to be 'read1', got '%s'", record.Qname)
	}
	if record.Qlength != 1000 {
		t.Errorf("Expected Qlength to be 1000, got %d", record.Qlength)
	}
	if record.Qstart != 0 {
		t.Errorf("Expected Qstart to be 0, got %d", record.Qstart)
	}
	if record.Qend != 1000 {
		t.Errorf("Expected Qend to be 1000, got %d", record.Qend)
	}
	if record.Strand != "+" {
		t.Errorf("Expected Strand to be '+', got '%s'", record.Strand)
	}
	if record.Tname != "ref1" {
		t.Errorf("Expected Tname to be 'ref1', got '%s'", record.Tname)
	}
	if record.Tlength != 2000 {
		t.Errorf("Expected Tlength to be 2000, got %d", record.Tlength)
	}
	if record.Tstart != 500 {
		t.Errorf("Expected Tstart to be 500, got %d", record.Tstart)
	}
	if record.Tend != 1500 {
		t.Errorf("Expected Tend to be 1500, got %d", record.Tend)
	}
	if record.NumMatches != 950 {
		t.Errorf("Expected NumMatches to be 950, got %d", record.NumMatches)
	}
	if record.AlignLength != 1000 {
		t.Errorf("Expected AlignLength to be 1000, got %d", record.AlignLength)
	}
	if record.Mapq != 60 {
		t.Errorf("Expected Mapq to be 60, got %d", record.Mapq)
	}

	// Test case 2: Invalid PAF record (too few fields)
	invalidFields := []string{"read1", "1000", "0"}
	_, err = NewPafRecord(invalidFields)
	if err != ErrInvalidPafRecord {
		t.Errorf("Expected ErrInvalidPafRecord, got %v", err)
	}

	// Test case 3: Invalid PAF record (invalid integer)
	invalidFields = []string{
		"read1", "not_an_int", "0", "1000", "+", "ref1", "2000", "500", "1500", "950", "1000", "60",
	}
	_, err = NewPafRecord(invalidFields)
	if err == nil {
		t.Errorf("Expected error for invalid integer, got nil")
	}
}

func TestCalculateScores(t *testing.T) {
	// Create test PAF records
	records := []PafRecord{
		{
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
		},
		{
			Qname:       "read1",
			Qlength:     1000,
			Qstart:      0,
			Qend:        1000,
			Strand:      "+",
			Tname:       "ref2",
			Tlength:     2000,
			Tstart:      500,
			Tend:        1500,
			NumMatches:  900,
			AlignLength: 1000,
			Mapq:        60,
		},
	}

	// Create test mapping database
	db := &MappingDatabase{
		Entries: map[string]MappingEntry{
			"ref1": {
				Accession: "ref1",
				Serotype:  "H1N1",
				Segment:   1,
			},
			"ref2": {
				Accession: "ref2",
				Serotype:  "H3N2",
				Segment:   1,
			},
		},
	}

	// Calculate scores
	scores, err := CalculateScores(records, db)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check scores
	if len(scores) != 2 {
		t.Errorf("Expected 2 scores, got %d", len(scores))
	}

	// Check first score
	if scores[0].Qname != "read1" {
		t.Errorf("Expected Qname to be 'read1', got '%s'", scores[0].Qname)
	}
	if scores[0].Tname != "ref1" {
		t.Errorf("Expected Tname to be 'ref1', got '%s'", scores[0].Tname)
	}
	if scores[0].Serotype != "H1N1" {
		t.Errorf("Expected Serotype to be 'H1N1', got '%s'", scores[0].Serotype)
	}
	if scores[0].ANI != 0.95 {
		t.Errorf("Expected ANI to be 0.95, got %f", scores[0].ANI)
	}
	if scores[0].AF != 1.0 {
		t.Errorf("Expected AF to be 1.0, got %f", scores[0].AF)
	}
	if scores[0].AlignScore != 0.95 {
		t.Errorf("Expected AlignScore to be 0.95, got %f", scores[0].AlignScore)
	}

	// Check second score
	if scores[1].Qname != "read1" {
		t.Errorf("Expected Qname to be 'read1', got '%s'", scores[1].Qname)
	}
	if scores[1].Tname != "ref2" {
		t.Errorf("Expected Tname to be 'ref2', got '%s'", scores[1].Tname)
	}
	if scores[1].Serotype != "H3N2" {
		t.Errorf("Expected Serotype to be 'H3N2', got '%s'", scores[1].Serotype)
	}
	if scores[1].ANI != 0.9 {
		t.Errorf("Expected ANI to be 0.9, got %f", scores[1].ANI)
	}
	if scores[1].AF != 1.0 {
		t.Errorf("Expected AF to be 1.0, got %f", scores[1].AF)
	}
	if scores[1].AlignScore != 0.9 {
		t.Errorf("Expected AlignScore to be 0.9, got %f", scores[1].AlignScore)
	}
}
