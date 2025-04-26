package model

import (
	"errors"
	"fmt"
	"strconv"
)

// Error definitions
var (
	ErrInvalidPafRecord = errors.New("invalid PAF record: insufficient fields")
)

// PafRecord represents a single record from a PAF file
type PafRecord struct {
	Qname       string // Query sequence name
	Qlength     int    // Query sequence length
	Qstart      int    // Query start (0-based)
	Qend        int    // Query end (0-based)
	Strand      string // Relative strand: "+" or "-"
	Tname       string // Target sequence name
	Tlength     int    // Target sequence length
	Tstart      int    // Target start on original strand (0-based)
	Tend        int    // Target end on original strand (0-based)
	NumMatches  int    // Number of matching bases in the mapping
	AlignLength int    // Alignment block length
	Mapq        int    // Mapping quality (0-255; 255 for missing)
}

// NewPafRecord creates a new PAF record from a slice of strings
// Optimized version with reduced string-to-integer conversions and error handling
func NewPafRecord(fields []string) (*PafRecord, error) {
	// Check if we have enough fields
	if len(fields) < 12 {
		return nil, ErrInvalidPafRecord
	}

	// Pre-allocate a single error variable to reduce allocations
	var err error
	var record PafRecord

	// Directly assign string fields to reduce pointer operations
	record.Qname = fields[0]
	record.Strand = fields[4]
	record.Tname = fields[5]

	// Convert integer fields in a single pass with optimized error handling
	// This reduces the number of function calls and error allocations
	record.Qlength, err = strconv.Atoi(fields[1])
	if err != nil {
		return nil, fmt.Errorf("invalid qlength: %w", err)
	}

	record.Qstart, err = strconv.Atoi(fields[2])
	if err != nil {
		return nil, fmt.Errorf("invalid qstart: %w", err)
	}

	record.Qend, err = strconv.Atoi(fields[3])
	if err != nil {
		return nil, fmt.Errorf("invalid qend: %w", err)
	}

	record.Tlength, err = strconv.Atoi(fields[6])
	if err != nil {
		return nil, fmt.Errorf("invalid tlength: %w", err)
	}

	record.Tstart, err = strconv.Atoi(fields[7])
	if err != nil {
		return nil, fmt.Errorf("invalid tstart: %w", err)
	}

	record.Tend, err = strconv.Atoi(fields[8])
	if err != nil {
		return nil, fmt.Errorf("invalid tend: %w", err)
	}

	record.NumMatches, err = strconv.Atoi(fields[9])
	if err != nil {
		return nil, fmt.Errorf("invalid num_matches: %w", err)
	}

	record.AlignLength, err = strconv.Atoi(fields[10])
	if err != nil {
		return nil, fmt.Errorf("invalid align_length: %w", err)
	}

	record.Mapq, err = strconv.Atoi(fields[11])
	if err != nil {
		return nil, fmt.Errorf("invalid mapq: %w", err)
	}

	return &record, nil
}

// parseInt converts a string to an integer (kept for backward compatibility)
func parseInt(s string) (int, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("failed to parse integer: %w", err)
	}
	return i, nil
}
