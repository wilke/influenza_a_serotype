package pafprocessor

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ReadMappingFile reads a TSV mapping file and returns a map of accession to MappingEntry
func ReadMappingFile(path string) (map[string]MappingEntry, error) {
	startTime := time.Now()
	defer func() {
		LogPerformance("ReadMappingFile", startTime)
	}()

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open mapping file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.LazyQuotes = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping file header: %w", err)
	}

	// Validate header
	expectedHeader := []string{"accession", "serotype", "segment", "Organism_Name", "Host", "Collection_Date"}
	for i, field := range expectedHeader {
		if i >= len(header) || header[i] != field {
			return nil, fmt.Errorf("invalid mapping file header: expected %s at position %d, got %s", field, i, header[i])
		}
	}

	// Read entries
	entries := make(map[string]MappingEntry)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read mapping file record: %w", err)
		}

		if len(record) < 6 {
			return nil, fmt.Errorf("invalid mapping file record: expected at least 6 fields, got %d", len(record))
		}

		segment := 0
		if record[2] != "" {
			// Try to parse as integer
			if s, err := strconv.Atoi(record[2]); err == nil {
				segment = s
			} else {
				// Handle segment names like PB1, PB2, etc.
				switch record[2] {
				case "PB2":
					segment = 1
				case "PB1":
					segment = 2
				case "PA":
					segment = 3
				case "HA":
					segment = 4
				case "NP":
					segment = 5
				case "NA":
					segment = 6
				case "M":
					segment = 7
				case "NS":
					segment = 8
				default:
					Logger.Warnf("Unknown segment name: %s, using 0", record[2])
				}
			}
		}

		entry := MappingEntry{
			Accession:      record[0],
			Serotype:       record[1],
			Segment:        segment,
			OrganismName:   record[3],
			Host:           record[4],
			CollectionDate: record[5],
		}

		entries[entry.Accession] = entry
	}

	Logger.Infof("Read %d entries from mapping file", len(entries))
	return entries, nil
}

// ReadPAFFile reads a PAF file in chunks and returns a channel of PAF entry chunks
func ReadPAFFile(path string, chunkSize int) (<-chan []PAFEntry, error) {
	startTime := time.Now()
	defer func() {
		LogPerformance("ReadPAFFile setup", startTime)
	}()

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open PAF file: %w", err)
	}

	chunks := make(chan []PAFEntry)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer file.Close()
		defer close(chunks)
		defer wg.Done()

		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024*10) // 10MB buffer

		chunk := make([]PAFEntry, 0, chunkSize)
		totalEntries := 0
		chunkCount := 0
		chunkStartTime := time.Now()

		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Split(line, "\t")

			if len(fields) < 12 {
				Logger.Warnf("Invalid PAF entry: expected at least 12 fields, got %d", len(fields))
				continue
			}

			qLength, _ := strconv.Atoi(fields[1])
			qStart, _ := strconv.Atoi(fields[2])
			qEnd, _ := strconv.Atoi(fields[3])
			tLength, _ := strconv.Atoi(fields[6])
			tStart, _ := strconv.Atoi(fields[7])
			tEnd, _ := strconv.Atoi(fields[8])
			numMatches, _ := strconv.Atoi(fields[9])
			alignLength, _ := strconv.Atoi(fields[10])
			mapQ, _ := strconv.Atoi(fields[11])

			entry := PAFEntry{
				QName:       fields[0],
				QLength:     qLength,
				QStart:      qStart,
				QEnd:        qEnd,
				Strand:      fields[4],
				TName:       fields[5],
				TLength:     tLength,
				TStart:      tStart,
				TEnd:        tEnd,
				NumMatches:  numMatches,
				AlignLength: alignLength,
				MapQ:        mapQ,
			}

			chunk = append(chunk, entry)
			totalEntries++

			if len(chunk) >= chunkSize {
				chunkCount++
				LogPerformance(fmt.Sprintf("ReadPAFFile chunk %d", chunkCount), chunkStartTime)
				chunks <- chunk
				chunk = make([]PAFEntry, 0, chunkSize)
				chunkStartTime = time.Now()
			}
		}

		if err := scanner.Err(); err != nil {
			Logger.Errorf("Error reading PAF file: %v", err)
			return
		}

		if len(chunk) > 0 {
			chunkCount++
			LogPerformance(fmt.Sprintf("ReadPAFFile chunk %d", chunkCount), chunkStartTime)
			chunks <- chunk
		}

		Logger.Infof("Read %d entries from PAF file in %d chunks", totalEntries, chunkCount)
	}()

	return chunks, nil
}
