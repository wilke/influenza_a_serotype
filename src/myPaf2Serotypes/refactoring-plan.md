# MyPafProcessor Production Refactoring Plan

## Overview
This document outlines the step-by-step plan to refactor myPafProcessor from prototype to production-ready code.

## Refactoring Phases

### Phase 1: Foundation (Week 1)
**Goal**: Establish proper project structure and configuration management

#### Step 1.1: Create Package Structure
```
src/myPaf2Serotypes/
├── cmd/
│   └── myPaf2Serotypes/
│       └── main.go          # CLI entry point only
├── internal/
│   ├── config/
│   │   ├── config.go        # Configuration struct and loading
│   │   └── config_test.go
│   ├── models/
│   │   ├── paf.go          # PAF-related structs
│   │   ├── mapping.go      # Mapping-related structs
│   │   └── assignment.go   # Assignment-related structs
│   ├── processor/
│   │   ├── processor.go    # Main processing logic
│   │   ├── calculator.go   # Score calculation
│   │   ├── assigner.go     # Serotype assignment
│   │   └── processor_test.go
│   ├── io/
│   │   ├── reader.go       # File reading interfaces
│   │   ├── writer.go       # File writing interfaces
│   │   └── io_test.go
│   └── utils/
│       ├── logger.go       # Logging utilities
│       └── utils.go        # Common utilities
├── pkg/
│   └── pafprocessor/       # Public API if needed
├── configs/
│   └── default.yaml        # Default configuration
└── docs/
    └── api.md             # API documentation
```

#### Step 1.2: Extract Configuration
```go
// internal/config/config.go
type Config struct {
    // Input/Output
    PAFFile      string
    MappingFile  string
    OutputDir    string
    
    // Processing
    MinScore     float64
    ScoreDistance float64
    NumWorkers   int
    
    // Features
    VisualizePlot bool
    Sample       string
    
    // Logging
    LogLevel     string
    LogFormat    string // "text" or "json"
    
    // Performance
    ChannelBufferSize int
}

type ConfigLoader interface {
    Load() (*Config, error)
}
```

#### Step 1.3: Implement Structured Logging
```go
// internal/utils/logger.go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    With(fields ...Field) Logger
}

// Use zerolog or zap for implementation
```

### Phase 2: Core Refactoring (Week 2)
**Goal**: Extract business logic and implement clean architecture

#### Step 2.1: Define Core Interfaces
```go
// internal/processor/interfaces.go
type PAFReader interface {
    Read(ctx context.Context, path string) (<-chan []models.PafHit, <-chan error)
}

type MappingLoader interface {
    Load(ctx context.Context, path string) (models.Mapping, error)
}

type ScoreCalculator interface {
    Calculate(ctx context.Context, records <-chan []models.PafHit, mapping models.Mapping) <-chan []models.PafScore
}

type SerotypeAssigner interface {
    Assign(ctx context.Context, scores <-chan []models.PafScore, config AssignConfig) <-chan models.Assignment
}

type ResultWriter interface {
    Write(ctx context.Context, assignments <-chan models.Assignment, outputDir string) error
}
```

#### Step 2.2: Extract Models
```go
// internal/models/paf.go
type PafHit struct {
    QueryName            string
    QueryLength          int
    QueryStart           int
    QueryEnd             int
    Strand               string
    TargetName           string
    TargetLength         int
    TargetStart          int
    TargetEnd            int
    NumResidueMatches    int
    AlignmentBlockLength int
    MappingQuality       int
    Serotype             Serotype
}

// internal/models/mapping.go
type MappingEntry struct {
    Serotype       Serotype
    Segment        Segment
    OrganismName   string
    Host           string
    CollectionDate time.Time
}

type Mapping map[Accession]MappingEntry
```

#### Step 2.3: Implement Clean Processor
```go
// internal/processor/processor.go
type Processor struct {
    config      *config.Config
    logger      utils.Logger
    reader      PAFReader
    mapper      MappingLoader
    calculator  ScoreCalculator
    assigner    SerotypeAssigner
    writer      ResultWriter
}

func (p *Processor) Process(ctx context.Context) error {
    // Implement with proper error handling and context cancellation
}
```

### Phase 3: Error Handling & Resilience (Week 3)
**Goal**: Implement robust error handling and recovery

#### Step 3.1: Define Custom Errors
```go
// internal/errors/errors.go
type ErrorType int

const (
    ErrTypeValidation ErrorType = iota
    ErrTypeIO
    ErrTypeProcessing
    ErrTypeConfiguration
)

type ProcessorError struct {
    Type    ErrorType
    Message string
    Cause   error
    Context map[string]interface{}
}

func (e ProcessorError) Error() string {
    return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}
```

#### Step 3.2: Implement Circuit Breaker
```go
// internal/utils/circuitbreaker.go
type CircuitBreaker interface {
    Execute(fn func() error) error
}
```

#### Step 3.3: Add Retry Logic
```go
// internal/utils/retry.go
func RetryWithBackoff(ctx context.Context, fn func() error, opts RetryOptions) error {
    // Implement exponential backoff
}
```

### Phase 4: Concurrency & Performance (Week 4)
**Goal**: Optimize performance and resource usage

#### Step 4.1: Implement Worker Pool
```go
// internal/processor/workerpool.go
type WorkerPool struct {
    size    int
    jobs    chan Job
    results chan Result
    errors  chan error
    wg      sync.WaitGroup
}

func (wp *WorkerPool) Start(ctx context.Context) {
    for i := 0; i < wp.size; i++ {
        go wp.worker(ctx, i)
    }
}
```

#### Step 4.2: Add Memory Pooling
```go
// internal/utils/pool.go
var pafHitPool = sync.Pool{
    New: func() interface{} {
        return &models.PafHit{}
    },
}
```

#### Step 4.3: Implement Metrics Collection
```go
// internal/metrics/metrics.go
type Metrics struct {
    ProcessedRecords   prometheus.Counter
    ProcessingDuration prometheus.Histogram
    ErrorCount         prometheus.Counter
    // ... other metrics
}
```

### Phase 5: Testing & Documentation (Week 5)
**Goal**: Comprehensive testing and documentation

#### Step 5.1: Unit Tests
- Achieve >80% code coverage
- Use table-driven tests
- Mock external dependencies

#### Step 5.2: Integration Tests
```go
// integration/processor_test.go
func TestEndToEndProcessing(t *testing.T) {
    // Test complete pipeline with real files
}
```

#### Step 5.3: Benchmarks
```go
// internal/processor/processor_bench_test.go
func BenchmarkProcessor(b *testing.B) {
    // Benchmark different configurations
}
```

#### Step 5.4: Documentation
- API documentation with examples
- Architecture decision records (ADRs)
- Deployment guide
- Performance tuning guide

## Migration Strategy

### Step 1: Incremental Refactoring
1. Start with models extraction (non-breaking)
2. Create new package structure alongside old code
3. Gradually move functionality to new structure
4. Maintain backward compatibility during transition

### Step 2: Parallel Implementation
1. Keep old main.go functional
2. Create new cmd/myPaf2Serotypes/main.go
3. Test new implementation thoroughly
4. Switch over when ready

### Step 3: Feature Flags
```go
if config.UseNewProcessor {
    return newProcessor.Process(ctx)
} else {
    return legacyProcess()
}
```

## Key Improvements Summary

1. **Architecture**: Clean, layered architecture with clear separation of concerns
2. **Error Handling**: Consistent, contextual error handling with recovery
3. **Configuration**: Flexible configuration with multiple sources
4. **Logging**: Structured logging with levels and context
5. **Testing**: Comprehensive test coverage with mocks
6. **Performance**: Optimized with worker pools and memory pooling
7. **Monitoring**: Built-in metrics and health checks
8. **Documentation**: Complete API and operational documentation
9. **Deployment**: Container-ready with health endpoints
10. **Maintenance**: Modular design for easy updates

## Success Criteria

- [ ] All tests pass with >80% coverage
- [ ] No global variables or state
- [ ] Proper error handling throughout
- [ ] Configurable via files and environment
- [ ] Structured logging implemented
- [ ] Performance benchmarks established
- [ ] Documentation complete
- [ ] CI/CD pipeline ready
- [ ] Monitoring and alerting configured
- [ ] Load tested for production workloads

## Timeline
- **Week 1**: Foundation and structure
- **Week 2**: Core refactoring
- **Week 3**: Error handling and resilience
- **Week 4**: Performance optimization
- **Week 5**: Testing and documentation
- **Week 6**: Migration and deployment

## Next Steps
1. Review and approve plan
2. Set up new package structure
3. Begin Phase 1 implementation
4. Create tracking issues for each phase