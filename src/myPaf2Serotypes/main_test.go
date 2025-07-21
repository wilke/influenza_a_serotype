package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// Helper function to create temporary test files
func createTempFile(t *testing.T, content string) string {
	tmpfile, err := ioutil.TempFile("", "test")
	if err != nil {
		t.Fatal(err)
	}
	
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}
	
	return tmpfile.Name()
}

// Test LoadMappingFile function
func TestLoadMappingFile(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		wantErr     bool
		expectedLen int
		validate    func(t *testing.T, m Mapping)
	}{
		{
			name: "Valid mapping file with H serotypes",
			fileContent: `CY021089	H1N1	1	Influenza A virus	Human	2009-04-30
CY021090	H1N1	2	Influenza A virus	Human	2009-04-30
CY021091	H3N2	PB2	Influenza A virus	Human	2009-05-01
CY021092	H3N2	PB1	Influenza A virus	Human	2009-05-01`,
			wantErr:     false,
			expectedLen: 4,
			validate: func(t *testing.T, m Mapping) {
				// Check first entry
				entry, ok := m[Accession("CY021089")]
				if !ok {
					t.Error("Expected CY021089 in mapping")
				}
				if entry.Serotype != "H1N1" {
					t.Errorf("Expected serotype H1N1, got %s", entry.Serotype)
				}
				if entry.Segment != "1" {
					t.Errorf("Expected segment 1, got %s", entry.Segment)
				}
				
				// Check segment name mapping
				entry3, ok := m[Accession("CY021091")]
				if !ok {
					t.Error("Expected CY021091 in mapping")
				}
				if entry3.Segment != "1" {
					t.Errorf("Expected segment 1 for PB2, got %s", entry3.Segment)
				}
			},
		},
		{
			name: "Invalid serotypes (non-H prefix)",
			fileContent: `CY021089	A1N1	1	Influenza A virus	Human	2009-04-30
CY021090	B2N2	2	Influenza A virus	Human	2009-04-30`,
			wantErr:     false,
			expectedLen: 0,
		},
		{
			name: "Empty file",
			fileContent: ``,
			wantErr:     false,
			expectedLen: 0,
		},
		{
			name: "File with empty lines",
			fileContent: `CY021089	H1N1	1	Influenza A virus	Human	2009-04-30

CY021090	H1N1	2	Influenza A virus	Human	2009-04-30
`,
			wantErr:     false,
			expectedLen: 2,
		},
		{
			name: "All records with complete fields",
			fileContent: `CY021089	H1N1	1	Influenza A virus	Human	2009-04-30
CY021090	H1N1	2	Influenza A virus	Human	2009-04-30`,
			wantErr:     false,
			expectedLen: 2,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filename := createTempFile(t, tt.fileContent)
			defer os.Remove(filename)
			
			mapping, err := LoadMappingFile(filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadMappingFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if len(mapping) != tt.expectedLen {
				t.Errorf("Expected %d entries, got %d", tt.expectedLen, len(mapping))
			}
			
			if tt.validate != nil {
				tt.validate(t, mapping)
			}
		})
	}
}

// Test GetNTopScores function
func TestGetNTopScores(t *testing.T) {
	tests := []struct {
		name      string
		summaries []Summary
		n         int
		want      []float64
	}{
		{
			name: "Get top 2 scores from 5 summaries",
			summaries: []Summary{
				{maxAlignmentScore: 0.95},
				{maxAlignmentScore: 0.90},
				{maxAlignmentScore: 0.85},
				{maxAlignmentScore: 0.95},
				{maxAlignmentScore: 0.80},
			},
			n:    2,
			want: []float64{0.95, 0.90},
		},
		{
			name: "Request more scores than available",
			summaries: []Summary{
				{maxAlignmentScore: 0.95},
				{maxAlignmentScore: 0.90},
			},
			n:    5,
			want: []float64{0.95, 0.90},
		},
		{
			name:      "Empty summaries",
			summaries: []Summary{},
			n:         2,
			want:      []float64{},
		},
		{
			name: "Duplicate scores",
			summaries: []Summary{
				{maxAlignmentScore: 0.95},
				{maxAlignmentScore: 0.95},
				{maxAlignmentScore: 0.90},
				{maxAlignmentScore: 0.90},
			},
			n:    2,
			want: []float64{0.95, 0.90},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetNTopScores(tt.summaries, tt.n)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNTopScores() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test GetSummariesByScore function
func TestGetSummariesByScore(t *testing.T) {
	summaries := []Summary{
		{maxAlignmentScore: 0.95, Serotype: "H1N1"},
		{maxAlignmentScore: 0.90, Serotype: "H3N2"},
		{maxAlignmentScore: 0.95, Serotype: "H1N1"},
		{maxAlignmentScore: 0.85, Serotype: "H5N1"},
		{maxAlignmentScore: 0.90, Serotype: "H3N2"},
	}
	
	tests := []struct {
		name               string
		scores             []float64
		wantSummariesCount int
		wantSerotypes      map[Serotype]int
	}{
		{
			name:               "Filter by top score",
			scores:             []float64{0.95},
			wantSummariesCount: 2,
			wantSerotypes:      map[Serotype]int{"H1N1": 2},
		},
		{
			name:               "Filter by multiple scores",
			scores:             []float64{0.95, 0.90},
			wantSummariesCount: 4,
			wantSerotypes:      map[Serotype]int{"H1N1": 2, "H3N2": 2},
		},
		{
			name:               "No matching scores",
			scores:             []float64{0.99},
			wantSummariesCount: 0,
			wantSerotypes:      map[Serotype]int{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSummaries, gotSerotypes := GetSummariesByScore(summaries, tt.scores)
			
			if len(gotSummaries) != tt.wantSummariesCount {
				t.Errorf("Expected %d summaries, got %d", tt.wantSummariesCount, len(gotSummaries))
			}
			
			if !reflect.DeepEqual(gotSerotypes, tt.wantSerotypes) {
				t.Errorf("GetSummariesByScore() serotypes = %v, want %v", gotSerotypes, tt.wantSerotypes)
			}
		})
	}
}

// Test CalculateScores function
func TestCalculateScores(t *testing.T) {
	// Initialize logger for the test
	logger = log.New(os.Stdout, "TEST: ", log.Ldate|log.Ltime|log.Lshortfile)
	
	// Create test mapping
	mapping := Mapping{
		"ref1": {Serotype: "H1N1", Segment: "1", OrganismName: "Influenza A", Host: "Human", CollectionDate: "2009-04-30"},
		"ref2": {Serotype: "H3N2", Segment: "2", OrganismName: "Influenza A", Host: "Human", CollectionDate: "2009-05-01"},
	}
	
	// Create a channel for test records
	records := make(chan []PafHit, 1)
	
	// Test data
	testRecords := []PafHit{
		{
			QueryName:            "read1",
			QueryLength:          100,
			QueryStart:           0,
			QueryEnd:             80,
			Strand:               "+",
			TargetName:           "ref1",
			TargetLength:         1000,
			TargetStart:          100,
			TargetEnd:            180,
			NumResidueMatches:    75,
			AlignmentBlockLength: 80,
			MappingQuality:       60,
		},
		{
			QueryName:            "read1",
			QueryLength:          100,
			QueryStart:           0,
			QueryEnd:             70,
			Strand:               "+",
			TargetName:           "ref2",
			TargetLength:         1000,
			TargetStart:          200,
			TargetEnd:            270,
			NumResidueMatches:    65,
			AlignmentBlockLength: 70,
			MappingQuality:       60,
		},
	}
	
	// Send test records
	records <- testRecords
	close(records)
	
	// Calculate scores
	scoresChan := CalculateScores(records, mapping)
	
	// Collect all scores
	var allScores []PafScore
	for scores := range scoresChan {
		allScores = append(allScores, scores...)
	}
	
	// Verify results
	if len(allScores) != 2 {
		t.Errorf("Expected 2 scores, got %d", len(allScores))
	}
	
	// Check first score (H1N1)
	found := false
	for _, score := range allScores {
		if score.Serotype == "H1N1" {
			found = true
			expectedANI := 75.0 / 80.0
			expectedAF := 80.0 / 100.0
			expectedScore := expectedANI * expectedAF
			
			if score.ANI != expectedANI {
				t.Errorf("Expected ANI %f, got %f", expectedANI, score.ANI)
			}
			if score.AFI != expectedAF {
				t.Errorf("Expected AFI %f, got %f", expectedAF, score.AFI)
			}
			if score.AlignmentScore != expectedScore {
				t.Errorf("Expected alignment score %f, got %f", expectedScore, score.AlignmentScore)
			}
		}
	}
	
	if !found {
		t.Error("H1N1 score not found")
	}
}

// Test AssignSerotype function
func TestAssignSerotype(t *testing.T) {
	mapping := Mapping{
		"ref1": {Serotype: "H1N1", Segment: "1", OrganismName: "Influenza A", Host: "Human", CollectionDate: "2009-04-30"},
		"ref2": {Serotype: "H3N2", Segment: "1", OrganismName: "Influenza A", Host: "Human", CollectionDate: "2009-05-01"},
		"ref3": {Serotype: "H5N1", Segment: "1", OrganismName: "Influenza A", Host: "Human", CollectionDate: "2009-05-01"},
	}
	
	tests := []struct {
		name              string
		scores            []PafScore
		topScoreThreshold float64
		scoreDistance     float64
		wantAssignment    string
	}{
		{
			name: "Clear winner - single serotype",
			scores: []PafScore{
				{
					QueryName:      "read1",
					Serotype:       "H1N1",
					Segment:        "1",
					AlignmentScore: 0.95,
					Hits:           []*PafHit{{TargetName: "ref1"}},
				},
			},
			topScoreThreshold: 0.9,
			scoreDistance:     0.003,
			wantAssignment:    "H1N1",
		},
		{
			name: "Multiple serotypes - clear winner",
			scores: []PafScore{
				{
					QueryName:      "read1",
					Serotype:       "H1N1",
					Segment:        "1",
					AlignmentScore: 0.95,
					Hits:           []*PafHit{{TargetName: "ref1"}},
				},
				{
					QueryName:      "read1",
					Serotype:       "H3N2",
					Segment:        "1",
					AlignmentScore: 0.90,
					Hits:           []*PafHit{{TargetName: "ref2"}},
				},
			},
			topScoreThreshold: 0.9,
			scoreDistance:     0.003,
			wantAssignment:    "H1N1",
		},
		{
			name: "Ambiguous - scores too close",
			scores: []PafScore{
				{
					QueryName:      "read1",
					Serotype:       "H1N1",
					Segment:        "1",
					AlignmentScore: 0.95,
					Hits:           []*PafHit{{TargetName: "ref1"}},
				},
				{
					QueryName:      "read1",
					Serotype:       "H3N2",
					Segment:        "1",
					AlignmentScore: 0.948,
					Hits:           []*PafHit{{TargetName: "ref2"}},
				},
			},
			topScoreThreshold: 0.9,
			scoreDistance:     0.003,
			wantAssignment:    "ambiguous",
		},
		{
			name: "Below threshold - no assignment",
			scores: []PafScore{
				{
					QueryName:      "read1",
					Serotype:       "H1N1",
					Segment:        "1",
					AlignmentScore: 0.85,
					Hits:           []*PafHit{{TargetName: "ref1"}},
				},
			},
			topScoreThreshold: 0.9,
			scoreDistance:     0.003,
			wantAssignment:    "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create scores channel
			scoresChan := make(chan []PafScore, 1)
			scoresChan <- tt.scores
			close(scoresChan)
			
			// Run assignment
			assignments := AssignSerotype(scoresChan, mapping, tt.topScoreThreshold, tt.scoreDistance)
			
			// Collect assignments
			var gotAssignment string
			for assignment := range assignments {
				gotAssignment = string(assignment.Serotype)
			}
			
			if gotAssignment != tt.wantAssignment {
				t.Errorf("AssignSerotype() = %v, want %v", gotAssignment, tt.wantAssignment)
			}
		})
	}
}

// Test StreamPaf2Record function
func TestStreamPaf2Record(t *testing.T) {
	// Create test PAF content
	pafContent := `read1	100	0	80	+	ref1	1000	100	180	75	80	60
read1	100	0	70	+	ref2	1000	200	270	65	70	60
read2	150	0	120	+	ref1	1000	300	420	110	120	60
read2	150	0	100	-	ref3	1000	400	500	90	100	60
invalid_line_with_too_few_fields
read3	200	0	180	+	ref1	1000	500	680	170	180	60
`
	
	// Create temp file
	tmpfile := createTempFile(t, pafContent)
	defer os.Remove(tmpfile)
	
	// Stream records
	errorChan, recordsChan := StreamPaf2Record(tmpfile)
	
	// Collect records
	var allRecords [][]PafHit
	go func() {
		for err := range errorChan {
			t.Errorf("Unexpected error: %s", err)
		}
	}()
	
	for records := range recordsChan {
		allRecords = append(allRecords, records)
	}
	
	// Verify results
	if len(allRecords) != 3 {
		t.Errorf("Expected 3 record groups, got %d", len(allRecords))
	}
	
	// Check first group (read1)
	if len(allRecords[0]) != 2 {
		t.Errorf("Expected 2 hits for read1, got %d", len(allRecords[0]))
	}
	if allRecords[0][0].QueryName != "read1" {
		t.Errorf("Expected query name 'read1', got '%s'", allRecords[0][0].QueryName)
	}
	
	// Check second group (read2)
	if len(allRecords[1]) != 2 {
		t.Errorf("Expected 2 hits for read2, got %d", len(allRecords[1]))
	}
	
	// Check third group (read3)
	if len(allRecords[2]) != 1 {
		t.Errorf("Expected 1 hit for read3, got %d", len(allRecords[2]))
	}
}

// Integration test for the complete pipeline
func TestIntegrationPipeline(t *testing.T) {
	// Create test PAF content
	pafContent := `read1	100	0	95	+	CY021089	1000	100	195	90	95	60
read1	100	0	90	+	CY021090	1000	200	290	85	90	60
read2	150	0	140	+	CY021089	1000	300	440	135	140	60
read2	150	0	130	+	CY021091	1000	400	530	120	130	60
read3	200	0	180	+	CY021089	1000	500	680	170	180	60
`
	
	// Create test mapping content
	mappingContent := `CY021089	H1N1	1	Influenza A virus	Human	2009-04-30
CY021090	H3N2	1	Influenza A virus	Human	2009-05-01
CY021091	H5N1	1	Influenza A virus	Human	2009-05-01
`
	
	// Create temp files
	pafFile := createTempFile(t, pafContent)
	defer os.Remove(pafFile)
	
	mappingFile := createTempFile(t, mappingContent)
	defer os.Remove(mappingFile)
	
	// Create temp output directory
	outputDir, err := ioutil.TempDir("", "test_output")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outputDir)
	
	// Load mapping
	mapping, err := LoadMappingFile(mappingFile)
	if err != nil {
		t.Fatalf("Failed to load mapping: %v", err)
	}
	
	// Process pipeline
	errorChan, recordsChan := StreamPaf2Record(pafFile)
	
	// Handle errors
	go func() {
		for err := range errorChan {
			t.Errorf("Pipeline error: %s", err)
		}
	}()
	
	// Calculate scores
	scoresChan := CalculateScores(recordsChan, mapping)
	
	// Assign serotypes
	assignmentsChan := AssignSerotype(scoresChan, mapping, 0.9, 0.003)
	
	// Collect assignments
	var assignments []Assignment
	for assignment := range assignmentsChan {
		assignments = append(assignments, assignment)
	}
	
	// Verify results
	if len(assignments) < 1 {
		t.Error("Expected at least 1 assignment")
	}
	
	// Check that we have assignments for our reads
	foundReads := make(map[string]bool)
	for _, assignment := range assignments {
		foundReads[assignment.QueryName] = true
		t.Logf("Assignment: %s -> %s (score: %f)", assignment.QueryName, assignment.Serotype, assignment.AlignmentScore)
	}
	
	// We expect assignments for reads that meet the threshold
	if len(foundReads) == 0 {
		t.Error("No reads were assigned")
	}
}

// Benchmark tests
func BenchmarkCalculateScores(b *testing.B) {
	// Create test mapping
	mapping := Mapping{
		"ref1": {Serotype: "H1N1", Segment: "1", OrganismName: "Influenza A", Host: "Human", CollectionDate: "2009-04-30"},
		"ref2": {Serotype: "H3N2", Segment: "2", OrganismName: "Influenza A", Host: "Human", CollectionDate: "2009-05-01"},
	}
	
	// Create test records
	testRecords := []PafHit{
		{QueryName: "read1", QueryLength: 100, NumResidueMatches: 75, AlignmentBlockLength: 80, TargetName: "ref1", Strand: "+"},
		{QueryName: "read1", QueryLength: 100, NumResidueMatches: 65, AlignmentBlockLength: 70, TargetName: "ref2", Strand: "+"},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		records := make(chan []PafHit, 1)
		records <- testRecords
		close(records)
		
		scoresChan := CalculateScores(records, mapping)
		for range scoresChan {
			// Consume scores
		}
	}
}

// Test export functions
func TestExportAssignmentsToFile(t *testing.T) {
	// Create temp output directory
	outputDir, err := ioutil.TempDir("", "test_export")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outputDir)
	
	// Create test assignments
	assignments := make(chan Assignment, 2)
	assignments <- Assignment{
		QueryName:      "read1",
		Serotype:       "H1N1",
		AlignmentScore: 0.95,
		AvgAlignmentScore: 0.93,
		Hits:          10,
	}
	assignments <- Assignment{
		QueryName:      "read2",
		Serotype:       "H3N2",
		AlignmentScore: 0.92,
		AvgAlignmentScore: 0.90,
		Hits:          8,
	}
	close(assignments)
	
	// Export assignments
	var wg sync.WaitGroup
	wg.Add(1)
	exportAssignmentsToFile(assignments, outputDir, &wg)
	wg.Wait()
	
	// Check output files
	h1n1File := filepath.Join(outputDir, "H1N1.serotype.txt")
	h3n2File := filepath.Join(outputDir, "H3N2.serotype.txt")
	
	// Verify H1N1 file
	content, err := ioutil.ReadFile(h1n1File)
	if err != nil {
		t.Errorf("Failed to read H1N1 file: %v", err)
	}
	if !strings.Contains(string(content), "read1") {
		t.Error("H1N1 file should contain read1")
	}
	
	// Verify H3N2 file
	content, err = ioutil.ReadFile(h3n2File)
	if err != nil {
		t.Errorf("Failed to read H3N2 file: %v", err)
	}
	if !strings.Contains(string(content), "read2") {
		t.Error("H3N2 file should contain read2")
	}
}

// Helper function to compare float slices
func floatSlicesEqual(a, b []float64, epsilon float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if diff := a[i] - b[i]; diff < -epsilon || diff > epsilon {
			return false
		}
	}
	return true
}

// Test for edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("Empty PAF file", func(t *testing.T) {
		tmpfile := createTempFile(t, "")
		defer os.Remove(tmpfile)
		
		errorChan, recordsChan := StreamPaf2Record(tmpfile)
		
		var recordCount int
		for range recordsChan {
			recordCount++
		}
		
		// Check for errors
		for range errorChan {
			// Should not have errors for empty file
		}
		
		if recordCount != 0 {
			t.Errorf("Expected 0 records from empty file, got %d", recordCount)
		}
	})
	
	t.Run("PAF file with only invalid lines", func(t *testing.T) {
		content := `invalid line 1
invalid line 2
short line`
		tmpfile := createTempFile(t, content)
		defer os.Remove(tmpfile)
		
		errorChan, recordsChan := StreamPaf2Record(tmpfile)
		
		var recordCount int
		for range recordsChan {
			recordCount++
		}
		
		for range errorChan {
			// Consume any errors
		}
		
		if recordCount != 0 {
			t.Errorf("Expected 0 records from invalid file, got %d", recordCount)
		}
	})
}