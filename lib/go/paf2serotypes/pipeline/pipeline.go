package pipeline

import (
	"context"
	"fmt"
	"sync"

	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/io"
	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/model"
	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/processor"
)

// PipelineOptions contains options for configuring the pipeline
type PipelineOptions struct {
	Workers        int  // Base number of workers
	MaxWorkers     int  // Maximum number of workers for dynamic scaling
	DynamicWorkers bool // Whether to use dynamic worker pool sizing
}

// Pipeline represents a processing pipeline
type Pipeline struct {
	stages         []Stage
	workers        int
	maxWorkers     int
	dynamicWorkers bool
}

// NewPipeline creates a new pipeline with a fixed worker count
func NewPipeline(workers int) *Pipeline {
	return &Pipeline{
		stages:         make([]Stage, 0),
		workers:        workers,
		maxWorkers:     workers,
		dynamicWorkers: false,
	}
}

// NewPipelineWithOptions creates a new pipeline with custom options
func NewPipelineWithOptions(options PipelineOptions) *Pipeline {
	return &Pipeline{
		stages:         make([]Stage, 0),
		workers:        options.Workers,
		maxWorkers:     options.MaxWorkers,
		dynamicWorkers: options.DynamicWorkers,
	}
}

// AddStage adds a stage to the pipeline
func (p *Pipeline) AddStage(stage Stage) *Pipeline {
	p.stages = append(p.stages, stage)
	return p
}

// Run runs the pipeline
func (p *Pipeline) Run(ctx context.Context) error {
	if len(p.stages) == 0 {
		return fmt.Errorf("pipeline has no stages")
	}

	// Create channels for each stage
	channels := make([]chan interface{}, len(p.stages)+1)

	// Determine buffer size based on worker count
	bufferSize := p.workers
	if p.dynamicWorkers && p.maxWorkers > p.workers {
		// If dynamic workers are enabled, use a larger buffer to accommodate potential scaling
		bufferSize = p.maxWorkers
	}

	for i := range channels {
		channels[i] = make(chan interface{}, bufferSize)
	}

	// Create error channel
	errChan := make(chan error, len(p.stages))

	// Start each stage
	var wg sync.WaitGroup
	for i, stage := range p.stages {
		wg.Add(1)
		go func(i int, stage Stage) {
			defer wg.Done()
			defer close(channels[i+1])

			if err := stage.Process(ctx, channels[i], channels[i+1]); err != nil {
				errChan <- fmt.Errorf("stage %d failed: %w", i, err)
			}
		}(i, stage)
	}

	// Close error channel when all stages are done
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Check for errors
	for err := range errChan {
		return err
	}

	return nil
}

// Stage represents a pipeline stage
type Stage interface {
	Process(ctx context.Context, in <-chan interface{}, out chan<- interface{}) error
}

// ReadStage reads data from files
type ReadStage struct {
	reader    io.Reader
	chunkSize int
}

// NewReadStage creates a new read stage
func NewReadStage(reader io.Reader, chunkSize int) *ReadStage {
	return &ReadStage{
		reader:    reader,
		chunkSize: chunkSize,
	}
}

// Process processes the read stage
func (s *ReadStage) Process(ctx context.Context, in <-chan interface{}, out chan<- interface{}) error {
	defer s.reader.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			chunk, err := s.reader.ReadChunk(s.chunkSize)
			if err != nil {
				return fmt.Errorf("failed to read chunk: %w", err)
			}

			if chunk == nil {
				return nil // EOF
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- chunk:
				// Chunk sent successfully
			}
		}
	}
}

// ProcessStage processes data
type ProcessStage struct {
	processor processor.Processor
	mapping   interface{}
}

// NewProcessStage creates a new process stage
func NewProcessStage(proc processor.Processor, mapping interface{}) *ProcessStage {
	return &ProcessStage{
		processor: proc,
		mapping:   mapping,
	}
}

// NewProcessStageWithProcessor creates a new process stage with a custom processor
// This allows using a ChunkProcessor directly
func NewProcessStageWithProcessor(proc interface{}, mapping interface{}) *ProcessStage {
	return &ProcessStage{
		processor: proc.(processor.Processor),
		mapping:   mapping,
	}
}

// Process processes the process stage
func (s *ProcessStage) Process(ctx context.Context, in <-chan interface{}, out chan<- interface{}) error {
	for chunk := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			result, err := s.processor.Process(chunk, s.mapping)
			if err != nil {
				return fmt.Errorf("failed to process chunk: %w", err)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- result:
				// Result sent successfully
			}
		}
	}

	return nil
}

// MergeStage merges results from multiple chunks
type MergeStage struct{}

// NewMergeStage creates a new merge stage
func NewMergeStage() *MergeStage {
	return &MergeStage{}
}

// Process processes the merge stage
func (s *MergeStage) Process(ctx context.Context, in <-chan interface{}, out chan<- interface{}) error {
	var allSummaries []model.SerotypeSummary

	for result := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			summaries, ok := result.([]model.SerotypeSummary)
			if !ok {
				return fmt.Errorf("expected []model.SerotypeSummary, got %T", result)
			}
			allSummaries = append(allSummaries, summaries...)
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case out <- allSummaries:
		// Result sent successfully
	}

	return nil
}

// WriteStage writes results to files
type WriteStage struct {
	writer io.Writer
}

// NewWriteStage creates a new write stage
func NewWriteStage(writer io.Writer) *WriteStage {
	return &WriteStage{
		writer: writer,
	}
}

// Process processes the write stage
func (s *WriteStage) Process(ctx context.Context, in <-chan interface{}, out chan<- interface{}) error {
	defer s.writer.Close()

	for data := range in {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := s.writer.Write(data); err != nil {
				return fmt.Errorf("failed to write data: %w", err)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- data:
				// Data forwarded successfully
			}
		}
	}

	return nil
}
