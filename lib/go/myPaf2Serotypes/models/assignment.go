package models

import (
	"fmt"
	"sort"
	"strings"
)

// Assignment represents the final serotype assignment for a query
type Assignment struct {
	Serotype          Serotype            // Assigned serotype (or "ambiguous", "no_assignment")
	QueryName         string              // Query/read name
	Summaries         []Summary           // All summaries for this query
	AlignmentScore    float64             // Best alignment score
	AvgAlignmentScore float64             // Average alignment score
	Hits              int                 // Total number of hits
	SerotypeMap       map[Serotype]int   // Count of hits per serotype
	Reason            string              // Reason for assignment (e.g., "below threshold", "ambiguous")
}

// NewAssignment creates a new assignment from summaries
func NewAssignment(queryName string, summaries []Summary) *Assignment {
	assignment := &Assignment{
		QueryName:   queryName,
		Summaries:   summaries,
		SerotypeMap: make(map[Serotype]int),
	}

	// Calculate total hits and serotype counts
	for _, summary := range summaries {
		assignment.Hits += summary.NumberOfContributions
		assignment.SerotypeMap[summary.Serotype]++
	}

	// Find best score
	if len(summaries) > 0 {
		assignment.AlignmentScore = summaries[0].MaxAlignmentScore
		
		// Calculate average score
		var totalScore float64
		for _, summary := range summaries {
			totalScore += summary.MaxAlignmentScore
		}
		assignment.AvgAlignmentScore = totalScore / float64(len(summaries))
	}

	return assignment
}

// AssignSerotype determines the final serotype based on scores and thresholds
func (a *Assignment) AssignSerotype(minScore, scoreDistance float64) {
	if len(a.Summaries) == 0 {
		a.Serotype = NoAssignment
		a.Reason = "no summaries"
		return
	}

	// Sort summaries by score
	SortSummariesByScore(a.Summaries)

	// Check if best score meets threshold
	if a.AlignmentScore < minScore {
		a.Serotype = NoAssignment
		a.Reason = fmt.Sprintf("below threshold (%.4f < %.4f)", a.AlignmentScore, minScore)
		return
	}

	// Get top scores within score distance
	topScore := a.AlignmentScore
	var topSerotypes []Serotype
	serotypeSet := make(map[Serotype]bool)

	for _, summary := range a.Summaries {
		// Check if score is within distance of top score
		if topScore-summary.MaxAlignmentScore <= scoreDistance {
			if !serotypeSet[summary.Serotype] {
				topSerotypes = append(topSerotypes, summary.Serotype)
				serotypeSet[summary.Serotype] = true
			}
		} else {
			// Scores are sorted, so we can break here
			break
		}
	}

	// Determine assignment based on number of top serotypes
	switch len(topSerotypes) {
	case 0:
		a.Serotype = NoAssignment
		a.Reason = "no serotypes found"
	case 1:
		a.Serotype = topSerotypes[0]
		a.Reason = fmt.Sprintf("clear winner (score: %.4f)", topScore)
	default:
		a.Serotype = AmbiguousSerotype
		a.Reason = fmt.Sprintf("multiple serotypes within score distance: %s", 
			strings.Join(serotypesToStrings(topSerotypes), ", "))
	}
}

// IsAssigned checks if a serotype was successfully assigned
func (a *Assignment) IsAssigned() bool {
	return a.Serotype != NoAssignment && a.Serotype != AmbiguousSerotype && a.Serotype != ""
}

// IsAmbiguous checks if the assignment is ambiguous
func (a *Assignment) IsAmbiguous() bool {
	return a.Serotype == AmbiguousSerotype
}

// GetTopSerotypes returns the serotypes with the highest scores
func (a *Assignment) GetTopSerotypes(n int) []Serotype {
	if len(a.Summaries) == 0 || n <= 0 {
		return []Serotype{}
	}

	// Sort summaries by score
	sorted := make([]Summary, len(a.Summaries))
	copy(sorted, a.Summaries)
	SortSummariesByScore(sorted)

	// Collect unique serotypes
	serotypeSet := make(map[Serotype]bool)
	var serotypes []Serotype

	for _, summary := range sorted {
		if !serotypeSet[summary.Serotype] {
			serotypes = append(serotypes, summary.Serotype)
			serotypeSet[summary.Serotype] = true
			if len(serotypes) >= n {
				break
			}
		}
	}

	return serotypes
}

// String returns a string representation of the assignment
func (a *Assignment) String() string {
	return fmt.Sprintf("Query:%s Serotype:%s Score:%.4f Hits:%d Reason:%s",
		a.QueryName, a.Serotype, a.AlignmentScore, a.Hits, a.Reason)
}

// ToOutputLine returns a tab-delimited line for output files
func (a *Assignment) ToOutputLine() string {
	return fmt.Sprintf("%s\t%s\t%.6f\t%d\t%s",
		a.QueryName, a.Serotype, a.AlignmentScore, a.Hits, a.Reason)
}

// Helper function to convert serotypes to strings
func serotypesToStrings(serotypes []Serotype) []string {
	strings := make([]string, len(serotypes))
	for i, s := range serotypes {
		strings[i] = string(s)
	}
	return strings
}

// AssignmentSummary provides statistics about assignments
type AssignmentSummary struct {
	TotalReads       int
	AssignedReads    int
	AmbiguousReads   int
	UnassignedReads  int
	SerotypeCountsMap   map[Serotype]int
	AverageScore     float64
}

// NewAssignmentSummary creates a summary from a list of assignments
func NewAssignmentSummary(assignments []Assignment) *AssignmentSummary {
	summary := &AssignmentSummary{
		SerotypeCountsMap: make(map[Serotype]int),
	}

	var totalScore float64
	for _, assignment := range assignments {
		summary.TotalReads++
		totalScore += assignment.AlignmentScore

		switch {
		case assignment.IsAssigned():
			summary.AssignedReads++
			summary.SerotypeCountsMap[assignment.Serotype]++
		case assignment.IsAmbiguous():
			summary.AmbiguousReads++
		default:
			summary.UnassignedReads++
		}
	}

	if summary.TotalReads > 0 {
		summary.AverageScore = totalScore / float64(summary.TotalReads)
	}

	return summary
}

// GetSortedSerotypes returns serotypes sorted by count (descending)
func (s *AssignmentSummary) GetSortedSerotypes() []struct {
	Serotype Serotype
	Count    int
} {
	var sorted []struct {
		Serotype Serotype
		Count    int
	}

	for serotype, count := range s.SerotypeCountsMap {
		sorted = append(sorted, struct {
			Serotype Serotype
			Count    int
		}{serotype, count})
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Count > sorted[j].Count
	})

	return sorted
}

// String returns a string representation of the summary
func (s *AssignmentSummary) String() string {
	return fmt.Sprintf("Total:%d Assigned:%d (%.1f%%) Ambiguous:%d (%.1f%%) Unassigned:%d (%.1f%%) AvgScore:%.4f",
		s.TotalReads,
		s.AssignedReads, float64(s.AssignedReads)/float64(s.TotalReads)*100,
		s.AmbiguousReads, float64(s.AmbiguousReads)/float64(s.TotalReads)*100,
		s.UnassignedReads, float64(s.UnassignedReads)/float64(s.TotalReads)*100,
		s.AverageScore)
}