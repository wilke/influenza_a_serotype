// Package models defines the core data structures for myPaf2Serotypes
package models

import (
	"time"
)

// Type aliases for better type safety and clarity
type (
	Accession      string
	Serotype       string
	Segment        string
	OrganismName   string
	Host           string
	CollectionDate string
)

// SegmentMap provides mapping from segment names to segment numbers
var SegmentMap = map[string]string{
	"PB2": "1", // Segment 1
	"PB1": "2", // Segment 2
	"PA":  "3", // Segment 3
	"HA":  "4", // Segment 4
	"NP":  "5", // Segment 5
	"NA":  "6", // Segment 6
	"M1":  "7", // Segment 7
	"MP":  "7", // Alternative name for M1
	"M2":  "7", // Alternative name for M1
	"NS":  "8", // Segment 8
	"NS1": "8", // Alternative name for NS
	"NS2": "8", // Alternative name for NS
}

// Constants for serotype assignment
const (
	UnknownSerotype   Serotype = "unknown"
	AmbiguousSerotype Serotype = "ambiguous"
	NoAssignment      Serotype = "no_assignment"
)

// IsValidSerotype checks if a serotype starts with 'H' (e.g., H1N1, H3N2)
func (s Serotype) IsValid() bool {
	return len(s) > 0 && s[0] == 'H'
}

// IsHA checks if the segment is HA (segment 4)
func (s Segment) IsHA() bool {
	return s == "4"
}

// IsNA checks if the segment is NA (segment 6)
func (s Segment) IsNA() bool {
	return s == "6"
}

// GetSegmentNumber converts segment name to number
func GetSegmentNumber(name string) (Segment, bool) {
	if num, ok := SegmentMap[name]; ok {
		return Segment(num), true
	}
	// Check if it's already a number between 1-8
	if len(name) == 1 && name[0] >= '1' && name[0] <= '8' {
		return Segment(name), true
	}
	return "", false
}

// ParseCollectionDate attempts to parse the collection date
func (cd CollectionDate) Parse() (time.Time, error) {
	// Try common date formats
	formats := []string{
		"2006-01-02",
		"2006/01/02",
		"01/02/2006",
		"02/01/2006",
		"2006",
		time.RFC3339,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, string(cd)); err == nil {
			return t, nil
		}
	}

	return time.Time{}, nil // Return zero time if parsing fails
}