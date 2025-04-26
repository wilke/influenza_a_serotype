package model

import (
	"sort"
	"strings"
	"testing"
)

// assignSerotypesForTest is a test-specific implementation of the serotype assignment algorithm
// that follows the R script logic more closely
func assignSerotypesForTest(scores []AlignmentScore, scoreThresh, ambiguityThresh float64) map[string]string {
	// Group scores by read name and serotype
	type readSerotype struct {
		read     string
		serotype string
	}

	// Store top scores for each read-serotype combination
	topScores := make(map[readSerotype]float64)

	// Extract read names from Qname field
	for _, score := range scores {
		// Get the base read name (before any pipe character)
		readName := score.Qname
		if idx := strings.Index(readName, "|"); idx != -1 {
			readName = readName[:idx]
		}

		key := readSerotype{read: readName, serotype: score.Serotype}

		// Keep the highest score for each read-serotype combination
		if currentTop, exists := topScores[key]; !exists || score.AlignScore > currentTop {
			topScores[key] = score.AlignScore
		}
	}

	// Group by read name
	readToSerotypes := make(map[string]map[string]float64)
	for key, score := range topScores {
		if _, exists := readToSerotypes[key.read]; !exists {
			readToSerotypes[key.read] = make(map[string]float64)
		}
		readToSerotypes[key.read][key.serotype] = score
	}

	// Determine serotype assignment for each read
	assignments := make(map[string]string)

	for read, serotypes := range readToSerotypes {
		// Convert to slice for sorting
		type serotypeScore struct {
			serotype string
			score    float64
		}

		var scoresList []serotypeScore
		for serotype, score := range serotypes {
			// Only consider scores above threshold
			if score >= scoreThresh {
				scoresList = append(scoresList, serotypeScore{serotype: serotype, score: score})
			}
		}

		// Skip if no scores above threshold
		if len(scoresList) == 0 {
			continue
		}

		// Sort by score (descending)
		sort.Slice(scoresList, func(i, j int) bool {
			return scoresList[i].score > scoresList[j].score
		})

		// If only one serotype, assign it
		if len(scoresList) == 1 {
			assignments[read] = scoresList[0].serotype
			continue
		}

		// Check if top two scores are from the same serotype
		if scoresList[0].serotype == scoresList[1].serotype {
			assignments[read] = scoresList[0].serotype
			continue
		}

		// Check if difference between top two scores is greater than threshold
		if (scoresList[0].score - ambiguityThresh) >= scoresList[1].score {
			assignments[read] = scoresList[0].serotype
		} else {
			assignments[read] = "ambiguous"
		}
	}

	return assignments
}

// TestSerotypeDetermination tests the core logic of serotype determination
// This test focuses on the key functionality of the AssignSerotypes function
// without relying on the specific implementation details
func TestSerotypeDetermination(t *testing.T) {
	testCases := []struct {
		name            string
		scores          []AlignmentScore
		scoreThresh     float64
		ambiguityThresh float64
		expectedReads   map[string]string // map of read name to expected serotype assignment
	}{
		{
			name: "Clear winner",
			scores: []AlignmentScore{
				{
					Qname:      "read1",
					Tname:      "ref1",
					Serotype:   "H1N1",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.95,
				},
				{
					Qname:      "read1",
					Tname:      "ref2",
					Serotype:   "H3N2",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.85, // Difference > 0.003
				},
			},
			scoreThresh:     0.8,
			ambiguityThresh: 0.003,
			expectedReads: map[string]string{
				"read1": "H1N1", // Should be assigned to the top serotype
			},
		},
		{
			name: "Ambiguous assignment",
			scores: []AlignmentScore{
				{
					Qname:      "read2",
					Tname:      "ref1",
					Serotype:   "H1N1",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.95,
				},
				{
					Qname:      "read2",
					Tname:      "ref2",
					Serotype:   "H3N2",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.949, // Within 0.003 threshold
				},
			},
			scoreThresh:     0.8,
			ambiguityThresh: 0.003,
			expectedReads: map[string]string{
				"read2": "ambiguous", // Should be ambiguous
			},
		},
		{
			name: "Single serotype",
			scores: []AlignmentScore{
				{
					Qname:      "read3",
					Tname:      "ref1",
					Serotype:   "H1N1",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.95,
				},
			},
			scoreThresh:     0.8,
			ambiguityThresh: 0.003,
			expectedReads: map[string]string{
				"read3": "H1N1", // Only one serotype, should be assigned to it
			},
		},
		{
			name: "Below score threshold",
			scores: []AlignmentScore{
				{
					Qname:      "read4",
					Tname:      "ref1",
					Serotype:   "H1N1",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.75, // Below threshold
				},
			},
			scoreThresh:     0.8,
			ambiguityThresh: 0.003,
			expectedReads:   map[string]string{}, // Should be filtered out
		},
		{
			name: "Same serotype different segments",
			scores: []AlignmentScore{
				{
					Qname:      "read5",
					Tname:      "ref1",
					Serotype:   "H1N1",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.95,
				},
				{
					Qname:      "read5",
					Tname:      "ref2",
					Serotype:   "H1N1",
					Segment:    2,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.90,
				},
			},
			scoreThresh:     0.8,
			ambiguityThresh: 0.003,
			expectedReads: map[string]string{
				"read5": "H1N1", // Same serotype, should be assigned to it
			},
		},
		{
			name: "Multiple reads with different assignments",
			scores: []AlignmentScore{
				// Read 6 - Clear winner
				{
					Qname:      "read6",
					Tname:      "ref1",
					Serotype:   "H1N1",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.95,
				},
				{
					Qname:      "read6",
					Tname:      "ref2",
					Serotype:   "H3N2",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.85,
				},
				// Read 7 - Ambiguous
				{
					Qname:      "read7",
					Tname:      "ref3",
					Serotype:   "H5N1",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.92,
				},
				{
					Qname:      "read7",
					Tname:      "ref4",
					Serotype:   "H7N9",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.919, // Within threshold
				},
			},
			scoreThresh:     0.8,
			ambiguityThresh: 0.003,
			expectedReads: map[string]string{
				"read6": "H1N1",      // Clear winner
				"read7": "ambiguous", // Ambiguous
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function under test
			result := AssignSerotypes(tc.scores, tc.scoreThresh, tc.ambiguityThresh)

			// Debug output
			t.Logf("Test case: %s", tc.name)
			t.Logf("Input scores: %+v", tc.scores)
			t.Logf("Result: %+v", result)

			// Create a map of read names to their assignments for easier checking
			readAssignments := make(map[string]string)
			for _, summary := range result {
				// Extract the read name from the Qname field
				// The Qname might be in the format "read1|ref1|H1N1|1|+"
				qname := summary.Qname
				t.Logf("Processing summary: %+v, qname before: %s", summary, qname)

				// If the Qname contains a pipe, extract the first part
				if idx := strings.Index(qname, "|"); idx != -1 {
					qname = qname[:idx]
					t.Logf("Qname contains pipe, extracted: %s", qname)
				}

				// Store the assignment for this read
				readAssignments[qname] = summary.ReadAssignment
				t.Logf("Stored assignment for %s: %s", qname, summary.ReadAssignment)
			}

			t.Logf("Final readAssignments map: %v", readAssignments)

			// Check that we have the expected number of reads
			if len(readAssignments) != len(tc.expectedReads) {
				t.Errorf("Expected %d read assignments, got %d",
					len(tc.expectedReads), len(readAssignments))
			}

			// Check each expected read assignment
			for read, expectedAssignment := range tc.expectedReads {
				actualAssignment, ok := readAssignments[read]
				if !ok {
					t.Errorf("Expected read %s to be assigned, but it was not found in results", read)
					continue
				}
				if actualAssignment != expectedAssignment {
					t.Errorf("Expected read %s to be assigned to %s, got %s",
						read, expectedAssignment, actualAssignment)
				}
			}
		})
	}
}

// Helper function to find the index of a character in a string
func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// TestSerotypeDeterminationWithCustomImpl tests the serotype assignment logic
// using our custom implementation that follows the R script more closely
func TestSerotypeDeterminationWithCustomImpl(t *testing.T) {
	testCases := []struct {
		name            string
		scores          []AlignmentScore
		scoreThresh     float64
		ambiguityThresh float64
		expectedReads   map[string]string // map of read name to expected serotype assignment
	}{
		{
			name: "Clear winner",
			scores: []AlignmentScore{
				{
					Qname:      "read1",
					Tname:      "ref1",
					Serotype:   "H1N1",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.95,
				},
				{
					Qname:      "read1",
					Tname:      "ref2",
					Serotype:   "H3N2",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.85, // Difference > 0.003
				},
			},
			scoreThresh:     0.8,
			ambiguityThresh: 0.003,
			expectedReads: map[string]string{
				"read1": "H1N1", // Should be assigned to the top serotype
			},
		},
		{
			name: "Ambiguous assignment",
			scores: []AlignmentScore{
				{
					Qname:      "read2",
					Tname:      "ref1",
					Serotype:   "H1N1",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.95,
				},
				{
					Qname:      "read2",
					Tname:      "ref2",
					Serotype:   "H3N2",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.949, // Within 0.003 threshold
				},
			},
			scoreThresh:     0.8,
			ambiguityThresh: 0.003,
			expectedReads: map[string]string{
				"read2": "ambiguous", // Should be ambiguous
			},
		},
		{
			name: "Single serotype",
			scores: []AlignmentScore{
				{
					Qname:      "read3",
					Tname:      "ref1",
					Serotype:   "H1N1",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.95,
				},
			},
			scoreThresh:     0.8,
			ambiguityThresh: 0.003,
			expectedReads: map[string]string{
				"read3": "H1N1", // Only one serotype, should be assigned to it
			},
		},
		{
			name: "Below score threshold",
			scores: []AlignmentScore{
				{
					Qname:      "read4",
					Tname:      "ref1",
					Serotype:   "H1N1",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.75, // Below threshold
				},
			},
			scoreThresh:     0.8,
			ambiguityThresh: 0.003,
			expectedReads:   map[string]string{}, // Should be filtered out
		},
		{
			name: "Same serotype different segments",
			scores: []AlignmentScore{
				{
					Qname:      "read5",
					Tname:      "ref1",
					Serotype:   "H1N1",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.95,
				},
				{
					Qname:      "read5",
					Tname:      "ref2",
					Serotype:   "H1N1",
					Segment:    2,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.90,
				},
			},
			scoreThresh:     0.8,
			ambiguityThresh: 0.003,
			expectedReads: map[string]string{
				"read5": "H1N1", // Same serotype, should be assigned to it
			},
		},
		{
			name: "Multiple reads with different assignments",
			scores: []AlignmentScore{
				// Read 6 - Clear winner
				{
					Qname:      "read6",
					Tname:      "ref1",
					Serotype:   "H1N1",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.95,
				},
				{
					Qname:      "read6",
					Tname:      "ref2",
					Serotype:   "H3N2",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.85,
				},
				// Read 7 - Ambiguous
				{
					Qname:      "read7",
					Tname:      "ref3",
					Serotype:   "H5N1",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.92,
				},
				{
					Qname:      "read7",
					Tname:      "ref4",
					Serotype:   "H7N9",
					Segment:    1,
					Strand:     "+",
					ReadLength: 1000,
					AlignScore: 0.919, // Within threshold
				},
			},
			scoreThresh:     0.8,
			ambiguityThresh: 0.003,
			expectedReads: map[string]string{
				"read6": "H1N1",      // Clear winner
				"read7": "ambiguous", // Ambiguous
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Call our custom implementation
			assignments := assignSerotypesForTest(tc.scores, tc.scoreThresh, tc.ambiguityThresh)

			// Check that we have the expected number of reads
			if len(assignments) != len(tc.expectedReads) {
				t.Errorf("Expected %d read assignments, got %d",
					len(tc.expectedReads), len(assignments))
			}

			// Check each expected read assignment
			for read, expectedAssignment := range tc.expectedReads {
				actualAssignment, ok := assignments[read]
				if !ok {
					t.Errorf("Expected read %s to be assigned, but it was not found in results", read)
					continue
				}
				if actualAssignment != expectedAssignment {
					t.Errorf("Expected read %s to be assigned to %s, got %s",
						read, expectedAssignment, actualAssignment)
				}
			}
		})
	}
}
