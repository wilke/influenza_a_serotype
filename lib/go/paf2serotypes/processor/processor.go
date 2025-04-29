package processor

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/model"
)

// Processor interface for processing chunks of data
type Processor interface {
	Process(chunk interface{}, mapping interface{}) (interface{}, error)
}

// PafProcessor processes PAF records
type PafProcessor struct {
	scoreThresh     float64
	ambiguityThresh float64
}

// NewPafProcessor creates a new PAF processor
func NewPafProcessor(scoreThresh, ambiguityThresh float64) *PafProcessor {
	return &PafProcessor{
		scoreThresh:     scoreThresh,
		ambiguityThresh: ambiguityThresh,
	}
}

// Process processes a chunk of PAF records
func (p *PafProcessor) Process(chunk interface{}, mapping interface{}) (interface{}, error) {
	records, ok := chunk.([]model.PafRecord)
	if !ok {
		return nil, fmt.Errorf("expected []model.PafRecord, got %T", chunk)
	}

	db, ok := mapping.(*model.MappingDatabase)
	if !ok {
		return nil, fmt.Errorf("expected *model.MappingDatabase, got %T", mapping)
	}

	// Calculate alignment scores
	scores, err := model.CalculateScores(records, db)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate scores: %w", err)
	}

	// Assign serotypes
	// Note: The AssignSerotypes function has been updated to match the R implementation's order of operations:
	// 1. First determines if a read is ambiguous by comparing ALL serotype scores
	// 2. Then filters out reads where the maximum score is below the threshold
	summaries := model.AssignSerotypes(scores, p.scoreThresh, p.ambiguityThresh)

	fmt.Printf("Processed %d records, found %d scores and %d serotypes\n", len(records), len(scores), len(summaries))
	// fmt.Printf("Records: %v \n", records)
	// fmt.Printf("Summaries: %v \n", summaries)
	// os.Exit(1)

	return summaries, nil
}

// ChunkProcessor processes chunks of data in parallel with dynamic worker pool sizing
type ChunkProcessor struct {
	processor        Processor
	workers          int
	maxWorkers       int
	dynamicWorkers   bool
	activeWorkers    int32 // Atomic counter for active workers
	resultPool       sync.Pool
	useObjectPooling bool
}

// ChunkProcessorOptions contains options for configuring the chunk processor
type ChunkProcessorOptions struct {
	Workers          int  // Base number of workers
	MaxWorkers       int  // Maximum number of workers for dynamic scaling
	DynamicWorkers   bool // Whether to use dynamic worker pool sizing
	UseObjectPooling bool // Whether to use object pooling
}

// DefaultChunkProcessorOptions returns default options for the chunk processor
func DefaultChunkProcessorOptions() ChunkProcessorOptions {
	numCPU := runtime.NumCPU()
	return ChunkProcessorOptions{
		Workers:          numCPU,
		MaxWorkers:       numCPU * 2,
		DynamicWorkers:   true,
		UseObjectPooling: true,
	}
}

// NewChunkProcessor creates a new chunk processor with default options
func NewChunkProcessor(processor Processor, workers int) *ChunkProcessor {
	return NewChunkProcessorWithOptions(processor, ChunkProcessorOptions{
		Workers:          workers,
		MaxWorkers:       workers * 2,
		DynamicWorkers:   false,
		UseObjectPooling: false,
	})
}

// NewChunkProcessorWithOptions creates a new chunk processor with custom options
func NewChunkProcessorWithOptions(processor Processor, options ChunkProcessorOptions) *ChunkProcessor {
	cp := &ChunkProcessor{
		processor:        processor,
		workers:          options.Workers,
		maxWorkers:       options.MaxWorkers,
		dynamicWorkers:   options.DynamicWorkers,
		activeWorkers:    0,
		useObjectPooling: options.UseObjectPooling,
	}

	// Initialize result pool if object pooling is enabled
	if options.UseObjectPooling {
		cp.resultPool = sync.Pool{
			New: func() interface{} {
				// Create a new result object
				// The exact type depends on what the processor returns
				return nil // Will be initialized when used
			},
		}
	}

	return cp
}

// ProcessChunks processes multiple chunks in parallel with dynamic worker pool sizing
func (p *ChunkProcessor) ProcessChunks(chunks []interface{}, mapping interface{}) ([]interface{}, error) {
	// Pre-allocate results slice with exact capacity needed
	results := make([]interface{}, len(chunks))

	// Create buffered error channel to collect errors
	errChan := make(chan error, len(chunks))

	// Create wait group to track completion
	var wg sync.WaitGroup

	// Create job channel for worker pool
	jobs := make(chan jobInfo, len(chunks))

	// Determine initial worker count
	workerCount := p.workers
	if p.dynamicWorkers {
		// Get current system load to adjust worker count
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		// If memory usage is high, reduce worker count
		memUsagePercent := float64(memStats.Alloc) / float64(memStats.Sys)
		if memUsagePercent > 0.8 {
			// High memory usage, reduce workers
			workerCount = p.workers / 2
			if workerCount < 1 {
				workerCount = 1
			}
		} else if memUsagePercent < 0.5 {
			// Low memory usage, increase workers
			workerCount = p.workers * 2
			if workerCount > p.maxWorkers {
				workerCount = p.maxWorkers
			}
		}

		fmt.Printf("Dynamic worker pool: using %d workers (memory usage: %.1f%%)\n",
			workerCount, memUsagePercent*100)
	}

	// Start worker pool
	for w := 0; w < workerCount; w++ {
		go p.worker(jobs, results, errChan, mapping, &wg)
	}

	// Send jobs to the worker pool
	for i, chunk := range chunks {
		wg.Add(1)
		jobs <- jobInfo{
			index: i,
			chunk: chunk,
		}
	}

	// Close job channel after all jobs are sent
	close(jobs)

	// Wait for all jobs to complete
	wg.Wait()

	// Close error channel
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return results, nil
}

// jobInfo represents a processing job
type jobInfo struct {
	index int
	chunk interface{}
}

// worker processes jobs from the job channel
func (p *ChunkProcessor) worker(jobs <-chan jobInfo, results []interface{}, errChan chan<- error, mapping interface{}, wg *sync.WaitGroup) {
	for job := range jobs {
		// Track active workers for monitoring
		atomic.AddInt32(&p.activeWorkers, 1)

		// Process the chunk
		var result interface{}
		var err error

		if p.useObjectPooling {
			// Get a result object from the pool if possible
			pooledResult := p.resultPool.Get()
			if pooledResult != nil {
				// Process using the pooled object
				result, err = p.processor.Process(job.chunk, mapping)
				// Return the object to the pool when done
				p.resultPool.Put(pooledResult)
			} else {
				// Fall back to normal processing
				result, err = p.processor.Process(job.chunk, mapping)
			}
		} else {
			// Normal processing without object pooling
			result, err = p.processor.Process(job.chunk, mapping)
		}

		if err != nil {
			errChan <- fmt.Errorf("failed to process chunk %d: %w", job.index, err)
		} else {
			results[job.index] = result
		}

		// Decrement active workers counter
		atomic.AddInt32(&p.activeWorkers, -1)

		// Mark job as done
		wg.Done()
	}
}

// GetActiveWorkers returns the current number of active workers
func (p *ChunkProcessor) GetActiveWorkers() int {
	return int(atomic.LoadInt32(&p.activeWorkers))
}

// Process processes a single chunk using the underlying processor
// This method allows ChunkProcessor to implement the Processor interface
func (p *ChunkProcessor) Process(chunk interface{}, mapping interface{}) (interface{}, error) {
	// Simply delegate to the underlying processor
	return p.processor.Process(chunk, mapping)
}

// MergeResults merges multiple result sets
func MergeResults(results []interface{}) ([]model.SerotypeSummary, error) {
	var allSummaries []model.SerotypeSummary

	for _, result := range results {
		summaries, ok := result.([]model.SerotypeSummary)
		if !ok {
			return nil, fmt.Errorf("expected []model.SerotypeSummary, got %T", result)
		}
		allSummaries = append(allSummaries, summaries...)
	}

	return allSummaries, nil
}
