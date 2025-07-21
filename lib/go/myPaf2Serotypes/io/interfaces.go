// Package io provides input/output interfaces and implementations
package io

import (
	"context"
	"io"

	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/models"
)

// PAFReader defines the interface for reading PAF files
type PAFReader interface {
	// Read reads a PAF file and streams records grouped by query name
	Read(ctx context.Context) (<-chan []models.PafHit, <-chan error)
	// Close closes the reader and releases resources
	Close() error
}

// ResultWriter defines the interface for writing results
type ResultWriter interface {
	// WriteAssignments writes assignments to output files
	WriteAssignments(ctx context.Context, assignments <-chan models.Assignment) error
	// WriteSummary writes a global summary file
	WriteSummary(ctx context.Context, summary *models.AssignmentSummary) error
	// Close closes the writer and releases resources
	Close() error
}

// FileReader defines a generic file reader interface
type FileReader interface {
	io.ReadCloser
	Name() string
}

// FileWriter defines a generic file writer interface  
type FileWriter interface {
	io.WriteCloser
	Name() string
}

// FileSystem defines filesystem operations
type FileSystem interface {
	// Open opens a file for reading
	Open(path string) (FileReader, error)
	// Create creates a file for writing
	Create(path string) (FileWriter, error)
	// MkdirAll creates directories
	MkdirAll(path string, perm uint32) error
	// Exists checks if a path exists
	Exists(path string) bool
}