package pafprocessor

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// GenerateResults generates the final results from the processed summaries
func GenerateResults(summaries []SummaryEntry, outputDir, sampleName string) error {
	startTime := time.Now()
	defer func() {
		LogPerformance("GenerateResults", startTime)
	}()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Group summaries by QName
	groupedByQName := make(map[string][]SummaryEntry)
	for _, summary := range summaries {
		groupedByQName[summary.QName] = append(groupedByQName[summary.QName], summary)
	}

	Logger.Infof("Grouped %d summaries into %d QName groups", len(summaries), len(groupedByQName))

	// Create final summaries with read assignments
	var finalSummaries []SummaryEntry
	for _, entries := range groupedByQName {
		// Sort entries by TopScore in descending order
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].TopScore > entries[j].TopScore
		})

		// Check if all serotypes are the same
		allSameSerotype := true
		serotype := entries[0].Serotype
		for _, entry := range entries {
			if entry.Serotype != serotype {
				allSameSerotype = false
				break
			}
		}

		// Create a list of unique segments
		segmentMap := make(map[int]bool)
		for _, entry := range entries {
			segmentMap[entry.Segment] = true
		}
		var segments []int
		for segment := range segmentMap {
			segments = append(segments, segment)
		}
		sort.Ints(segments)

		// Create a summary entry for this QName
		summary := entries[0]
		if allSameSerotype {
			summary.ReadAssignment = serotype
		} else {
			summary.ReadAssignment = "ambiguous"
		}

		// Add the list of segments to the summary
		summary.AllSegments = segments

		// Collect all unique strands across all entries for this QName
		strandMap := make(map[string]bool)

		// First add any strands that might already be in the AllStrands field
		for _, strand := range summary.AllStrands {
			strandMap[strand] = true
		}

		// Then add strands from all entries
		for _, entry := range entries {
			strandMap[entry.Strand] = true
			// Also add any strands from the AllStrands field of other entries
			for _, strand := range entry.AllStrands {
				strandMap[strand] = true
			}
		}

		// Convert map keys to slice
		allStrands := make([]string, 0, len(strandMap))
		for strand := range strandMap {
			allStrands = append(allStrands, strand)
		}
		sort.Strings(allStrands) // Sort for consistent output

		Logger.Debugf("QName %s: Final all strands: %v", summary.QName, allStrands)
		summary.AllStrands = allStrands

		finalSummaries = append(finalSummaries, summary)
	}

	// Write summary file
	summaryFile := filepath.Join(outputDir, fmt.Sprintf("%s_read_summary.tsv", sampleName))
	if err := writeSummaryFile(finalSummaries, summaryFile); err != nil {
		return fmt.Errorf("failed to write summary file: %w", err)
	}

	// Write individual read assignment files
	if err := writeReadAssignmentFiles(finalSummaries, outputDir, sampleName); err != nil {
		return fmt.Errorf("failed to write read assignment files: %w", err)
	}

	// Write performance report
	perfFile := filepath.Join(outputDir, fmt.Sprintf("%s_performance.tsv", sampleName))
	if err := writePerformanceReport(perfFile); err != nil {
		return fmt.Errorf("failed to write performance report: %w", err)
	}

	Logger.Infof("Generated results for %d QName groups", len(groupedByQName))
	return nil
}

// StreamingGenerateResults generates the final results from a channel of summaries
// This version processes summaries as they come in, reducing memory usage
func StreamingGenerateResults(summariesChan <-chan []SummaryEntry, outputDir, sampleName string) error {
	startTime := time.Now()
	defer func() {
		LogPerformance("StreamingGenerateResults", startTime)
	}()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Group summaries by QName as they come in
	groupedByQName := make(map[string][]SummaryEntry)
	totalSummaries := 0

	// Process summaries as they arrive
	for summaryBatch := range summariesChan {
		for _, summary := range summaryBatch {
			groupedByQName[summary.QName] = append(groupedByQName[summary.QName], summary)
			totalSummaries++
		}
	}

	Logger.Infof("Grouped %d summaries into %d QName groups", totalSummaries, len(groupedByQName))

	// Create final summaries with read assignments
	var finalSummaries []SummaryEntry
	for _, entries := range groupedByQName {
		// Sort entries by TopScore in descending order
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].TopScore > entries[j].TopScore
		})

		// Check if all serotypes are the same
		allSameSerotype := true
		serotype := entries[0].Serotype
		for _, entry := range entries {
			if entry.Serotype != serotype {
				allSameSerotype = false
				break
			}
		}

		// Create a list of unique segments
		segmentMap := make(map[int]bool)
		for _, entry := range entries {
			segmentMap[entry.Segment] = true
		}
		var segments []int
		for segment := range segmentMap {
			segments = append(segments, segment)
		}
		sort.Ints(segments)

		// Create a summary entry for this QName
		summary := entries[0]
		if allSameSerotype {
			summary.ReadAssignment = serotype
		} else {
			summary.ReadAssignment = "ambiguous"
		}

		// Add the list of segments to the summary
		summary.AllSegments = segments

		// Collect all unique strands across all entries for this QName
		strandMap := make(map[string]bool)

		// First add any strands that might already be in the AllStrands field
		for _, strand := range summary.AllStrands {
			strandMap[strand] = true
		}

		// Then add strands from all entries
		for _, entry := range entries {
			strandMap[entry.Strand] = true
			// Also add any strands from the AllStrands field of other entries
			for _, strand := range entry.AllStrands {
				strandMap[strand] = true
			}
		}

		// Convert map keys to slice
		allStrands := make([]string, 0, len(strandMap))
		for strand := range strandMap {
			allStrands = append(allStrands, strand)
		}
		sort.Strings(allStrands) // Sort for consistent output

		Logger.Debugf("QName %s: Final all strands: %v", summary.QName, allStrands)
		summary.AllStrands = allStrands

		finalSummaries = append(finalSummaries, summary)
	}

	// Write summary file
	summaryFile := filepath.Join(outputDir, fmt.Sprintf("%s_read_summary.tsv", sampleName))
	if err := writeSummaryFile(finalSummaries, summaryFile); err != nil {
		return fmt.Errorf("failed to write summary file: %w", err)
	}

	// Write individual read assignment files
	if err := writeReadAssignmentFiles(finalSummaries, outputDir, sampleName); err != nil {
		return fmt.Errorf("failed to write read assignment files: %w", err)
	}

	// Write performance report
	perfFile := filepath.Join(outputDir, fmt.Sprintf("%s_performance.tsv", sampleName))
	if err := writePerformanceReport(perfFile); err != nil {
		return fmt.Errorf("failed to write performance report: %w", err)
	}

	Logger.Infof("Generated results for %d QName groups", len(groupedByQName))
	return nil
}

// writeSummaryFile writes the summary file
func writeSummaryFile(summaries []SummaryEntry, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create summary file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = '\t'
	defer writer.Flush()

	// Write header
	header := []string{"qname", "serotype", "segment", "strand", "count", "top_score", "avg_score", "read_assignment", "all_strands", "all_segments"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write entries
	for _, summary := range summaries {
		// Join all strands with commas
		allStrandsStr := ""
		if len(summary.AllStrands) > 0 {
			allStrandsStr = strings.Join(summary.AllStrands, ",")
		}

		// Join all segments with commas
		allSegmentsStr := ""
		if len(summary.AllSegments) > 0 {
			// Convert each segment (int) to string
			segmentStrs := make([]string, len(summary.AllSegments))
			for i, segment := range summary.AllSegments {
				segmentStrs[i] = strconv.Itoa(segment)
			}
			allSegmentsStr = strings.Join(segmentStrs, ",")
		}

		record := []string{
			summary.QName,
			summary.Serotype,
			strconv.Itoa(summary.Segment),
			summary.Strand,
			strconv.Itoa(summary.Count),
			strconv.FormatFloat(summary.TopScore, 'f', 6, 64),
			strconv.FormatFloat(summary.AvgScore, 'f', 6, 64),
			summary.ReadAssignment,
			allStrandsStr,
			allSegmentsStr,
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	Logger.Infof("Wrote %d entries to summary file %s", len(summaries), filePath)
	return nil
}

// writeReadAssignmentFiles writes individual read assignment files
func writeReadAssignmentFiles(summaries []SummaryEntry, outputDir, sampleName string) error {
	// Group summaries by read assignment
	groupedByAssignment := make(map[string][]SummaryEntry)
	for _, summary := range summaries {
		groupedByAssignment[summary.ReadAssignment] = append(groupedByAssignment[summary.ReadAssignment], summary)
	}

	// Write a file for each read assignment
	for assignment, entries := range groupedByAssignment {
		filePath := filepath.Join(outputDir, fmt.Sprintf("%s_%s.txt", sampleName, assignment))
		file, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create read assignment file: %w", err)
		}
		defer file.Close()

		writer := csv.NewWriter(file)
		writer.Comma = '\t'
		defer writer.Flush()

		// Write entries (just the QName)
		for _, entry := range entries {
			if err := writer.Write([]string{entry.QName}); err != nil {
				return fmt.Errorf("failed to write record: %w", err)
			}
		}

		Logger.Infof("Wrote %d entries to read assignment file %s", len(entries), filePath)
	}

	return nil
}

// writePerformanceReport writes the performance report
func writePerformanceReport(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create performance report: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Comma = '\t'
	defer writer.Flush()

	// Write header
	header := []string{"operation", "duration_ms"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write entries
	metrics := GetPerformanceMetrics()
	for operation, duration := range metrics {
		record := []string{
			operation,
			strconv.FormatInt(duration.Milliseconds(), 10),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write record: %w", err)
		}
	}

	Logger.Infof("Wrote %d entries to performance report %s", len(metrics), filePath)
	return nil
}
