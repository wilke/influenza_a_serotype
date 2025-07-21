package models

import (
	"fmt"
	"sort"
)

// PafScore represents aggregated scores for a group of PAF hits
type PafScore struct {
	TotalReadLength       int       // Sum of read lengths
	TotalAlignmentLength  int       // Sum of alignment lengths
	TotalMatches          int       // Sum of matching bases
	ANI                   float64   // Average Nucleotide Identity
	AFI                   float64   // Alignment Fraction Index
	AlignmentScore        float64   // Combined score (ANI * AFI)
	Group                 string    // Grouping key
	Hits                  []*PafHit // List of contributing PAF hits
	Serotype              Serotype  // Assigned serotype
	QueryName             string    // Query/read name
	Segment               Segment   // Segment number
	NumberOfContributions int       // Number of hits contributing to this score
}

// NewPafScore creates a new PafScore from a group of hits
func NewPafScore(queryName string, hits []*PafHit) *PafScore {
	if len(hits) == 0 {
		return &PafScore{
			QueryName: queryName,
		}
	}

	score := &PafScore{
		QueryName:             queryName,
		Hits:                  hits,
		NumberOfContributions: len(hits),
	}

	// Use the first hit to set common properties
	if len(hits) > 0 {
		score.Serotype = hits[0].Serotype
		score.TotalReadLength = hits[0].QueryLength
	}

	// Calculate aggregated values
	for _, hit := range hits {
		score.TotalAlignmentLength += hit.GetAlignmentLength()
		score.TotalMatches += hit.NumResidueMatches
	}

	// Calculate scores
	score.calculateScores()

	return score
}

// calculateScores calculates ANI, AFI, and combined alignment score
func (s *PafScore) calculateScores() {
	if s.TotalAlignmentLength > 0 {
		s.ANI = float64(s.TotalMatches) / float64(s.TotalAlignmentLength)
	}
	
	if s.TotalReadLength > 0 {
		s.AFI = float64(s.TotalAlignmentLength) / float64(s.TotalReadLength)
	}
	
	s.AlignmentScore = s.ANI * s.AFI
}

// String returns a string representation of the score
func (s *PafScore) String() string {
	return fmt.Sprintf("Query:%s Serotype:%s Score:%.4f ANI:%.4f AFI:%.4f Hits:%d",
		s.QueryName, s.Serotype, s.AlignmentScore, s.ANI, s.AFI, s.NumberOfContributions)
}

// Summary represents a summary of scores for a query
type Summary struct {
	Scores                []PafScore
	MaxAlignmentScore     float64
	AvgAlignmentScore     float64
	Serotype              Serotype
	Segment               Segment
	QueryName             string
	NumberOfContributions int
	GroupedBy             string
}

// NewSummary creates a summary from a list of scores
func NewSummary(queryName string, scores []PafScore) *Summary {
	if len(scores) == 0 {
		return &Summary{
			QueryName: queryName,
			Scores:    scores,
		}
	}

	summary := &Summary{
		QueryName: queryName,
		Scores:    scores,
	}

	// Calculate max and average scores
	var totalScore float64
	for i, score := range scores {
		if i == 0 || score.AlignmentScore > summary.MaxAlignmentScore {
			summary.MaxAlignmentScore = score.AlignmentScore
			summary.Serotype = score.Serotype
			summary.Segment = score.Segment
		}
		totalScore += score.AlignmentScore
		summary.NumberOfContributions += score.NumberOfContributions
	}

	if len(scores) > 0 {
		summary.AvgAlignmentScore = totalScore / float64(len(scores))
	}

	return summary
}

// GetTopNScores returns the top N unique scores from summaries
func GetTopNScores(summaries []Summary, n int) []float64 {
	if len(summaries) == 0 || n <= 0 {
		return []float64{}
	}

	// Collect all unique scores
	scoreMap := make(map[float64]bool)
	for _, summary := range summaries {
		scoreMap[summary.MaxAlignmentScore] = true
	}

	// Convert to slice
	scores := make([]float64, 0, len(scoreMap))
	for score := range scoreMap {
		scores = append(scores, score)
	}

	// Sort in descending order
	sort.Float64s(scores)
	for i, j := 0, len(scores)-1; i < j; i, j = i+1, j-1 {
		scores[i], scores[j] = scores[j], scores[i]
	}

	// Return top N
	if n > len(scores) {
		return scores
	}
	return scores[:n]
}

// GetSummariesByScore returns summaries that have the specified score
func GetSummariesByScore(summaries []Summary, targetScores []float64) []Summary {
	if len(targetScores) == 0 {
		return []Summary{}
	}

	// Create a map for quick lookup
	scoreMap := make(map[float64]bool)
	for _, score := range targetScores {
		scoreMap[score] = true
	}

	// Filter summaries
	var filtered []Summary
	for _, summary := range summaries {
		if scoreMap[summary.MaxAlignmentScore] {
			filtered = append(filtered, summary)
		}
	}

	return filtered
}

// SortSummariesByScore sorts summaries by their max alignment score in descending order
func SortSummariesByScore(summaries []Summary) {
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].MaxAlignmentScore > summaries[j].MaxAlignmentScore
	})
}

// GroupScoresBySerotype groups scores by serotype
func GroupScoresBySerotype(scores []PafScore) map[Serotype][]PafScore {
	groups := make(map[Serotype][]PafScore)
	for _, score := range scores {
		groups[score.Serotype] = append(groups[score.Serotype], score)
	}
	return groups
}