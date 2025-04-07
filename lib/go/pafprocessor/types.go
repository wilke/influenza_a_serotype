package pafprocessor

// PAFEntry represents a single line in a PAF file
type PAFEntry struct {
	QName       string
	QLength     int
	QStart      int
	QEnd        int
	Strand      string
	TName       string
	TLength     int
	TStart      int
	TEnd        int
	NumMatches  int
	AlignLength int
	MapQ        int
}

// MappingEntry represents a single line in the mapping TSV file
type MappingEntry struct {
	Accession      string
	Serotype       string
	Segment        int
	OrganismName   string
	Host           string
	CollectionDate string
}

// GroupedEntry represents a PAF entry with serotype and segment information
type GroupedEntry struct {
	QName       string
	TName       string
	Serotype    string
	Segment     int
	Strand      string
	ReadLength  int
	AlignLength int
	NumMatches  int
	ANI         float64
	AF          float64
	AlignScore  float64
}

// SummaryEntry represents a summary of grouped entries
type SummaryEntry struct {
	QName          string
	Serotype       string
	Segment        int
	Strand         string // Primary strand (kept for backward compatibility)
	Count          int
	TopScore       float64
	AvgScore       float64
	ReadAssignment string
	AllStrands     []string // New field to store all strands
	AllSegments    []int    // Field to store all segments used for the read assignment
}
