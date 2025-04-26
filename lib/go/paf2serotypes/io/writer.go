package io

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/model"
	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/visualization"
)

// Writer interface for writing output files
type Writer interface {
	Write(data interface{}) error
	Close() error
}

// ResultWriter writes results to files
type ResultWriter struct {
	outDir     string
	sampleName string
}

// NewResultWriter creates a new result writer
func NewResultWriter(outDir, sampleName string) (*ResultWriter, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &ResultWriter{
		outDir:     outDir,
		sampleName: sampleName,
	}, nil
}

// Write writes the results to files
func (w *ResultWriter) Write(data interface{}) error {
	summaries, ok := data.([]model.SerotypeSummary)
	if !ok {
		return fmt.Errorf("expected []model.SerotypeSummary, got %T", data)
	}

	// Write summary table
	if err := model.WriteSummary(summaries, w.outDir, w.sampleName); err != nil {
		return fmt.Errorf("failed to write summary table: %w", err)
	}

	// Write read lists by serotype
	if err := model.WriteReadLists(summaries, w.outDir, w.sampleName); err != nil {
		return fmt.Errorf("failed to write read lists: %w", err)
	}

	return nil
}

// Close closes the writer (no-op for result writer)
func (w *ResultWriter) Close() error {
	return nil
}

// VisualizationWriter writes visualization files
type VisualizationWriter struct {
	outDir     string
	sampleName string
}

// NewVisualizationWriter creates a new visualization writer
func NewVisualizationWriter(outDir, sampleName string) (*VisualizationWriter, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	return &VisualizationWriter{
		outDir:     outDir,
		sampleName: sampleName,
	}, nil
}

// Write writes the visualization files
func (w *VisualizationWriter) Write(data interface{}) error {
	summaries, ok := data.([]model.SerotypeSummary)
	if !ok {
		return fmt.Errorf("expected []model.SerotypeSummary, got %T", data)
	}

	// Generate a bar chart visualization
	if err := visualization.GenerateBarChart(summaries, w.outDir, w.sampleName); err != nil {
		return fmt.Errorf("failed to generate visualization: %w", err)
	}

	// Also write a CSV file with counts for external use
	outPath := filepath.Join(w.outDir, fmt.Sprintf("%s_serotype_counts.csv", w.sampleName))
	file, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create serotype counts file: %w", err)
	}
	defer file.Close()

	// Count serotypes
	counts := make(map[string]int)
	for _, summary := range summaries {
		if summary.ReadAssignment != "" {
			counts[summary.ReadAssignment]++
		}
	}

	// Write header
	if _, err := fmt.Fprintln(file, "serotype,count"); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write counts
	for serotype, count := range counts {
		if _, err := fmt.Fprintf(file, "%s,%d\n", serotype, count); err != nil {
			return fmt.Errorf("failed to write count: %w", err)
		}
	}

	return nil
}

// Close closes the writer (no-op for visualization writer)
func (w *VisualizationWriter) Close() error {
	return nil
}
