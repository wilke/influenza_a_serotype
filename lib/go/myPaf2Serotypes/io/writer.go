package io

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/errors"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/models"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/utils"
)

// FileResultWriter implements ResultWriter for writing to files
type FileResultWriter struct {
	outputDir    string
	sample       string
	logger       utils.Logger
	mu           sync.Mutex
	files        map[string]*os.File
	writtenReads map[models.Serotype]int
}

// NewFileResultWriter creates a new file result writer
func NewFileResultWriter(outputDir, sample string, logger utils.Logger) (*FileResultWriter, error) {
	if outputDir == "" {
		return nil, errors.ValidationError("FileResultWriter", "NewFileResultWriter", "outputDir cannot be empty")
	}
	if sample == "" {
		return nil, errors.ValidationError("FileResultWriter", "NewFileResultWriter", "sample cannot be empty")
	}
	if logger == nil {
		return nil, errors.ValidationError("FileResultWriter", "NewFileResultWriter", "logger cannot be nil")
	}

	return &FileResultWriter{
		outputDir:    outputDir,
		sample:       sample,
		logger:       logger,
		files:        make(map[string]*os.File),
		writtenReads: make(map[models.Serotype]int),
	}, nil
}

// WriteAssignments writes assignments to serotype-specific files
func (w *FileResultWriter) WriteAssignments(ctx context.Context, assignments <-chan models.Assignment) error {
	w.logger.Info("Starting assignment writing",
		utils.String("output_dir", w.outputDir),
		utils.String("sample", w.sample),
	)

	// Track assignments for summary
	var allAssignments []models.Assignment

	for assignment := range assignments {
		// Check context
		select {
		case <-ctx.Done():
			return errors.CanceledError("FileResultWriter", "WriteAssignments")
		default:
		}

		allAssignments = append(allAssignments, assignment)

		// Skip unassigned reads
		if !assignment.IsAssigned() {
			continue
		}

		// Get or create file for this serotype
		file, err := w.getOrCreateFile(assignment.Serotype)
		if err != nil {
			return err
		}

		// Write assignment
		if _, err := fmt.Fprintln(file, assignment.QueryName); err != nil {
			return errors.IOError("FileResultWriter", "WriteAssignments", 
				"failed to write assignment", err).
				WithContext("serotype", string(assignment.Serotype)).
				WithContext("query", assignment.QueryName)
		}

		w.writtenReads[assignment.Serotype]++
	}

	// Create summary
	summary := models.NewAssignmentSummary(allAssignments)

	// Write individual summary files
	if err := w.writeIndividualSummaries(allAssignments); err != nil {
		return err
	}

	// Write global summary
	if err := w.WriteSummary(ctx, summary); err != nil {
		return err
	}

	w.logger.Info("Completed assignment writing",
		utils.Int("total_assignments", len(allAssignments)),
		utils.Int("files_created", len(w.files)),
	)

	return nil
}

// WriteSummary writes the global summary file
func (w *FileResultWriter) WriteSummary(ctx context.Context, summary *models.AssignmentSummary) error {
	summaryPath := filepath.Join(w.outputDir, fmt.Sprintf("%s.summary.txt", w.sample))
	
	file, err := os.Create(summaryPath)
	if err != nil {
		return errors.IOError("FileResultWriter", "WriteSummary", 
			"failed to create summary file", err).
			WithContext("path", summaryPath)
	}
	defer file.Close()

	// Write header
	fmt.Fprintln(file, "# Global Summary")
	fmt.Fprintf(file, "# Generated: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "# Sample: %s\n\n", w.sample)

	// Write statistics
	fmt.Fprintf(file, "Total Reads: %d\n", summary.TotalReads)
	fmt.Fprintf(file, "Assigned Reads: %d (%.2f%%)\n", 
		summary.AssignedReads, 
		float64(summary.AssignedReads)/float64(summary.TotalReads)*100)
	fmt.Fprintf(file, "Ambiguous Reads: %d (%.2f%%)\n", 
		summary.AmbiguousReads,
		float64(summary.AmbiguousReads)/float64(summary.TotalReads)*100)
	fmt.Fprintf(file, "Unassigned Reads: %d (%.2f%%)\n", 
		summary.UnassignedReads,
		float64(summary.UnassignedReads)/float64(summary.TotalReads)*100)
	fmt.Fprintf(file, "Average Score: %.4f\n\n", summary.AverageScore)

	// Write serotype breakdown
	fmt.Fprintln(file, "## Serotype Distribution")
	fmt.Fprintln(file, "Serotype\tCount\tPercentage")

	sorted := summary.GetSortedSerotypes()
	for _, item := range sorted {
		percentage := float64(item.Count) / float64(summary.TotalReads) * 100
		fmt.Fprintf(file, "%s\t%d\t%.2f%%\n", item.Serotype, item.Count, percentage)
	}

	// Write file list
	fmt.Fprintln(file, "\n## Output Files")
	var fileList []string
	for serotype := range w.writtenReads {
		filename := fmt.Sprintf("%s.%s.txt", w.sample, serotype)
		fileList = append(fileList, filename)
	}
	sort.Strings(fileList)
	for _, filename := range fileList {
		fmt.Fprintln(file, filename)
	}

	w.logger.Info("Written global summary",
		utils.String("path", summaryPath),
		utils.Int("serotypes", len(sorted)),
	)

	return nil
}

// writeIndividualSummaries writes summary files for each serotype
func (w *FileResultWriter) writeIndividualSummaries(assignments []models.Assignment) error {
	// Group assignments by serotype
	serotypeAssignments := make(map[models.Serotype][]models.Assignment)
	for _, assignment := range assignments {
		if assignment.IsAssigned() {
			serotypeAssignments[assignment.Serotype] = append(
				serotypeAssignments[assignment.Serotype], assignment)
		}
	}

	// Write summary for each serotype
	for serotype, assigns := range serotypeAssignments {
		summaryPath := filepath.Join(w.outputDir, 
			fmt.Sprintf("%s.%s.summary.txt", w.sample, serotype))
		
		file, err := os.Create(summaryPath)
		if err != nil {
			return errors.IOError("FileResultWriter", "writeIndividualSummaries",
				"failed to create serotype summary file", err).
				WithContext("path", summaryPath)
		}
		defer file.Close()

		// Write header
		fmt.Fprintf(file, "# %s Summary\n", serotype)
		fmt.Fprintf(file, "# Sample: %s\n", w.sample)
		fmt.Fprintf(file, "# Total Reads: %d\n\n", len(assigns))

		// Calculate statistics
		var totalScore float64
		scoreDistribution := make(map[int]int) // score bucket -> count
		
		for _, assign := range assigns {
			totalScore += assign.AlignmentScore
			bucket := int(assign.AlignmentScore * 100) // 0.95 -> 95
			scoreDistribution[bucket]++
		}

		avgScore := totalScore / float64(len(assigns))
		fmt.Fprintf(file, "Average Score: %.4f\n\n", avgScore)

		// Write score distribution
		fmt.Fprintln(file, "## Score Distribution")
		fmt.Fprintln(file, "Score Range\tCount")
		
		// Sort buckets
		var buckets []int
		for bucket := range scoreDistribution {
			buckets = append(buckets, bucket)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(buckets)))

		for _, bucket := range buckets {
			minScore := float64(bucket) / 100
			maxScore := float64(bucket+1) / 100
			fmt.Fprintf(file, "%.2f-%.2f\t%d\n", 
				minScore, maxScore, scoreDistribution[bucket])
		}
	}

	return nil
}

// getOrCreateFile gets or creates a file for a serotype
func (w *FileResultWriter) getOrCreateFile(serotype models.Serotype) (*os.File, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if file already exists
	if file, ok := w.files[string(serotype)]; ok {
		return file, nil
	}

	// Create new file
	filename := fmt.Sprintf("%s.%s.txt", w.sample, serotype)
	filepath := filepath.Join(w.outputDir, filename)

	file, err := os.Create(filepath)
	if err != nil {
		return nil, errors.IOError("FileResultWriter", "getOrCreateFile",
			"failed to create output file", err).
			WithContext("path", filepath).
			WithContext("serotype", string(serotype))
	}

	w.files[string(serotype)] = file
	w.logger.Debug("Created output file",
		utils.String("serotype", string(serotype)),
		utils.String("path", filepath),
	)

	return file, nil
}

// Close closes all open files
func (w *FileResultWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var errs []error
	for serotype, file := range w.files {
		if err := file.Close(); err != nil {
			errs = append(errs, errors.IOError("FileResultWriter", "Close",
				"failed to close file", err).
				WithContext("serotype", serotype))
		}
	}

	w.files = make(map[string]*os.File)

	if len(errs) > 0 {
		errList := errors.NewErrorList()
		for _, err := range errs {
			errList.Add(err)
		}
		return errList
	}

	return nil
}

// BufferedWriter wraps a writer with buffering
type BufferedWriter struct {
	writer      ResultWriter
	assignments chan models.Assignment
	bufferSize  int
	wg          sync.WaitGroup
}

// NewBufferedWriter creates a new buffered writer
func NewBufferedWriter(writer ResultWriter, bufferSize int) *BufferedWriter {
	if bufferSize <= 0 {
		bufferSize = 1000
	}

	return &BufferedWriter{
		writer:      writer,
		assignments: make(chan models.Assignment, bufferSize),
		bufferSize:  bufferSize,
	}
}

// Write adds an assignment to the buffer
func (b *BufferedWriter) Write(assignment models.Assignment) {
	b.assignments <- assignment
}

// Start starts the background writer
func (b *BufferedWriter) Start(ctx context.Context) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		_ = b.writer.WriteAssignments(ctx, b.assignments)
	}()
}

// Close flushes remaining assignments and closes the writer
func (b *BufferedWriter) Close() error {
	close(b.assignments)
	b.wg.Wait()
	return b.writer.Close()
}