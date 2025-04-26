package io

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/model"
)

// Reader interface for reading input files
type Reader interface {
	ReadChunk(chunkSize int) (interface{}, error)
	Close() error
}

// PafReader reads PAF files in a streaming fashion
type PafReader struct {
	file          *os.File
	scanner       *bufio.Scanner
	buffer        []model.PafRecord
	recordPool    *sync.Pool
	adaptiveChunk bool
	minChunkSize  int
	maxChunkSize  int
}

// PafReaderOptions contains options for configuring the PAF reader
type PafReaderOptions struct {
	MaxScanTokenSize int  // Maximum size of scanner buffer
	UseRecordPool    bool // Whether to use object pooling for records
	AdaptiveChunk    bool // Whether to use adaptive chunk sizing
	MinChunkSize     int  // Minimum chunk size for adaptive sizing
	MaxChunkSize     int  // Maximum chunk size for adaptive sizing
}

// DefaultPafReaderOptions returns default options for the PAF reader
func DefaultPafReaderOptions() PafReaderOptions {
	return PafReaderOptions{
		MaxScanTokenSize: 1024 * 1024, // 1MB
		UseRecordPool:    true,
		AdaptiveChunk:    true,
		MinChunkSize:     1000,
		MaxChunkSize:     10000,
	}
}

// NewPafReader creates a new PAF reader with default options
func NewPafReader(filename string) (*PafReader, error) {
	return NewPafReaderWithOptions(filename, DefaultPafReaderOptions())
}

// NewPafReaderWithOptions creates a new PAF reader with custom options
func NewPafReaderWithOptions(filename string, options PafReaderOptions) (*PafReader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open PAF file: %w", err)
	}

	scanner := bufio.NewScanner(file)

	// Set a larger buffer for the scanner to handle large lines
	maxScanTokenSize := options.MaxScanTokenSize
	if maxScanTokenSize <= 0 {
		maxScanTokenSize = 1024 * 1024 // Default to 1MB
	}
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)

	// Create record pool if enabled
	var recordPool *sync.Pool
	if options.UseRecordPool {
		recordPool = &sync.Pool{
			New: func() interface{} {
				return new(model.PafRecord)
			},
		}
	}

	// Set adaptive chunk parameters
	minChunkSize := options.MinChunkSize
	if minChunkSize <= 0 {
		minChunkSize = 1000 // Default minimum
	}

	maxChunkSize := options.MaxChunkSize
	if maxChunkSize <= 0 {
		maxChunkSize = 10000 // Default maximum
	}

	return &PafReader{
		file:          file,
		scanner:       scanner,
		buffer:        make([]model.PafRecord, 0, minChunkSize),
		recordPool:    recordPool,
		adaptiveChunk: options.AdaptiveChunk,
		minChunkSize:  minChunkSize,
		maxChunkSize:  maxChunkSize,
	}, nil
}

// ReadChunk reads a chunk of PAF records with optimized memory allocation
func (r *PafReader) ReadChunk(requestedChunkSize int) (interface{}, error) {
	// Determine actual chunk size based on adaptive settings
	chunkSize := requestedChunkSize
	if r.adaptiveChunk {
		// If adaptive chunking is enabled, use the requested size but constrain it
		// between min and max chunk sizes
		if chunkSize < r.minChunkSize {
			chunkSize = r.minChunkSize
		} else if chunkSize > r.maxChunkSize {
			chunkSize = r.maxChunkSize
		}
	}

	// Pre-allocate the chunk with capacity for the expected number of records
	chunk := make([]model.PafRecord, 0, chunkSize)
	count := 0

	for r.scanner.Scan() && count < chunkSize {
		line := r.scanner.Text()
		if line == "" {
			continue
		}

		// Split the line into fields (reuse the same slice if possible)
		fields := strings.Split(line, "\t")

		// Create a new record, using the object pool if available
		var record *model.PafRecord
		var err error

		if r.recordPool != nil {
			// Get a record from the pool
			record = r.recordPool.Get().(*model.PafRecord)
			// Parse the fields into the record
			record, err = model.NewPafRecord(fields)
		} else {
			// Create a new record
			record, err = model.NewPafRecord(fields)
		}

		if err != nil {
			return nil, fmt.Errorf("failed to parse PAF record: %w", err)
		}

		// Add the record to the chunk
		chunk = append(chunk, *record)

		// If using object pool, return the record to the pool after copying its data
		if r.recordPool != nil {
			r.recordPool.Put(record)
		}

		count++
	}

	if err := r.scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading PAF file: %w", err)
	}

	if len(chunk) == 0 {
		return nil, nil // EOF
	}

	return chunk, nil
}

// Close closes the reader and releases resources
func (r *PafReader) Close() error {
	// Clear the buffer to help with garbage collection
	r.buffer = nil

	// Close the file
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// MappingReader reads mapping files
type MappingReader struct {
	filename string
}

// NewMappingReader creates a new mapping reader
func NewMappingReader(filename string) *MappingReader {
	return &MappingReader{
		filename: filename,
	}
}

// ReadChunk reads the entire mapping database
// Since mapping files are typically small, we read the entire file at once
func (r *MappingReader) ReadChunk(chunkSize int) (interface{}, error) {
	db, err := model.NewMappingDatabase(r.filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read mapping database: %w", err)
	}
	return db, nil
}

// Close closes the reader (no-op for mapping reader)
func (r *MappingReader) Close() error {
	return nil
}
