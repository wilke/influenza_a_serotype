package processor

import (
	"context"
	"sync"
	"time"

	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/config"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/errors"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/io"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/models"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/utils"
)

// DefaultPipeline implements the complete processing pipeline
type DefaultPipeline struct {
	config     *config.Config
	reader     io.PAFReader
	calculator ScoreCalculator
	assigner   SerotypeAssigner
	writer     io.ResultWriter
	logger     utils.Logger
	metrics    PipelineMetrics
	mu         sync.Mutex
}

// NewDefaultPipeline creates a new pipeline with all components
func NewDefaultPipeline(cfg *config.Config, logger utils.Logger) (*DefaultPipeline, error) {
	if cfg == nil {
		return nil, errors.ValidationError("DefaultPipeline", "NewDefaultPipeline", "config cannot be nil")
	}
	if logger == nil {
		return nil, errors.ValidationError("DefaultPipeline", "NewDefaultPipeline", "logger cannot be nil")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(err, errors.ErrTypeConfiguration, "DefaultPipeline", "NewDefaultPipeline", 
			"invalid configuration")
	}

	// Load mapping file
	logger.Info("Loading mapping file", utils.String("file", cfg.MappingFile))
	mapping, err := models.LoadMappingFile(cfg.MappingFile)
	if err != nil {
		return nil, errors.IOError("DefaultPipeline", "NewDefaultPipeline", 
			"failed to load mapping file", err)
	}

	// Create components
	reader, err := io.NewPAFFileReader(cfg.PAFFile, mapping, logger, cfg.FileBufferSize)
	if err != nil {
		return nil, err
	}

	calculator, err := NewDefaultScoreCalculator(mapping, cfg.NumWorkers, logger)
	if err != nil {
		return nil, err
	}

	processorConfig := ProcessorConfig{
		MinScore:      cfg.MinScore,
		ScoreDistance: cfg.ScoreDistance,
		NumWorkers:    cfg.NumWorkers,
	}
	assigner, err := NewDefaultSerotypeAssigner(processorConfig, logger)
	if err != nil {
		return nil, err
	}

	writer, err := io.NewFileResultWriter(cfg.OutputDir, cfg.Sample, logger)
	if err != nil {
		return nil, err
	}

	return &DefaultPipeline{
		config:     cfg,
		reader:     reader,
		calculator: calculator,
		assigner:   assigner,
		writer:     writer,
		logger:     logger,
	}, nil
}

// Run executes the complete pipeline
func (p *DefaultPipeline) Run(ctx context.Context) error {
	startTime := time.Now()
	p.logger.Info("Starting pipeline execution",
		utils.String("paf_file", p.config.PAFFile),
		utils.String("output_dir", p.config.OutputDir),
		utils.String("sample", p.config.Sample),
	)

	// Create context with timeout if configured
	if p.config.ProcessTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.ProcessTimeout)
		defer cancel()
	}

	// Execute pipeline stages
	errChan := make(chan error, 1)
	
	go func() {
		defer close(errChan)
		
		// Stage 1: Read PAF file
		records, readErrors := p.reader.Read(ctx)
		
		// Monitor read errors
		go func() {
			for err := range readErrors {
				if err != nil {
					p.logger.Error("PAF reading error", utils.Err(err))
					p.incrementErrorCount()
					// Don't return - continue processing what we can
				}
			}
		}()

		// Stage 2: Calculate scores
		scores := p.calculator.Calculate(ctx, records)
		
		// Stage 3: Assign serotypes
		assignments := p.assigner.Assign(ctx, scores)
		
		// Stage 4: Write results
		if err := p.writer.WriteAssignments(ctx, assignments); err != nil {
			errChan <- err
			return
		}
	}()

	// Wait for completion or error
	select {
	case err := <-errChan:
		if err != nil {
			return errors.Wrap(err, errors.ErrTypeProcessing, "DefaultPipeline", "Run",
				"pipeline execution failed")
		}
	case <-ctx.Done():
		return errors.TimeoutError("DefaultPipeline", "Run", p.config.ProcessTimeout)
	}

	// Update final metrics
	p.updateMetrics(startTime)

	p.logger.Info("Pipeline execution completed",
		utils.Duration("duration", time.Since(startTime)),
		utils.Int64("records_processed", p.metrics.RecordsProcessed),
		utils.Int64("assignments_made", p.metrics.AssignmentsMade),
	)

	return nil
}

// GetMetrics returns pipeline metrics
func (p *DefaultPipeline) GetMetrics() PipelineMetrics {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.metrics
}

// updateMetrics updates pipeline metrics
func (p *DefaultPipeline) updateMetrics(startTime time.Time) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Get metrics from components
	if calc, ok := p.calculator.(*DefaultScoreCalculator); ok {
		p.metrics.RecordsProcessed, p.metrics.ScoresCalculated = calc.GetMetrics()
	}

	if assign, ok := p.assigner.(*DefaultSerotypeAssigner); ok {
		_, p.metrics.AssignmentsMade, _, _ = assign.GetMetrics()
	}

	p.metrics.ProcessingDuration = time.Since(startTime).Seconds()
}

// incrementErrorCount increments the error counter
func (p *DefaultPipeline) incrementErrorCount() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.metrics.ErrorCount++
}

// Close closes all pipeline components
func (p *DefaultPipeline) Close() error {
	errList := errors.NewErrorList()

	if p.reader != nil {
		if err := p.reader.Close(); err != nil {
			errList.Add(err)
		}
	}

	if p.writer != nil {
		if err := p.writer.Close(); err != nil {
			errList.Add(err)
		}
	}

	if errList.HasErrors() {
		return errList
	}

	return nil
}

// PipelineBuilder provides a fluent interface for building pipelines
type PipelineBuilder struct {
	config     *config.Config
	reader     io.PAFReader
	calculator ScoreCalculator
	assigner   SerotypeAssigner
	writer     io.ResultWriter
	logger     utils.Logger
	mapping    models.Mapping
}

// NewPipelineBuilder creates a new pipeline builder
func NewPipelineBuilder() *PipelineBuilder {
	return &PipelineBuilder{}
}

// WithConfig sets the configuration
func (b *PipelineBuilder) WithConfig(cfg *config.Config) *PipelineBuilder {
	b.config = cfg
	return b
}

// WithLogger sets the logger
func (b *PipelineBuilder) WithLogger(logger utils.Logger) *PipelineBuilder {
	b.logger = logger
	return b
}

// WithMapping sets the mapping
func (b *PipelineBuilder) WithMapping(mapping models.Mapping) *PipelineBuilder {
	b.mapping = mapping
	return b
}

// WithReader sets a custom reader
func (b *PipelineBuilder) WithReader(reader io.PAFReader) *PipelineBuilder {
	b.reader = reader
	return b
}

// WithCalculator sets a custom calculator
func (b *PipelineBuilder) WithCalculator(calculator ScoreCalculator) *PipelineBuilder {
	b.calculator = calculator
	return b
}

// WithAssigner sets a custom assigner
func (b *PipelineBuilder) WithAssigner(assigner SerotypeAssigner) *PipelineBuilder {
	b.assigner = assigner
	return b
}

// WithWriter sets a custom writer
func (b *PipelineBuilder) WithWriter(writer io.ResultWriter) *PipelineBuilder {
	b.writer = writer
	return b
}

// Build creates the pipeline
func (b *PipelineBuilder) Build() (*CustomPipeline, error) {
	// Validate required components
	if b.config == nil {
		return nil, errors.ValidationError("PipelineBuilder", "Build", "config is required")
	}
	if b.logger == nil {
		return nil, errors.ValidationError("PipelineBuilder", "Build", "logger is required")
	}

	// Load mapping if not provided
	if b.mapping == nil {
		mapping, err := models.LoadMappingFile(b.config.MappingFile)
		if err != nil {
			return nil, err
		}
		b.mapping = mapping
	}

	// Create default components if not provided
	if b.reader == nil {
		reader, err := io.NewPAFFileReader(b.config.PAFFile, b.mapping, b.logger, b.config.FileBufferSize)
		if err != nil {
			return nil, err
		}
		b.reader = reader
	}

	if b.calculator == nil {
		calculator, err := NewDefaultScoreCalculator(b.mapping, b.config.NumWorkers, b.logger)
		if err != nil {
			return nil, err
		}
		b.calculator = calculator
	}

	if b.assigner == nil {
		processorConfig := ProcessorConfig{
			MinScore:      b.config.MinScore,
			ScoreDistance: b.config.ScoreDistance,
			NumWorkers:    b.config.NumWorkers,
		}
		assigner, err := NewDefaultSerotypeAssigner(processorConfig, b.logger)
		if err != nil {
			return nil, err
		}
		b.assigner = assigner
	}

	if b.writer == nil {
		writer, err := io.NewFileResultWriter(b.config.OutputDir, b.config.Sample, b.logger)
		if err != nil {
			return nil, err
		}
		b.writer = writer
	}

	return &CustomPipeline{
		config:     b.config,
		reader:     b.reader,
		calculator: b.calculator,
		assigner:   b.assigner,
		writer:     b.writer,
		logger:     b.logger,
	}, nil
}

// CustomPipeline is a pipeline with custom components
type CustomPipeline struct {
	config     *config.Config
	reader     io.PAFReader
	calculator ScoreCalculator
	assigner   SerotypeAssigner
	writer     io.ResultWriter
	logger     utils.Logger
	metrics    PipelineMetrics
	mu         sync.Mutex
}

// Run executes the custom pipeline
func (p *CustomPipeline) Run(ctx context.Context) error {
	// Implementation is same as DefaultPipeline.Run
	return (&DefaultPipeline{
		config:     p.config,
		reader:     p.reader,
		calculator: p.calculator,
		assigner:   p.assigner,
		writer:     p.writer,
		logger:     p.logger,
	}).Run(ctx)
}

// GetMetrics returns pipeline metrics
func (p *CustomPipeline) GetMetrics() PipelineMetrics {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.metrics
}