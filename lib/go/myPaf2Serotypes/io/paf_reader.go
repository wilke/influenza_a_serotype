package io

import (
	"bufio"
	"context"
	"os"
	"sync"

	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/errors"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/models"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/utils"
)

// PAFFileReader implements PAFReader for reading PAF files
type PAFFileReader struct {
	filename string
	mapping  models.Mapping
	logger   utils.Logger
	file     *os.File
	scanner  *bufio.Scanner
	bufSize  int
}

// NewPAFFileReader creates a new PAF file reader
func NewPAFFileReader(filename string, mapping models.Mapping, logger utils.Logger, bufSize int) (*PAFFileReader, error) {
	if filename == "" {
		return nil, errors.ValidationError("PAFFileReader", "NewPAFFileReader", "filename cannot be empty")
	}
	if mapping == nil {
		return nil, errors.ValidationError("PAFFileReader", "NewPAFFileReader", "mapping cannot be nil")
	}
	if logger == nil {
		return nil, errors.ValidationError("PAFFileReader", "NewPAFFileReader", "logger cannot be nil")
	}
	if bufSize <= 0 {
		bufSize = 4096 // Default buffer size
	}

	return &PAFFileReader{
		filename: filename,
		mapping:  mapping,
		logger:   logger,
		bufSize:  bufSize,
	}, nil
}

// Read implements the PAFReader interface
func (r *PAFFileReader) Read(ctx context.Context) (<-chan []models.PafHit, <-chan error) {
	records := make(chan []models.PafHit, 100)
	errorChan := make(chan error, 1)

	go func() {
		defer close(records)
		defer close(errorChan)

		// Open file
		file, err := os.Open(r.filename)
		if err != nil {
			errorChan <- errors.IOError("PAFFileReader", "Read", "failed to open PAF file", err).
				WithContext("filename", r.filename)
			return
		}
		r.file = file
		defer r.Close()

		r.logger.Info("Starting PAF file reading",
			utils.String("filename", r.filename),
			utils.Int("buffer_size", r.bufSize),
		)

		// Create scanner with custom buffer
		r.scanner = bufio.NewScanner(file)
		buf := make([]byte, r.bufSize)
		r.scanner.Buffer(buf, r.bufSize*10) // Allow for larger lines

		var currentHits []models.PafHit
		var currentQuery string
		lineNum := 0
		recordCount := 0

		for r.scanner.Scan() {
			// Check context cancellation
			select {
			case <-ctx.Done():
				errorChan <- errors.CanceledError("PAFFileReader", "Read")
				return
			default:
			}

			lineNum++
			line := r.scanner.Text()

			// Skip empty lines
			if line == "" {
				continue
			}

			// Parse PAF line
			hit, err := models.ParsePafLine(line)
			if err != nil {
				r.logger.Warn("Skipping invalid PAF line",
					utils.Int("line_number", lineNum),
					utils.Err(err),
				)
				continue
			}

			// Enrich with mapping data
			if entry, ok := r.mapping.GetEntry(models.Accession(hit.TargetName)); ok {
				hit.Serotype = entry.Serotype
			} else {
				r.logger.Debug("No mapping found for target",
					utils.String("target", hit.TargetName),
					utils.Int("line_number", lineNum),
				)
				continue // Skip hits without mapping
			}

			// Check if we're still processing the same query
			if currentQuery == "" {
				currentQuery = hit.QueryName
			}

			if hit.QueryName != currentQuery {
				// Send current batch
				if len(currentHits) > 0 {
					select {
					case records <- currentHits:
						recordCount++
					case <-ctx.Done():
						errorChan <- errors.CanceledError("PAFFileReader", "Read")
						return
					}
				}

				// Start new batch
				currentQuery = hit.QueryName
				currentHits = []models.PafHit{*hit}
			} else {
				currentHits = append(currentHits, *hit)
			}
		}

		// Send last batch
		if len(currentHits) > 0 {
			select {
			case records <- currentHits:
				recordCount++
			case <-ctx.Done():
				errorChan <- errors.CanceledError("PAFFileReader", "Read")
				return
			}
		}

		// Check for scanner errors
		if err := r.scanner.Err(); err != nil {
			errorChan <- errors.IOError("PAFFileReader", "Read", "error reading PAF file", err).
				WithContext("filename", r.filename).
				WithContext("line_number", lineNum)
			return
		}

		r.logger.Info("Completed PAF file reading",
			utils.String("filename", r.filename),
			utils.Int("lines_processed", lineNum),
			utils.Int("records_sent", recordCount),
		)
	}()

	return records, errorChan
}

// Close closes the reader and releases resources
func (r *PAFFileReader) Close() error {
	if r.file != nil {
		if err := r.file.Close(); err != nil {
			return errors.IOError("PAFFileReader", "Close", "failed to close file", err).
				WithContext("filename", r.filename)
		}
		r.file = nil
	}
	return nil
}

// PAFStreamReader reads PAF records from a stream
type PAFStreamReader struct {
	reader   bufio.Reader
	mapping  models.Mapping
	logger   utils.Logger
	mu       sync.Mutex
	closed   bool
}

// NewPAFStreamReader creates a new PAF stream reader
func NewPAFStreamReader(reader *bufio.Reader, mapping models.Mapping, logger utils.Logger) (*PAFStreamReader, error) {
	if reader == nil {
		return nil, errors.ValidationError("PAFStreamReader", "NewPAFStreamReader", "reader cannot be nil")
	}
	if mapping == nil {
		return nil, errors.ValidationError("PAFStreamReader", "NewPAFStreamReader", "mapping cannot be nil")
	}
	if logger == nil {
		return nil, errors.ValidationError("PAFStreamReader", "NewPAFStreamReader", "logger cannot be nil")
	}

	return &PAFStreamReader{
		reader:  *reader,
		mapping: mapping,
		logger:  logger,
	}, nil
}

// ReadBatch reads a batch of PAF hits for the same query
func (r *PAFStreamReader) ReadBatch(ctx context.Context) ([]models.PafHit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, errors.ProcessingError("PAFStreamReader", "ReadBatch", "reader is closed")
	}

	var hits []models.PafHit
	var currentQuery string

	for {
		// Check context
		select {
		case <-ctx.Done():
			return nil, errors.CanceledError("PAFStreamReader", "ReadBatch")
		default:
		}

		line, err := r.reader.ReadString('\n')
		if err != nil {
			if err.Error() == "EOF" && len(hits) > 0 {
				return hits, nil
			}
			return nil, err
		}

		// Skip empty lines
		if line == "" || line == "\n" {
			continue
		}

		// Parse PAF line
		hit, err := models.ParsePafLine(line[:len(line)-1]) // Remove newline
		if err != nil {
			r.logger.Debug("Skipping invalid PAF line", utils.Err(err))
			continue
		}

		// Enrich with mapping data
		if entry, ok := r.mapping.GetEntry(models.Accession(hit.TargetName)); ok {
			hit.Serotype = entry.Serotype
		} else {
			continue // Skip hits without mapping
		}

		// Check query name
		if currentQuery == "" {
			currentQuery = hit.QueryName
		} else if hit.QueryName != currentQuery {
			// Put back the line for next batch
			// Note: This is a simplified approach. In production, you might want
			// to implement a proper pushback mechanism
			return hits, nil
		}

		hits = append(hits, *hit)
	}
}

// Close closes the stream reader
func (r *PAFStreamReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	return nil
}