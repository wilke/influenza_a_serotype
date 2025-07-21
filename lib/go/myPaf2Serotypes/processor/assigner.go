package processor

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/errors"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/models"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/utils"
)

// DefaultSerotypeAssigner implements SerotypeAssigner with concurrent processing
type DefaultSerotypeAssigner struct {
	config     ProcessorConfig
	numWorkers int
	logger     utils.Logger
	metrics    struct {
		scoresProcessed   int64
		assignmentsMade   int64
		ambiguousCount    int64
		unassignedCount   int64
	}
}

// NewDefaultSerotypeAssigner creates a new serotype assigner
func NewDefaultSerotypeAssigner(config ProcessorConfig, logger utils.Logger) (*DefaultSerotypeAssigner, error) {
	if config.MinScore < 0 || config.MinScore > 1 {
		return nil, errors.ValidationError("DefaultSerotypeAssigner", "NewDefaultSerotypeAssigner", 
			"minScore must be between 0 and 1")
	}
	if config.ScoreDistance < 0 || config.ScoreDistance > 1 {
		return nil, errors.ValidationError("DefaultSerotypeAssigner", "NewDefaultSerotypeAssigner",
			"scoreDistance must be between 0 and 1")
	}
	if config.NumWorkers <= 0 {
		config.NumWorkers = 4
	}
	if logger == nil {
		return nil, errors.ValidationError("DefaultSerotypeAssigner", "NewDefaultSerotypeAssigner",
			"logger cannot be nil")
	}

	return &DefaultSerotypeAssigner{
		config:     config,
		numWorkers: config.NumWorkers,
		logger:     logger,
	}, nil
}

// Assign implements SerotypeAssigner interface
func (a *DefaultSerotypeAssigner) Assign(ctx context.Context, scores <-chan []models.PafScore) <-chan models.Assignment {
	assignments := make(chan models.Assignment, 100)

	go func() {
		defer close(assignments)

		a.logger.Info("Starting serotype assignment",
			utils.Float64("min_score", a.config.MinScore),
			utils.Float64("score_distance", a.config.ScoreDistance),
			utils.Int("num_workers", a.numWorkers),
		)

		// Group scores by query name
		queryGroups := make(map[string][]models.PafScore)
		
		for scoreSet := range scores {
			select {
			case <-ctx.Done():
				return
			default:
			}

			for _, score := range scoreSet {
				queryGroups[score.QueryName] = append(queryGroups[score.QueryName], score)
				atomic.AddInt64(&a.metrics.scoresProcessed, 1)
			}
		}

		// Process each query group
		var wg sync.WaitGroup
		workChan := make(chan queryWork, len(queryGroups))

		// Start workers
		for i := 0; i < a.numWorkers; i++ {
			wg.Add(1)
			go a.worker(ctx, i, workChan, assignments, &wg)
		}

		// Send work to workers
		for queryName, queryScores := range queryGroups {
			workChan <- queryWork{
				queryName: queryName,
				scores:    queryScores,
			}
		}
		close(workChan)

		// Wait for workers to complete
		wg.Wait()

		a.logger.Info("Completed serotype assignment",
			utils.Int64("scores_processed", atomic.LoadInt64(&a.metrics.scoresProcessed)),
			utils.Int64("assignments_made", atomic.LoadInt64(&a.metrics.assignmentsMade)),
			utils.Int64("ambiguous", atomic.LoadInt64(&a.metrics.ambiguousCount)),
			utils.Int64("unassigned", atomic.LoadInt64(&a.metrics.unassignedCount)),
		)
	}()

	return assignments
}

// queryWork represents work for a single query
type queryWork struct {
	queryName string
	scores    []models.PafScore
}

// worker processes query work and makes assignments
func (a *DefaultSerotypeAssigner) worker(ctx context.Context, id int, work <-chan queryWork, assignments chan<- models.Assignment, wg *sync.WaitGroup) {
	defer wg.Done()

	a.logger.Debug("Assignment worker started", utils.Int("worker_id", id))

	for w := range work {
		select {
		case <-ctx.Done():
			return
		default:
		}

		assignment := a.assignSerotype(w.queryName, w.scores)
		
		select {
		case <-ctx.Done():
			return
		case assignments <- *assignment:
			atomic.AddInt64(&a.metrics.assignmentsMade, 1)
			
			// Update metrics
			if assignment.IsAmbiguous() {
				atomic.AddInt64(&a.metrics.ambiguousCount, 1)
			} else if !assignment.IsAssigned() {
				atomic.AddInt64(&a.metrics.unassignedCount, 1)
			}
		}
	}

	a.logger.Debug("Assignment worker completed", utils.Int("worker_id", id))
}

// assignSerotype determines the serotype assignment for a query
func (a *DefaultSerotypeAssigner) assignSerotype(queryName string, scores []models.PafScore) *models.Assignment {
	// Create summaries from scores
	summaries := a.createSummaries(scores)
	
	// Create assignment
	assignment := models.NewAssignment(queryName, summaries)
	
	// Assign serotype based on configuration
	assignment.AssignSerotype(a.config.MinScore, a.config.ScoreDistance)

	a.logger.Debug("Assigned serotype",
		utils.String("query", queryName),
		utils.String("serotype", string(assignment.Serotype)),
		utils.Float64("score", assignment.AlignmentScore),
		utils.String("reason", assignment.Reason),
	)

	return assignment
}

// createSummaries groups scores by serotype and creates summaries
func (a *DefaultSerotypeAssigner) createSummaries(scores []models.PafScore) []models.Summary {
	// Group scores by serotype
	serotypeGroups := make(map[models.Serotype][]models.PafScore)
	
	for _, score := range scores {
		serotypeGroups[score.Serotype] = append(serotypeGroups[score.Serotype], score)
	}

	// Create summary for each serotype
	summaries := make([]models.Summary, 0, len(serotypeGroups))
	
	for serotype, serotypeScores := range serotypeGroups {
		summary := models.NewSummary(scores[0].QueryName, serotypeScores)
		summary.Serotype = serotype
		summaries = append(summaries, *summary)
	}

	return summaries
}

// GetMetrics returns assigner metrics
func (a *DefaultSerotypeAssigner) GetMetrics() (scoresProcessed, assignmentsMade, ambiguous, unassigned int64) {
	return atomic.LoadInt64(&a.metrics.scoresProcessed),
		   atomic.LoadInt64(&a.metrics.assignmentsMade),
		   atomic.LoadInt64(&a.metrics.ambiguousCount),
		   atomic.LoadInt64(&a.metrics.unassignedCount)
}

// FilteringAssigner wraps an assigner with additional filtering logic
type FilteringAssigner struct {
	assigner SerotypeAssigner
	filter   AssignmentFilter
	logger   utils.Logger
}

// AssignmentFilter defines a filter for assignments
type AssignmentFilter func(models.Assignment) bool

// NewFilteringAssigner creates a new filtering assigner
func NewFilteringAssigner(assigner SerotypeAssigner, filter AssignmentFilter, logger utils.Logger) *FilteringAssigner {
	return &FilteringAssigner{
		assigner: assigner,
		filter:   filter,
		logger:   logger,
	}
}

// Assign processes scores and filters assignments
func (f *FilteringAssigner) Assign(ctx context.Context, scores <-chan []models.PafScore) <-chan models.Assignment {
	assignments := make(chan models.Assignment, 100)

	go func() {
		defer close(assignments)

		// Get assignments from wrapped assigner
		rawAssignments := f.assigner.Assign(ctx, scores)
		
		filtered := 0
		total := 0
		
		for assignment := range rawAssignments {
			total++
			
			// Apply filter
			if f.filter(assignment) {
				select {
				case <-ctx.Done():
					return
				case assignments <- assignment:
				}
			} else {
				filtered++
				f.logger.Debug("Filtered assignment",
					utils.String("query", assignment.QueryName),
					utils.String("serotype", string(assignment.Serotype)),
				)
			}
		}

		f.logger.Info("Filtering complete",
			utils.Int("total", total),
			utils.Int("filtered", filtered),
			utils.Int("passed", total-filtered),
		)
	}()

	return assignments
}

// Common filters

// MinScoreFilter creates a filter that removes assignments below a score threshold
func MinScoreFilter(minScore float64) AssignmentFilter {
	return func(a models.Assignment) bool {
		return a.AlignmentScore >= minScore
	}
}

// SerotypeFilter creates a filter that only allows specific serotypes
func SerotypeFilter(allowedSerotypes []models.Serotype) AssignmentFilter {
	allowed := make(map[models.Serotype]bool)
	for _, s := range allowedSerotypes {
		allowed[s] = true
	}
	
	return func(a models.Assignment) bool {
		return allowed[a.Serotype]
	}
}

// AssignedOnlyFilter creates a filter that only allows assigned reads
func AssignedOnlyFilter() AssignmentFilter {
	return func(a models.Assignment) bool {
		return a.IsAssigned()
	}
}