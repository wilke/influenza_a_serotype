package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
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
}

//  This is a simple program which reads
//  1. file in paf format
//  2. file in serotype format
//  and generates a bar chart of read assignments by serotype

func StreamPaf2Record(pafFile string, record chan []PafHit) error {
	// Open the PAF file
	log.Println("Opening PAF file:", pafFile)
	file, err := os.Open(pafFile)
	if err != nil {
		log.Fatalln("Error opening PAF file:", err)
	}
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
		return fmt.Errorf("failed to read PAF file: %w", err)
	}

	// Send the last batch of hits
	if len(hits) > 0 {
		log.Printf("Processing Qname: %s, Line Count: %d\n", Qname, lineCount)
		record <- hits
	}
	// Close the channel to signal completion
	close(record)

	return nil
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

func WaitRandomly(record []PafHit, summaries chan string) {
	// Wait for a random time between 1 and 10 seconds
	time.Sleep(time.Duration(rand.Intn(10)+1) * time.Second)
	summaries <- fmt.Sprintf("Processed %d records", len(record))

}

// func WaitRandomly() {

// func CalculateScores(record chan []string, mapping Mapping) (scores Scores, error) {
// }

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
	flag.StringVar(&outputDir, "output", "", "Path to the output directory")
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
	logger.Printf("Mapping: %v\n", mapping)

	record := make(chan []PafHit, 10)
	go StreamPaf2Record(pafFile, record)

	debug := false

	summaries := make(chan string, 10)
	var wg sync.WaitGroup

	for r := range record {
		// logger.Println(lines[l])
		// fmt.Printf("Processing PAF record with %d entries.\n", len(r))
		wg.Add(1)
		go func(rec []PafHit) {
			defer wg.Done()
			WaitRandomly(rec, summaries)
		}(r)

		if debug == true {
			for l := range r {
				fmt.Println(r[l].MappingQuality)
				// fmt.Print(".")
				// fmt.Print(l)
			}
			fmt.Println()
		}
	}

	// Close the summaries channel after all goroutines complete
	go func() {
		wg.Wait()
		close(summaries)
	}()

	for s := range summaries {
		fmt.Println(s)
	}
}
