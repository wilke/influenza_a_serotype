# MyPafProcessor Architecture Design

## Executive Summary
This document outlines the target architecture for the production-ready myPafProcessor, transforming it from a monolithic prototype to a modular, scalable, and maintainable system.

## Current State vs Target State

### Current State (Prototype)
```
┌─────────────────────────────────────┐
│         main.go (849 lines)         │
│  ┌─────────────────────────────┐    │
│  │  Global Variables           │    │
│  │  - logger                   │    │
│  │  - debug flag               │    │
│  │  - SegmentMap               │    │
│  └─────────────────────────────┘    │
│  ┌─────────────────────────────┐    │
│  │  All Functions Mixed        │    │
│  │  - I/O Operations           │    │
│  │  - Business Logic           │    │
│  │  - Data Structures          │    │
│  └─────────────────────────────┘    │
└─────────────────────────────────────┘
```

### Target State (Production)
```
┌──────────────────────────────────────────────────────────┐
│                    Application Layer                      │
├──────────────────────────────────────────────────────────┤
│  ┌────────────┐  ┌────────────┐  ┌──────────────────┐  │
│  │    CLI     │  │    API     │  │   Health Check   │  │
│  │  (cmd/)    │  │  (pkg/)    │  │   (/healthz)     │  │
│  └────────────┘  └────────────┘  └──────────────────┘  │
├──────────────────────────────────────────────────────────┤
│                    Business Layer                         │
├──────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────┐    │
│  │              Processor Pipeline                   │    │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌──────────┐ │    │
│  │  │Reader  │→│Calculator│→│Assigner│→│Writer    │ │    │
│  │  └────────┘ └────────┘ └────────┘ └──────────┘ │    │
│  └─────────────────────────────────────────────────┘    │
├──────────────────────────────────────────────────────────┤
│                    Core Layer                             │
├──────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌────────────────────┐    │
│  │  Models  │  │  Config  │  │   Infrastructure   │    │
│  │  - PAF   │  │  - YAML  │  │   - Logging        │    │
│  │  - Map   │  │  - ENV   │  │   - Metrics        │    │
│  │  - Score │  │  - CLI   │  │   - Error Handling │    │
│  └──────────┘  └──────────┘  └────────────────────┘    │
└──────────────────────────────────────────────────────────┘
```

## Core Design Principles

### 1. Separation of Concerns
- **Models**: Pure data structures with validation
- **Business Logic**: Processing algorithms isolated from I/O
- **Infrastructure**: Cross-cutting concerns (logging, metrics, config)
- **Interfaces**: Clear contracts between components

### 2. Dependency Injection
```go
type Processor struct {
    config     *Config
    logger     Logger
    reader     PAFReader
    calculator ScoreCalculator
    assigner   SerotypeAssigner
    writer     ResultWriter
    metrics    MetricsCollector
}

func NewProcessor(deps Dependencies) *Processor {
    return &Processor{
        config:     deps.Config,
        logger:     deps.Logger,
        // ... inject all dependencies
    }
}
```

### 3. Interface-Based Design
```go
// All major components defined as interfaces
type PAFReader interface {
    Read(ctx context.Context, path string) (<-chan []PafHit, <-chan error)
}

type ScoreCalculator interface {
    Calculate(ctx context.Context, records <-chan []PafHit, mapping Mapping) <-chan []PafScore
}
```

### 4. Context-Aware Operations
```go
// All operations accept context for cancellation and timeout
func (p *Processor) Process(ctx context.Context) error {
    ctx, cancel := context.WithTimeout(ctx, p.config.ProcessTimeout)
    defer cancel()
    
    // Check context throughout processing
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        // Continue processing
    }
}
```

## Component Architecture

### 1. Configuration Management
```go
// Layered configuration with precedence: CLI > ENV > File > Defaults
type ConfigManager struct {
    fileLoader FileConfigLoader
    envLoader  EnvConfigLoader
    cliLoader  CLIConfigLoader
}

func (cm *ConfigManager) Load() (*Config, error) {
    config := DefaultConfig()
    
    // Layer configurations
    if err := cm.fileLoader.Load(config); err != nil {
        return nil, err
    }
    if err := cm.envLoader.Load(config); err != nil {
        return nil, err
    }
    if err := cm.cliLoader.Load(config); err != nil {
        return nil, err
    }
    
    return config, config.Validate()
}
```

### 2. Pipeline Architecture
```go
// Composable pipeline stages
type Pipeline struct {
    stages []Stage
}

type Stage interface {
    Process(ctx context.Context, input <-chan interface{}) (<-chan interface{}, <-chan error)
}

// Example usage
pipeline := NewPipeline(
    NewReaderStage(config),
    NewValidationStage(),
    NewCalculatorStage(),
    NewAssignerStage(),
    NewWriterStage(),
)

err := pipeline.Execute(ctx)
```

### 3. Error Handling Strategy
```go
// Rich error types with context
type ProcessorError struct {
    Type      ErrorType
    Component string
    Operation string
    Message   string
    Cause     error
    Context   map[string]interface{}
    Timestamp time.Time
}

// Error aggregation for concurrent operations
type ErrorCollector struct {
    mu     sync.Mutex
    errors []error
}

func (ec *ErrorCollector) Collect(err error) {
    ec.mu.Lock()
    defer ec.mu.Unlock()
    ec.errors = append(ec.errors, err)
}
```

### 4. Concurrency Model
```go
// Fan-out/Fan-in pattern with worker pools
type WorkerPool struct {
    size       int
    jobQueue   chan Job
    resultChan chan Result
    errorChan  chan error
    wg         sync.WaitGroup
}

func (wp *WorkerPool) Process(ctx context.Context) {
    // Start workers
    for i := 0; i < wp.size; i++ {
        wp.wg.Add(1)
        go wp.worker(ctx, i)
    }
    
    // Collect results
    go wp.collector(ctx)
    
    // Wait for completion
    wp.wg.Wait()
}
```

### 5. Observability
```go
// Structured logging with context
type LogEntry struct {
    Timestamp   time.Time
    Level       LogLevel
    Component   string
    Operation   string
    Message     string
    Fields      map[string]interface{}
    TraceID     string
    SpanID      string
}

// Metrics collection
type Metrics struct {
    recordsProcessed   prometheus.Counter
    processingDuration prometheus.Histogram
    errorRate          prometheus.Counter
    activeWorkers      prometheus.Gauge
}

// Health checks
type HealthChecker struct {
    checks []HealthCheck
}

type HealthCheck interface {
    Name() string
    Check(ctx context.Context) error
}
```

## Data Flow Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   PAF File  │────▶│   Reader    │────▶│  Validator  │
└─────────────┘     └─────────────┘     └─────────────┘
                            │                    │
                            ▼                    ▼
                    ┌─────────────┐     ┌─────────────┐
                    │  Channel    │     │  Channel    │
                    │  []PafHit   │     │  []PafHit   │
                    └─────────────┘     └─────────────┘
                            │                    │
┌─────────────┐            ▼                    ▼
│Mapping File │────▶┌─────────────┐     ┌─────────────┐
└─────────────┘     │ Calculator  │────▶│  Assigner   │
                    └─────────────┘     └─────────────┘
                            │                    │
                            ▼                    ▼
                    ┌─────────────┐     ┌─────────────┐
                    │  Channel    │     │  Channel    │
                    │ []PafScore  │     │ Assignment  │
                    └─────────────┘     └─────────────┘
                                                │
                                                ▼
                                        ┌─────────────┐
                                        │   Writer    │
                                        └─────────────┘
                                                │
                                                ▼
                                        ┌─────────────┐
                                        │Output Files │
                                        └─────────────┘
```

## Security Considerations

### 1. Input Validation
- Validate all file paths and prevent directory traversal
- Sanitize all input data before processing
- Set resource limits (file size, memory usage)

### 2. Access Control
- Run with minimal privileges
- Implement file permission checks
- Use secure temporary file handling

### 3. Data Protection
- No sensitive data in logs
- Secure configuration storage
- Audit trail for operations

## Performance Optimizations

### 1. Memory Management
```go
// Object pooling for frequently allocated structures
var pafHitPool = sync.Pool{
    New: func() interface{} {
        return &PafHit{}
    },
}

// Buffer pooling for I/O operations
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)
    },
}
```

### 2. Efficient I/O
```go
// Buffered reading with configurable buffer size
reader := bufio.NewReaderSize(file, config.BufferSize)

// Parallel file writing
type ParallelWriter struct {
    writers []io.Writer
    ch      chan WriteRequest
}
```

### 3. Caching Strategy
```go
// LRU cache for mapping lookups
type MappingCache struct {
    cache *lru.Cache
    mu    sync.RWMutex
}

func (mc *MappingCache) Get(key string) (MappingEntry, bool) {
    mc.mu.RLock()
    defer mc.mu.RUnlock()
    return mc.cache.Get(key)
}
```

## Deployment Architecture

### 1. Container Support
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o myPaf2Serotypes ./cmd/myPaf2Serotypes

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/myPaf2Serotypes /usr/local/bin/
ENTRYPOINT ["myPaf2Serotypes"]
```

### 2. Kubernetes Ready
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: mypaf2serotypes-config
data:
  config.yaml: |
    minScore: 0.9
    numWorkers: 4
    logLevel: info
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mypaf2serotypes
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: processor
        image: mypaf2serotypes:latest
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
```

## Migration Path

### Phase 1: Parallel Development
- Keep existing code functional
- Build new architecture alongside
- Maintain compatibility layer

### Phase 2: Gradual Migration
- Move components one at a time
- Use feature flags for switching
- Maintain comprehensive tests

### Phase 3: Cutover
- Performance testing comparison
- Staged rollout with monitoring
- Rollback plan ready

## Success Metrics

1. **Performance**
   - 2x throughput improvement
   - 50% memory reduction
   - Sub-second startup time

2. **Reliability**
   - 99.9% uptime
   - Graceful error handling
   - Zero data loss

3. **Maintainability**
   - 80%+ test coverage
   - <10 minute build time
   - Clear documentation

4. **Scalability**
   - Horizontal scaling ready
   - Cloud-native deployment
   - Resource efficiency

## Conclusion

This architecture design transforms myPafProcessor into a production-ready system that is:
- **Modular**: Easy to understand and modify
- **Scalable**: Ready for increased workloads
- **Observable**: Full visibility into operations
- **Reliable**: Robust error handling and recovery
- **Maintainable**: Clean code with comprehensive tests

The incremental migration approach ensures business continuity while improving the system.