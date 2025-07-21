package models

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// MappingEntry represents a single entry in the mapping file
type MappingEntry struct {
	Serotype       Serotype
	Segment        Segment
	OrganismName   OrganismName
	Host           Host
	CollectionDate CollectionDate
}

// Mapping is a map from Accession to MappingEntry
type Mapping map[Accession]MappingEntry

// LoadMappingFile loads the mapping file and returns a Mapping
func LoadMappingFile(filename string) (Mapping, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open mapping file: %w", err)
	}
	defer file.Close()

	mapping := make(Mapping)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines
		if line == "" {
			continue
		}

		// Split by tab
		fields := strings.Split(line, "\t")
		if len(fields) < 6 {
			// Skip lines with insufficient fields
			continue
		}

		// Extract fields
		accession := Accession(fields[0])
		serotype := Serotype(fields[1])
		segmentStr := fields[2]
		organism := OrganismName(fields[3])
		host := Host(fields[4])
		date := CollectionDate(fields[5])

		// Validate serotype (must start with H)
		if !serotype.IsValid() {
			continue
		}

		// Convert segment name to number if needed
		segment, ok := GetSegmentNumber(segmentStr)
		if !ok {
			// If not a valid segment, skip this entry
			continue
		}

		// Create mapping entry
		mapping[accession] = MappingEntry{
			Serotype:       serotype,
			Segment:        segment,
			OrganismName:   organism,
			Host:           host,
			CollectionDate: date,
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading mapping file: %w", err)
	}

	if len(mapping) == 0 {
		return nil, fmt.Errorf("no valid entries found in mapping file")
	}

	return mapping, nil
}

// GetSerotype returns the serotype for a given accession
func (m Mapping) GetSerotype(accession Accession) (Serotype, bool) {
	entry, ok := m[accession]
	if !ok {
		return "", false
	}
	return entry.Serotype, true
}

// GetEntry returns the full mapping entry for a given accession
func (m Mapping) GetEntry(accession Accession) (MappingEntry, bool) {
	entry, ok := m[accession]
	return entry, ok
}

// GetSerotypes returns all unique serotypes in the mapping
func (m Mapping) GetSerotypes() []Serotype {
	serotypeMap := make(map[Serotype]bool)
	for _, entry := range m {
		serotypeMap[entry.Serotype] = true
	}

	serotypes := make([]Serotype, 0, len(serotypeMap))
	for serotype := range serotypeMap {
		serotypes = append(serotypes, serotype)
	}
	return serotypes
}

// CountBySerotype returns a map of serotype to count
func (m Mapping) CountBySerotype() map[Serotype]int {
	counts := make(map[Serotype]int)
	for _, entry := range m {
		counts[entry.Serotype]++
	}
	return counts
}

// FilterBySerotype returns all accessions for a given serotype
func (m Mapping) FilterBySerotype(serotype Serotype) []Accession {
	var accessions []Accession
	for acc, entry := range m {
		if entry.Serotype == serotype {
			accessions = append(accessions, acc)
		}
	}
	return accessions
}

// FilterBySegment returns all accessions for a given segment
func (m Mapping) FilterBySegment(segment Segment) []Accession {
	var accessions []Accession
	for acc, entry := range m {
		if entry.Segment == segment {
			accessions = append(accessions, acc)
		}
	}
	return accessions
}

// String returns a string representation of the mapping entry
func (e MappingEntry) String() string {
	return fmt.Sprintf("Serotype:%s Segment:%s Organism:%s Host:%s Date:%s",
		e.Serotype, e.Segment, e.OrganismName, e.Host, e.CollectionDate)
}