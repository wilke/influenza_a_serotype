package model

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

// MappingEntry represents a single entry from the mapping file
type MappingEntry struct {
	Accession string // Accession number
	Serotype  string // Serotype information
	Segment   int    // Segment number
	// Additional fields can be added as needed
}

// MappingDatabase represents a collection of mapping entries
type MappingDatabase struct {
	Entries map[string]MappingEntry // Map of accession to mapping entry
}

// NewMappingDatabase creates a new mapping database from a file
func NewMappingDatabase(filename string) (*MappingDatabase, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open mapping file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.Comment = '#'

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file header: %w", err)
	}

	// Find column indices
	accessionIdx := -1
	serotypeIdx := -1
	segmentIdx := -1

	for i, col := range header {
		switch col {
		case "accession":
			accessionIdx = i
		case "serotype":
			serotypeIdx = i
		case "segment":
			segmentIdx = i
		}
	}

	if accessionIdx == -1 || serotypeIdx == -1 || segmentIdx == -1 {
		return nil, fmt.Errorf("mapping file missing required columns: accession, serotype, segment")
	}

	// Read entries
	db := &MappingDatabase{
		Entries: make(map[string]MappingEntry),
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read mapping file record: %w", err)
		}

		// Try to parse segment as integer
		segment, err := parseInt(record[segmentIdx])
		if err != nil {
			// Log the error but continue with a default value
			fmt.Printf("Warning: failed to parse segment '%s': %v, using default value 0\n",
				record[segmentIdx], err)
			segment = 0
		}

		entry := MappingEntry{
			Accession: record[accessionIdx],
			Serotype:  record[serotypeIdx],
			Segment:   segment,
		}

		db.Entries[entry.Accession] = entry
	}

	return db, nil
}

// GetEntry returns a mapping entry for the given accession
func (db *MappingDatabase) GetEntry(accession string) (MappingEntry, bool) {
	entry, ok := db.Entries[accession]
	return entry, ok
}

// Size returns the number of entries in the database
func (db *MappingDatabase) Size() int {
	return len(db.Entries)
}
