package models

import (
	"fmt"
	"strconv"
	"strings"
)

// PafHit represents a single record in the PAF (Pairwise mApping Format) file
type PafHit struct {
	QueryName            string   // Field 1: Name of the query sequence
	QueryLength          int      // Field 2: Length of the query sequence
	QueryStart           int      // Field 3: Start position (0-based) on the query
	QueryEnd             int      // Field 4: End position (0-based, exclusive) on the query
	Strand               string   // Field 5: '+' or '-', indicating the query strand
	TargetName           string   // Field 6: Name of the target sequence (reference)
	TargetLength         int      // Field 7: Length of the target sequence
	TargetStart          int      // Field 8: Start position on the target
	TargetEnd            int      // Field 9: End position (exclusive) on the target
	NumResidueMatches    int      // Field 10: Number of matching bases
	AlignmentBlockLength int      // Field 11: Total length of the alignment block
	MappingQuality       int      // Field 12: Mapping quality (0–255, 255 means missing)
	Serotype             Serotype // Serotype from mapping file
}

// ParsePafLine parses a single line from a PAF file
func ParsePafLine(line string) (*PafHit, error) {
	fields := strings.Split(line, "\t")
	if len(fields) < 12 {
		return nil, fmt.Errorf("invalid PAF line: expected at least 12 fields, got %d", len(fields))
	}

	paf := &PafHit{
		QueryName:  fields[0],
		Strand:     fields[4],
		TargetName: fields[5],
	}

	// Parse numeric fields
	var err error
	if paf.QueryLength, err = strconv.Atoi(fields[1]); err != nil {
		return nil, fmt.Errorf("invalid query length: %v", err)
	}
	if paf.QueryStart, err = strconv.Atoi(fields[2]); err != nil {
		return nil, fmt.Errorf("invalid query start: %v", err)
	}
	if paf.QueryEnd, err = strconv.Atoi(fields[3]); err != nil {
		return nil, fmt.Errorf("invalid query end: %v", err)
	}
	if paf.TargetLength, err = strconv.Atoi(fields[6]); err != nil {
		return nil, fmt.Errorf("invalid target length: %v", err)
	}
	if paf.TargetStart, err = strconv.Atoi(fields[7]); err != nil {
		return nil, fmt.Errorf("invalid target start: %v", err)
	}
	if paf.TargetEnd, err = strconv.Atoi(fields[8]); err != nil {
		return nil, fmt.Errorf("invalid target end: %v", err)
	}
	if paf.NumResidueMatches, err = strconv.Atoi(fields[9]); err != nil {
		return nil, fmt.Errorf("invalid number of residue matches: %v", err)
	}
	if paf.AlignmentBlockLength, err = strconv.Atoi(fields[10]); err != nil {
		return nil, fmt.Errorf("invalid alignment block length: %v", err)
	}
	if paf.MappingQuality, err = strconv.Atoi(fields[11]); err != nil {
		return nil, fmt.Errorf("invalid mapping quality: %v", err)
	}

	return paf, nil
}

// CalculateANI calculates the Average Nucleotide Identity
func (p *PafHit) CalculateANI() float64 {
	if p.AlignmentBlockLength == 0 {
		return 0
	}
	return float64(p.NumResidueMatches) / float64(p.AlignmentBlockLength)
}

// CalculateAF calculates the Alignment Fraction (coverage)
func (p *PafHit) CalculateAF() float64 {
	if p.QueryLength == 0 {
		return 0
	}
	return float64(p.QueryEnd-p.QueryStart) / float64(p.QueryLength)
}

// CalculateScore calculates the alignment score (ANI * AF)
func (p *PafHit) CalculateScore() float64 {
	return p.CalculateANI() * p.CalculateAF()
}

// GetAlignmentLength returns the alignment length on the query
func (p *PafHit) GetAlignmentLength() int {
	return p.QueryEnd - p.QueryStart
}

// GetTargetAlignmentLength returns the alignment length on the target
func (p *PafHit) GetTargetAlignmentLength() int {
	return p.TargetEnd - p.TargetStart
}

// IsForwardStrand checks if the alignment is on the forward strand
func (p *PafHit) IsForwardStrand() bool {
	return p.Strand == "+"
}

// String returns a string representation of the PAF hit
func (p *PafHit) String() string {
	return fmt.Sprintf("%s\t%d\t%d\t%d\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d",
		p.QueryName, p.QueryLength, p.QueryStart, p.QueryEnd,
		p.Strand, p.TargetName, p.TargetLength,
		p.TargetStart, p.TargetEnd, p.NumResidueMatches,
		p.AlignmentBlockLength, p.MappingQuality)
}

// Validate checks if the PAF hit has valid values
func (p *PafHit) Validate() error {
	if p.QueryName == "" {
		return fmt.Errorf("query name is empty")
	}
	if p.TargetName == "" {
		return fmt.Errorf("target name is empty")
	}
	if p.QueryLength < 0 {
		return fmt.Errorf("query length is negative")
	}
	if p.TargetLength < 0 {
		return fmt.Errorf("target length is negative")
	}
	if p.QueryStart < 0 || p.QueryEnd < p.QueryStart || p.QueryEnd > p.QueryLength {
		return fmt.Errorf("invalid query coordinates")
	}
	if p.TargetStart < 0 || p.TargetEnd < p.TargetStart || p.TargetEnd > p.TargetLength {
		return fmt.Errorf("invalid target coordinates")
	}
	if p.Strand != "+" && p.Strand != "-" {
		return fmt.Errorf("invalid strand: %s", p.Strand)
	}
	if p.NumResidueMatches < 0 || p.NumResidueMatches > p.AlignmentBlockLength {
		return fmt.Errorf("invalid number of matches")
	}
	if p.MappingQuality < 0 || p.MappingQuality > 255 {
		return fmt.Errorf("mapping quality out of range")
	}
	return nil
}