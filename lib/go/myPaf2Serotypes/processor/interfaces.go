// Package processor contains the core business logic for PAF processing
package processor

import (
	"context"

	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/models"
)

// ScoreCalculator defines the interface for calculating scores from PAF hits
type ScoreCalculator interface {
	// Calculate processes PAF hits and calculates scores
	Calculate(ctx context.Context, records <-chan []models.PafHit) <-chan []models.PafScore
}

// SerotypeAssigner defines the interface for assigning serotypes based on scores
type SerotypeAssigner interface {
	// Assign processes scores and assigns serotypes
	Assign(ctx context.Context, scores <-chan []models.PafScore) <-chan models.Assignment
}

// Pipeline defines the interface for the complete processing pipeline
type Pipeline interface {
	// Run executes the complete pipeline
	Run(ctx context.Context) error
	// GetMetrics returns pipeline metrics
	GetMetrics() PipelineMetrics
}

// PipelineMetrics contains metrics about pipeline execution
type PipelineMetrics struct {
	RecordsProcessed   int64
	ScoresCalculated   int64
	AssignmentsMade    int64
	ProcessingDuration float64 // seconds
	ErrorCount         int64
}

// ProcessorConfig contains configuration for processors
type ProcessorConfig struct {
	MinScore      float64
	ScoreDistance float64
	NumWorkers    int
}