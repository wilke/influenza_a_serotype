package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	// "github.com/me/influenza_a_serotype/lib/go/paf2serotypes/log"
)

const debug = true

var SegmentMap = map[string]string{
	"PB2": "1", // Segment 1
	"PB1": "2", // Segment 2
	"PA":  "3", // Segment 3
	"HA":  "4", // Segment 4
	"NP":  "5", // Segment 5
	"NA":  "6", // Segment 6
	"M1":  "7", // Segment 7
	"NS":  "8", // Segment 8
}

type Accession string
type Serotype string
type Segment string
type OrganismName string
type Host string
type CollectionDate string

// Create type Mapping which is a mapping from Accession to { Serotype, Segment, OrganismName, Host, CollectionDate }
type Mapping map[Accession]struct {
	Serotype       Serotype
	Segment        Segment
	OrganismName   OrganismName
	Host           Host
	CollectionDate CollectionDate
}

// PafHit represents a single record in the PAF file format
type PafHit struct {
	QueryName            string // Field 1: Name of the query sequence
	QueryLength          int    // Field 2: Length of the query sequence
	QueryStart           int    // Field 3: Start position (0-based) on the query
	QueryEnd             int    // Field 4: End position (0-based, exclusive) on the query
	Strand               string // Field 5: '+' or '-', indicating the query strand
	TargetName           string // Field 6: Name of the target sequence
	TargetLength         int    // Field 7: Length of the target sequence
	TargetStart          int    // Field 8: Start position on the target
	TargetEnd            int    // Field 9: End position (exclusive) on the target
	NumResidueMatches    int    // Field 10: Number of matching bases
	AlignmentBlockLength int    // Field 11: Total length of the alignment block
	MappingQuality       int    // Field 12: Mapping quality (0–255, 255 means missing)
	Serotype             Serotype
}

type PafRecord []PafHit

type PafScore struct {
	PafHit
	TotalReadLength      int
	TotalAlignmentLength int
	TotalMatches         int
	ANI                  float64
	AFI                  float64
	AlignmentScore       float64
}

type Summary struct {
	scores                []PafScore
	maxAlignmentScore     float64
	avgAlignmentScore     float64
	Serotype              Serotype
	Segment               Segment
	QueryName             string
	NumberOfContributions int
	groupedBy             string
}

// final Assignment
type Assignmnet struct {
	Serotype          Serotype
	QueryName         string
	Summaries         []Summary
	AlignmentScore    float64
	AvgAlignmentScore float64
	Hits              int
	SerotypeMap       map[Serotype]int
}

// type Summary []SummaryEntry

//  This is a simple program which reads
//  1. file in paf format
//  2. file in serotype format
//  and generates a bar chart of read assignments by serotype

func StreamPaf2Record(pafFile string) (<-chan string, <-chan []PafHit) {

	// Create a channel to send PafHit records
	// This channel will be used to send batches of PafHit records
	record := make(chan []PafHit, 10)
	errorMessage := make(chan string, 1)
	// record chan []PafHit

	// Open the PAF file
	log.Println("Opening PAF file:", pafFile)
	file, err := os.Open(pafFile)
	if err != nil {
		log.Fatalln("Error opening PAF file:", err)
		errorMessage <- fmt.Sprintf("Error opening PAF file: %s", err)
		return errorMessage, record
	}
	// defer file.Close()

	go func() {
		// Close the channel when done
		defer close(record)
		defer close(errorMessage)
		defer file.Close()

		// Read the file line by line
		var hits []PafHit
		var lineCount int
		var Qname string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			// Split the line into fields
			line := scanner.Text()
			if line == "" {
				continue
			}

			// Split the line by tab
			fields := strings.Split(line, "\t")
			if len(fields) < 12 {
				log.Printf("Skipping invalid PAF line: %s\n", line)
				continue
			}

			if Qname != fields[0] {
				// Print qname and line count
				if debug && Qname != "" {
					log.Printf("Processing Qname: %s, Line Count: %d\n", Qname, lineCount)
				}

				if len(hits) > 0 {
					record <- hits
				}
				// Reset hits for new Qname
				hits = []PafHit{}
				// New Qname
				Qname = fields[0]
				// Reset line count for new Qname
				lineCount = 0
			}

			// Parse fields into PafHit
			queryLength, _ := strconv.Atoi(fields[1])
			queryStart, _ := strconv.Atoi(fields[2])
			queryEnd, _ := strconv.Atoi(fields[3])
			targetLength, _ := strconv.Atoi(fields[6])
			targetStart, _ := strconv.Atoi(fields[7])
			targetEnd, _ := strconv.Atoi(fields[8])
			numResidueMatches, _ := strconv.Atoi(fields[9])
			alignmentBlockLength, _ := strconv.Atoi(fields[10])
			mappingQuality, _ := strconv.Atoi(fields[11])

			hit := PafHit{
				QueryName:            fields[0],
				QueryLength:          queryLength,
				QueryStart:           queryStart,
				QueryEnd:             queryEnd,
				Strand:               fields[4],
				TargetName:           fields[5],
				TargetLength:         targetLength,
				TargetStart:          targetStart,
				TargetEnd:            targetEnd,
				NumResidueMatches:    numResidueMatches,
				AlignmentBlockLength: alignmentBlockLength,
				MappingQuality:       mappingQuality,
			}

			hits = append(hits, hit)
			lineCount++
		}

		if err := scanner.Err(); err != nil {
			errorMessage <- fmt.Sprintf("Error reading PAF file: %s", err)
		}

		// Send the last batch of hits
		if len(hits) > 0 {
			log.Printf("Processing Qname: %s, Line Count: %d\n", Qname, lineCount)
			record <- hits
		}
	}()
	// Close the channel to signal completion
	// close(record)

	return errorMessage, record
}

func LoadMappingFile(filename string) (mapping Mapping, err error) {

	mapping = make(Mapping)

	// Open the mapping file
	log.Println("Opening mapping file:", filename)
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalln("Error opening mapping file:", err)
	}
	defer file.Close()

	// Read the mapping file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Split the line by tab
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}

		// Process the mapping
		debug := false
		if debug {
			log.Printf("Processing mapping: %s -> %s\n", fields[0], fields[1])
		}
		// fmt.Println(fields)

		// Check if field 2 is a valid serotype and starts with an uppercase H
		if len(fields[1]) > 0 && fields[1][0] == 'H' {
			// Check if field 3 is not a number and not between 1 and 8
			if len(fields[2]) > 0 && (fields[2][0] < '1' || fields[2][0] > '8') {

				// Check if field 2 is a valid segment
				if _, ok := SegmentMap[fields[2]]; !ok {
					log.Printf("Invalid segment: %s\n", fields[2])
					continue
				} else {
					// Assign the segment number
					fields[2] = SegmentMap[fields[2]]
				}
			}
			// Create a new mapping entry

			mapping[Accession(fields[0])] = struct {
				Serotype       Serotype
				Segment        Segment
				OrganismName   OrganismName
				Host           Host
				CollectionDate CollectionDate
			}{
				Serotype:       Serotype(fields[1]),
				Segment:        Segment(fields[2]),
				OrganismName:   OrganismName(fields[3]),
				Host:           Host(fields[4]),
				CollectionDate: CollectionDate(fields[5]),
			}
		} else {
			log.Printf("Skipping invalid mapping: %s\n", fields[1])
			continue
		}

		// os.Exit(1)
	}

	if err := scanner.Err(); err != nil {
		return Mapping{}, fmt.Errorf("failed to read mapping file: %w", err)
	}

	return mapping, nil
}

// Create a function which waits randomly for 1 - 10 seconds
// and then returns the scores

func GetNTopScores(summaries []Summary, n int) []float64 {

	scoresMap := make(map[float64]int)
	scoresList := make([]float64, 0)

	for _, summary := range summaries {
		scoresMap[summary.maxAlignmentScore] += 1
	}

	for score := range scoresMap {
		scoresList = append(scoresList, score)
	}
	// Sort the scores by alignment score
	sort.Slice(scoresList, func(i, j int) bool {
		return scoresList[i] > scoresList[j]
	})

	// Get the top n scores
	topScores := make([]float64, 0)
	if len(scoresList) > n {
		topScores = scoresList[:n]
	} else {
		topScores = scoresList
	}

	return topScores
}

func GetSummariesByScore(summaries []Summary, scores []float64) ([]Summary, map[Serotype]int) {

	// Select the summaries which have the top scores
	filteredSummaries := make([]Summary, 0)

	// Create a map of serotypes and return a list of serotypes in addition to the summaries
	serotypeMap := make(map[Serotype]int)

	for _, summary := range summaries {
		for _, score := range scores {
			if summary.maxAlignmentScore == score {
				filteredSummaries = append(filteredSummaries, summary)
				serotypeMap[summary.Serotype] += 1
			}
		}
	}
	// Sort the filtered summaries by max alignment score
	sort.Slice(filteredSummaries, func(i, j int) bool {
		return filteredSummaries[i].maxAlignmentScore > filteredSummaries[j].maxAlignmentScore
	})

	return filteredSummaries, serotypeMap
}

func WaitRandomly(record []PafHit, summaries chan string) {
	// Wait for a random time between 1 and 10 seconds
	time.Sleep(time.Duration(rand.Intn(10)+1) * time.Second)
	summaries <- fmt.Sprintf("Processed %d records", len(record))

}

// Create a function which calculates the scores
func CalculateScores(records <-chan []PafHit, mapping Mapping) <-chan []PafScore {
	scores := make(chan []PafScore, 10)
	// Create a wait group to wait for all goroutines to finish
	var wg sync.WaitGroup

	numberOfTasks := 4
	task := make(chan int, numberOfTasks)

	// Calculate scores for each record
	for record := range records {
		task <- 1
		wg.Add(1)
		fmt.Printf("Summary for record %s\n", record[0].QueryName)
		go func(record []PafHit) {
			defer wg.Done()

			// First goup by qname, tname, serotype, segment, strand
			// Then calculate the scores for each group

			groupsStrand := make(map[string][]PafHit)

			for _, hit := range record {
				mapped := mapping[Accession(hit.TargetName)]
				key := fmt.Sprintf("%s_%s_%s_%s_%s", hit.QueryName, hit.TargetName, mapped.Serotype, mapped.Segment, hit.Strand)
				groupsStrand[key] = append(groupsStrand[key], hit)
			}
			fmt.Printf("Groups in record %d\n", len(groupsStrand))

			// For each group, calculate the scores
			Scores := []PafScore{}

			// Iterate over the groups
			for key, group := range groupsStrand {
				// Calculate the scores
				fmt.Printf("Processing group %s\n", key)
				var totalReadLength, totalAlignmentLength, totalMatches int

				for _, hit := range group {
					totalReadLength += hit.QueryLength
					totalAlignmentLength += hit.AlignmentBlockLength
					totalMatches += hit.NumResidueMatches
				}

				for _, hit := range group {
					// Calculate the ANI
					// ANI = (totalMatches / totalAlignmentLength) * 100
					// AFI = (totalMatches / totalReadLength) * 100
					// AlignmentScore = (totalMatches / totalAlignmentLength) * 100

					ani := float64(totalMatches) / float64(totalAlignmentLength)
					afi := float64(totalAlignmentLength) / float64(totalReadLength)
					alignmentScore := float64(ani) * float64(afi)

					Score := PafScore{
						PafHit:               hit,
						TotalReadLength:      totalReadLength,
						TotalAlignmentLength: totalAlignmentLength,
						TotalMatches:         totalMatches,
						ANI:                  float64(totalMatches) / float64(totalAlignmentLength),
						AFI:                  float64(totalAlignmentLength) / float64(totalReadLength),
						AlignmentScore:       alignmentScore,
					}
					Scores = append(Scores, Score)
				}

				// Sort scores by alignment score
				sort.Slice(Scores, func(i, j int) bool {
					return Scores[i].AlignmentScore > Scores[j].AlignmentScore
				})

				// Get top two top scores
				scoreList := map[float64][]PafScore{}
				for _, score := range Scores {
					scoreList[score.AlignmentScore] = append(scoreList[score.AlignmentScore], score)
				}
				// Sort the scores by alignment score
				var topScores []float64
				for k := range scoreList {
					topScores = append(topScores, k)
				}
				// sort descending topScores
				sort.Slice(topScores, func(i, j int) bool {
					return topScores[i] > topScores[j]
				})

				// Get the top two scores
				if len(topScores) > 2 {
					topScores = topScores[:2]
				} else {
					topScores = topScores[:len(topScores)]
				}

				// Print the top scores
				fmt.Printf("Top scores: %v\n", topScores)
				// os.Exit(1)

			}

			scores <- Scores
			<-task
		}(record)
	}

	go func() {
		wg.Wait()
		close(scores)
	}()

	return scores
}

func AssignSerotype(scoreRecords <-chan []PafScore, mapping Mapping, topScoreThreshold float64, scoreDistance float64) <-chan Assignmnet {

	assignments := make(chan Assignmnet, 10)
	wg := sync.WaitGroup{}

	for scoreRecord := range scoreRecords {

		// summary := make(Summary, 0)

		// Compute top alignmnet score and average alignment score
		// for each group of qname, serotype, segment in the []PafScore
		wg.Add(1)
		go func() {

			defer wg.Done()
			groupsSegment := make(map[string][]PafScore)
			for _, score := range scoreRecord {
				mapped := mapping[Accession(score.PafHit.TargetName)]
				key := fmt.Sprintf("%s_%s_%s", score.PafHit.QueryName, mapped.Serotype, mapped.Segment)
				groupsSegment[key] = append(groupsSegment[key], score)
			}

			summaries := make([]Summary, 0)
			// For each group, calculate max and average alignment score
			for key, group := range groupsSegment {
				var summaryForGroup Summary
				var totalAlignmentScore float64
				var maxAlignmentScore float64
				var avgAlignmentScore float64

				// Set groupedBy to the key
				summaryForGroup.groupedBy = key
				summaryForGroup.Serotype = mapping[Accession(group[0].PafHit.TargetName)].Serotype
				summaryForGroup.Segment = mapping[Accession(group[0].PafHit.TargetName)].Segment
				summaryForGroup.QueryName = group[0].PafHit.QueryName
				summaryForGroup.NumberOfContributions = len(group)

				// Calculate the max and average alignment score
				for _, score := range group {
					totalAlignmentScore += score.AlignmentScore
					if score.AlignmentScore > maxAlignmentScore {
						maxAlignmentScore = score.AlignmentScore
					}
				}
				summaryForGroup.scores = group
				avgAlignmentScore = totalAlignmentScore / float64(len(group))
				summaryForGroup.maxAlignmentScore = maxAlignmentScore
				summaryForGroup.avgAlignmentScore = avgAlignmentScore

				fmt.Printf("Group %s | %s: Max Alignment Score: %f, Avg Alignment Score: %f\t%v\n", key,
					summaryForGroup.QueryName,
					summaryForGroup.maxAlignmentScore,
					summaryForGroup.avgAlignmentScore,
					summaryForGroup.NumberOfContributions)
				summaries = append(summaries, summaryForGroup)
				// os.Exit(1)
			}

			// Print number of groups and number of summaries
			fmt.Printf("Number of groups: %d, Number of summaries: %d\n", len(groupsSegment), len(summaries))

			// Sort by top scores, get top 2 unique scores
			sort.Slice(summaries, func(i, j int) bool {
				return summaries[i].maxAlignmentScore > summaries[j].maxAlignmentScore
			})

			for i, summary := range summaries {
				fmt.Printf("Summary %d: %s\t%f\t%f\n", i, summary.QueryName, summary.maxAlignmentScore, summary.avgAlignmentScore)
			}

			topScores := GetNTopScores(summaries, 2)
			fmt.Printf("Top scores: %v\n", topScores)

			// Limit the top scores to only scores within scoreDistance interval
			maxScore := topScores[0]
			if len(topScores) > 2 {
				// Remove scores which are not within the scoreDistance
				for i := 1; i < len(topScores); i++ {
					if topScores[i] < maxScore-scoreDistance {
						topScores = topScores[:i]
						break
					}
				}
			}
			fmt.Printf("Top scores interval: %v\n", topScores)

			// Get the top scores within the scoreDistance

			topSummariesForScores, serotypeMap := GetSummariesByScore(summaries, topScores)
			fmt.Printf("Top summaries for scores: %v\n", len(topSummariesForScores))

			if topScores[0] < topScoreThreshold {
				fmt.Printf("Group %s: No serotype assigned. Score %d below threshold %d\n", topSummariesForScores[0].QueryName, topScores[0], topScoreThreshold)
				return
			}

			// Scores are above threshold and within the scoreDistance
			// Check if there are multiple unique serotypes, if so, assign "ambiguous"

			serotypeList := make([]Serotype, 0)
			for s := range serotypeMap {
				serotypeList = append(serotypeList, s)
			}
			assignment := Assignmnet{}
			assignment.QueryName = topSummariesForScores[0].QueryName
			assignment.Summaries = topSummariesForScores
			assignment.AlignmentScore = topScores[0]
			assignment.AvgAlignmentScore = topSummariesForScores[0].avgAlignmentScore
			assignment.SerotypeMap = serotypeMap
			// Hits is the sum over all PafScore
			for _, summary := range topSummariesForScores {
				assignment.Hits += len(summary.scores)
			}

			// Assign the serotype
			if len(serotypeList) > 1 {
				fmt.Printf("Group %s: Multiple serotypes found: %v\n", topSummariesForScores[0].QueryName, serotypeList)
				assignment.Serotype = "ambiguous"
			} else {
				// Get the serotype
				fmt.Printf("Group %s: Serotype: %s\n", topSummariesForScores[0].QueryName, serotypeList[0])
				assignment.Serotype = serotypeList[0]
			}
			assignments <- assignment
		}()
	}

	go func() {
		wg.Wait()
		close(assignments)
	}()

	return assignments
}

func exportAssignmnetsToFile(assignments <-chan Assignmnet, outputDir string) {

	// Create one output file for each distinct serotype
	// Create a map of serotypes to file handles
	serotypeFiles := make(map[Serotype]*os.File)

	// Create a wait group to wait for all goroutines to finish
	// var wg sync.WaitGroup

	for assignment := range assignments {

		if _, ok := serotypeFiles[assignment.Serotype]; !ok {
			// Create a new file for the serotype
			fileName := fmt.Sprintf("%s/%s.serotype.txt", outputDir, assignment.Serotype)
			file, err := os.Create(fileName)
			if err != nil {
				log.Fatalln("Error creating file:", err)
			}
			serotypeFiles[assignment.Serotype] = file
		}
		// Write the assignment to the file
		file := serotypeFiles[assignment.Serotype]
		_, err := file.WriteString(fmt.Sprintf("%s\t%s\t%f\t%f\t%d\n", assignment.QueryName, assignment.Serotype, assignment.AlignmentScore, assignment.AvgAlignmentScore, assignment.Hits))
		if err != nil {
			log.Fatalln("Error writing to file:", err)
		}

	}
	// Close all files
	for _, file := range serotypeFiles {
		err := file.Close()
		if err != nil {
			log.Fatalln("Error closing file:", err)
		}
	}
}

func makeGlobalSummary(assignments <-chan Assignmnet, outputDir string) {
	// Create a global summary file
	fileName := fmt.Sprintf("%s/global_summary.txt", outputDir)
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatalln("Error creating file:", err)
	}
	defer file.Close()

	// Write the header to the file
	_, err = file.WriteString("Serotype\tCount\n")
	if err != nil {
		log.Fatalln("Error writing to file:", err)
	}

	// Create a map of serotypes to counts
	serotypeCounts := make(map[Serotype]int)

	for assignment := range assignments {
		serotypeCounts[assignment.Serotype] += 1
	}

	// Write the counts to the file
	for serotype, count := range serotypeCounts {
		_, err = file.WriteString(fmt.Sprintf("%s\t%d\n", serotype, count))
		if err != nil {
			log.Fatalln("Error writing to file:", err)
		}
	}
}

func main() {

	// Initialize logger
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)

	var (
		pafFile     string
		mappingFile string
		outputDir   string
		numWorkers  int
	)

	flag.StringVar(&pafFile, "paf", "", "Path to the input PAF file")
	flag.StringVar(&mappingFile, "mapping-file", "", "Path to the serotype mapping file")
	flag.StringVar(&outputDir, "output", "./", "Path to the output directory")
	flag.IntVar(&numWorkers, "workers", 1, "Number of workers to use for processing")

	flag.Parse()

	// Log the input arguments
	logger.Println("PAF file:", pafFile)
	logger.Println("Mapping file:", mappingFile)
	logger.Println("Output directory:", outputDir)
	logger.Println("Number of workers:", numWorkers)

	// Load the mapping file
	mapping, err := LoadMappingFile(mappingFile)
	if err != nil {
		logger.Fatalln("Error loading mapping file:", err)
	}
	logger.Println("Loaded mapping file successfully")
	logger.Printf("Mapping: %v\n", len(mapping))

	// record := make(chan []PafHit, 10)
	errorMessage, records := StreamPaf2Record(pafFile)

	go func() {
		for errMsg := range errorMessage {
			logger.Println("Error:", errMsg)
		}
	}()

	scores := CalculateScores(records, mapping)
	assignments := AssignSerotype(scores, mapping, 0.8, 0.003)

	toFile := make(chan Assignmnet)
	assignmentsToCounts := make(chan Assignmnet)

	go exportAssignmnetsToFile(toFile, outputDir)
	go makeGlobalSummary(assignmentsToCounts, outputDir)

	for assignment := range assignments {

		logger.Printf("Assignment: %v\t%v\t%v\t%v\n", assignment.QueryName, assignment.Serotype, len(assignment.Summaries), assignment.Hits)
		logger.Printf("Assignment Serotype Map: %v\n", assignment.SerotypeMap)

		toFile <- assignment
		assignmentsToCounts <- assignment

	}

	close(toFile)
	close(assignmentsToCounts)

}
