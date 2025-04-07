package pafprocessor

import (
	"fmt"
	"sync"
	"time"
)

// ProcessChunk processes a chunk of PAF entries
func ProcessChunk(chunk []PAFEntry, mapping map[string]MappingEntry, minScore float64, paired bool, useRAlgorithm bool) ([]SummaryEntry, error) {
	startTime := time.Now()
	defer func() {
		LogPerformance("ProcessChunk", startTime)
	}()

	// Group entries by QName
	groupedByQName := make(map[string][]PAFEntry)
	for _, entry := range chunk {
		groupedByQName[entry.QName] = append(groupedByQName[entry.QName], entry)
	}

	Logger.Debugf("Grouped %d entries into %d QName groups", len(chunk), len(groupedByQName))

	var summaries []SummaryEntry

	// Process each QName group
	for qname, entries := range groupedByQName {
		groupSummaries, err := processQNameGroup(qname, entries, mapping, minScore, paired, useRAlgorithm)
		if err != nil {
			return nil, fmt.Errorf("failed to process QName group %s: %w", qname, err)
		}
		summaries = append(summaries, groupSummaries...)
	}

	Logger.Debugf("Processed %d QName groups into %d summaries", len(groupedByQName), len(summaries))
	return summaries, nil
}

// processQNameGroup processes a group of PAF entries with the same QName
func processQNameGroup(qname string, entries []PAFEntry, mapping map[string]MappingEntry, minScore float64, paired bool, useRAlgorithm bool) ([]SummaryEntry, error) {
	// 1. Compute alignment score for each entry
	var groupedEntries []GroupedEntry
	for _, entry := range entries {
		mappingEntry, ok := mapping[entry.TName]
		if !ok {
			Logger.Warnf("No mapping entry found for TName %s, using default values", entry.TName)
			// Use default values for unmapped entries
			mappingEntry = MappingEntry{
				Accession:      entry.TName,
				Serotype:       "unknown",
				Segment:        0,
				OrganismName:   "Unknown",
				Host:           "Unknown",
				CollectionDate: "",
			}
		}

		// Compute alignment score
		var alignScore float64
		if useRAlgorithm {
			// R algorithm: align_score = ANI * AF
			ani := float64(entry.NumMatches) / float64(entry.AlignLength)
			af := float64(entry.AlignLength) / float64(entry.QLength)
			alignScore = ani * af
		} else {
			// Original Go algorithm
			alignScore = float64(entry.NumMatches) / float64(entry.AlignLength)
		}

		// 2. Filter by min alignment score
		if alignScore < minScore {
			continue
		}

		groupedEntry := GroupedEntry{
			QName:       entry.QName,
			TName:       entry.TName,
			Serotype:    mappingEntry.Serotype,
			Segment:     mappingEntry.Segment,
			Strand:      entry.Strand,
			ReadLength:  entry.QLength,
			AlignLength: entry.AlignLength,
			NumMatches:  entry.NumMatches,
			ANI:         float64(entry.NumMatches) / float64(entry.AlignLength),
			AF:          float64(entry.AlignLength) / float64(entry.QLength),
			AlignScore:  alignScore,
		}

		groupedEntries = append(groupedEntries, groupedEntry)
	}

	if len(groupedEntries) == 0 {
		return nil, nil
	}

	// 3. Group by qname, tname, serotype, segment, strand
	groupedByFields := make(map[string][]GroupedEntry)
	for _, entry := range groupedEntries {
		var key string
		if paired {
			key = fmt.Sprintf("%s_%s_%s_%d_%s", entry.QName, entry.TName, entry.Serotype, entry.Segment, entry.Strand)
		} else {
			key = fmt.Sprintf("%s_%s_%s_%d", entry.QName, entry.TName, entry.Serotype, entry.Segment)
		}
		groupedByFields[key] = append(groupedByFields[key], entry)
	}

	// 4. Compute max and avg scores for each group
	var qssSummaries []SummaryEntry
	for _, entries := range groupedByFields {
		var totalScore float64
		var maxScore float64
		var totalReadLength int
		var totalAlignLength int
		var totalMatches int

		for _, entry := range entries {
			totalScore += entry.AlignScore
			if entry.AlignScore > maxScore {
				maxScore = entry.AlignScore
			}
			totalReadLength += entry.ReadLength
			totalAlignLength += entry.AlignLength
			totalMatches += entry.NumMatches
		}

		avgScore := totalScore / float64(len(entries))
		// Calculate ANI and AF for logging purposes
		ani := float64(totalMatches) / float64(totalAlignLength)
		af := float64(totalAlignLength) / float64(totalReadLength)
		Logger.Debugf("Group %s: ANI=%f, AF=%f, AlignScore=%f", entries[0].QName, ani, af, ani*af)

		// Collect all unique strands for this group
		strandMap := make(map[string]bool)
		for _, entry := range entries {
			strandMap[entry.Strand] = true
		}

		// Convert map keys to slice
		allStrands := make([]string, 0, len(strandMap))
		for strand := range strandMap {
			allStrands = append(allStrands, strand)
		}

		Logger.Debugf("Group %s: Found %d unique strands: %v", entries[0].QName, len(allStrands), allStrands)

		summary := SummaryEntry{
			QName:      entries[0].QName,
			Serotype:   entries[0].Serotype,
			Segment:    entries[0].Segment,
			Strand:     entries[0].Strand,
			Count:      len(entries),
			TopScore:   maxScore,
			AvgScore:   avgScore,
			AllStrands: allStrands,
		}

		qssSummaries = append(qssSummaries, summary)
	}

	// 5. Log grouping information based on paired flag
	if paired {
		Logger.Debugf("Using paired grouping for %s with strand %s", qname, entries[0].Strand)
	} else {
		Logger.Debugf("Using unpaired grouping for %s", qname)
	}

	// Find max top score within the group
	var maxTopScore float64
	for _, summary := range qssSummaries {
		if summary.TopScore > maxTopScore {
			maxTopScore = summary.TopScore
		}
	}

	// Keep entries within maxTopScore - lowerBound
	const lowerBound = 0.003
	var filteredSummaries []SummaryEntry
	for _, summary := range qssSummaries {
		if summary.TopScore >= maxTopScore-lowerBound {
			filteredSummaries = append(filteredSummaries, summary)
		}
	}

	// 6. Create read assignment
	if useRAlgorithm {
		// R algorithm:
		// 1. Take top 2 scores
		// 2. If max(top_score - 0.003) >= min(top_score) OR all serotypes are the same, assign first serotype
		// 3. Otherwise, assign "ambiguous"

		// Sort by TopScore in descending order (already done above)

		// Take top 2 scores (or all if less than 2)
		var topSummaries []SummaryEntry
		if len(filteredSummaries) > 2 {
			topSummaries = filteredSummaries[:2]
		} else {
			topSummaries = filteredSummaries
		}

		// Check if all serotypes are the same
		allSameSerotype := true
		serotype := ""
		if len(topSummaries) > 0 {
			serotype = topSummaries[0].Serotype
			for _, summary := range topSummaries {
				if summary.Serotype != serotype {
					allSameSerotype = false
					break
				}
			}
		}

		// Apply R algorithm logic
		if len(topSummaries) > 0 {
			if allSameSerotype || len(topSummaries) == 1 {
				// All serotypes are the same, assign the serotype
				for i := range filteredSummaries {
					filteredSummaries[i].ReadAssignment = serotype
				}
			} else if len(topSummaries) >= 2 {
				// Check if max(top_score - 0.003) >= min(top_score)
				maxScore := topSummaries[0].TopScore
				minScore := topSummaries[1].TopScore

				if maxScore-0.003 >= minScore {
					// Scores are close enough, assign the top serotype
					for i := range filteredSummaries {
						filteredSummaries[i].ReadAssignment = topSummaries[0].Serotype
					}
				} else {
					// Scores are too different, assign "ambiguous"
					for i := range filteredSummaries {
						filteredSummaries[i].ReadAssignment = "ambiguous"
					}
				}
			}
		}
	} else {
		// Original Go algorithm
		// Check if all serotypes are the same
		allSameSerotype := true
		serotype := ""
		if len(filteredSummaries) > 0 {
			serotype = filteredSummaries[0].Serotype
			for _, summary := range filteredSummaries {
				if summary.Serotype != serotype {
					allSameSerotype = false
					break
				}
			}
		}

		// Assign serotype or "ambiguous"
		for i := range filteredSummaries {
			if allSameSerotype {
				filteredSummaries[i].ReadAssignment = serotype
			} else {
				filteredSummaries[i].ReadAssignment = "ambiguous"
			}
		}
	}

	return filteredSummaries, nil
}

// ProcessAllChunks processes all chunks of PAF entries in parallel
func ProcessAllChunks(chunks <-chan []PAFEntry, mapping map[string]MappingEntry, minScore float64, paired bool, numWorkers int, useRAlgorithm bool) ([]SummaryEntry, error) {
	startTime := time.Now()
	defer func() {
		LogPerformance("ProcessAllChunks", startTime)
	}()

	var wg sync.WaitGroup
	resultChan := make(chan []SummaryEntry, numWorkers)
	errorChan := make(chan error, numWorkers)

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			workerStartTime := time.Now()

			for chunk := range chunks {
				chunkStartTime := time.Now()
				summaries, err := ProcessChunk(chunk, mapping, minScore, paired, useRAlgorithm)
				if err != nil {
					errorChan <- err
					return
				}
				resultChan <- summaries
				LogPerformance(fmt.Sprintf("Worker %d processed chunk", workerID), chunkStartTime)
			}

			LogPerformance(fmt.Sprintf("Worker %d total", workerID), workerStartTime)
		}(i)
	}

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	// Check for errors
	select {
	case err := <-errorChan:
		if err != nil {
			return nil, err
		}
	default:
		// No errors
	}

	// Collect results
	var allSummaries []SummaryEntry
	for summaries := range resultChan {
		allSummaries = append(allSummaries, summaries...)
	}

	Logger.Infof("Processed %d summaries in total", len(allSummaries))
	return allSummaries, nil
}
