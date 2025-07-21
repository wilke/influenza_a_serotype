package processor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/errors"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/models"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/utils"
)

// DefaultScoreCalculator implements ScoreCalculator with concurrent processing
type DefaultScoreCalculator struct {
	mapping    models.Mapping
	numWorkers int
	logger     utils.Logger
	metrics    struct {
		recordsProcessed int64
		scoresCalculated int64
	}
}

// NewDefaultScoreCalculator creates a new score calculator
func NewDefaultScoreCalculator(mapping models.Mapping, numWorkers int, logger utils.Logger) (*DefaultScoreCalculator, error) {
	if mapping == nil {
		return nil, errors.ValidationError("DefaultScoreCalculator", "NewDefaultScoreCalculator", "mapping cannot be nil")
	}
	if numWorkers <= 0 {
		numWorkers = 4 // Default workers
	}
	if logger == nil {
		return nil, errors.ValidationError("DefaultScoreCalculator", "NewDefaultScoreCalculator", "logger cannot be nil")
	}

	return &DefaultScoreCalculator{
		mapping:    mapping,
		numWorkers: numWorkers,
		logger:     logger,
	}, nil
}

// Calculate implements ScoreCalculator interface
func (c *DefaultScoreCalculator) Calculate(ctx context.Context, records <-chan []models.PafHit) <-chan []models.PafScore {
	scores := make(chan []models.PafScore, 100)

	go func() {
		defer close(scores)

		c.logger.Info("Starting score calculation",
			utils.Int("num_workers", c.numWorkers),
		)

		// Create worker pool
		var wg sync.WaitGroup
		workerRecords := make(chan []models.PafHit, c.numWorkers*2)

		// Start workers
		for i := 0; i < c.numWorkers; i++ {
			wg.Add(1)
			go c.worker(ctx, i, workerRecords, scores, &wg)
		}

		// Distribute work to workers
		go func() {
			defer close(workerRecords)
			for record := range records {
				select {
				case <-ctx.Done():
					return
				case workerRecords <- record:
					atomic.AddInt64(&c.metrics.recordsProcessed, 1)
				}
			}
		}()

		// Wait for all workers to complete
		wg.Wait()

		c.logger.Info("Completed score calculation",
			utils.Int64("records_processed", atomic.LoadInt64(&c.metrics.recordsProcessed)),
			utils.Int64("scores_calculated", atomic.LoadInt64(&c.metrics.scoresCalculated)),
		)
	}()

	return scores
}

// worker processes records and calculates scores
func (c *DefaultScoreCalculator) worker(ctx context.Context, id int, records <-chan []models.PafHit, scores chan<- []models.PafScore, wg *sync.WaitGroup) {
	defer wg.Done()

	c.logger.Debug("Worker started",
		utils.Int("worker_id", id),
	)

	for hits := range records {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if len(hits) == 0 {
			continue
		}

		// Calculate scores for this group of hits
		calculated := c.calculateScoresForQuery(hits)
		
		if len(calculated) > 0 {
			select {
			case <-ctx.Done():
				return
			case scores <- calculated:
				atomic.AddInt64(&c.metrics.scoresCalculated, int64(len(calculated)))
			}
		}
	}

	c.logger.Debug("Worker completed",
		utils.Int("worker_id", id),
	)
}

// calculateScoresForQuery calculates scores for all hits from a single query
func (c *DefaultScoreCalculator) calculateScoresForQuery(hits []models.PafHit) []models.PafScore {
	if len(hits) == 0 {
		return nil
	}

	// Group hits by target name, serotype, segment, and strand
	groups := c.groupHits(hits)
	
	// Calculate score for each group
	scores := make([]models.PafScore, 0, len(groups))
	
	for groupKey, groupHits := range groups {
		score := c.calculateGroupScore(groupKey, groupHits)
		scores = append(scores, *score)
	}

	return scores
}

// groupKey represents a unique grouping of hits
type groupKey struct {
	QueryName string
	TargetName string
	Serotype  models.Serotype
	Segment   models.Segment
	Strand    string
}

// groupHits groups hits by their key attributes
func (c *DefaultScoreCalculator) groupHits(hits []models.PafHit) map[groupKey][]*models.PafHit {
	groups := make(map[groupKey][]*models.PafHit)

	for i := range hits {
		hit := &hits[i]
		
		// Get segment from mapping if available
		segment := models.Segment("")
		if entry, ok := c.mapping.GetEntry(models.Accession(hit.TargetName)); ok {
			segment = entry.Segment
		}

		key := groupKey{
			QueryName:  hit.QueryName,
			TargetName: hit.TargetName,
			Serotype:   hit.Serotype,
			Segment:    segment,
			Strand:     hit.Strand,
		}

		groups[key] = append(groups[key], hit)
	}

	return groups
}

// calculateGroupScore calculates the aggregated score for a group of hits
func (c *DefaultScoreCalculator) calculateGroupScore(key groupKey, hits []*models.PafHit) *models.PafScore {
	if len(hits) == 0 {
		return nil
	}

	score := &models.PafScore{
		QueryName:             key.QueryName,
		Serotype:              key.Serotype,
		Segment:               key.Segment,
		Group:                 fmt.Sprintf("%s:%s:%s:%s", key.QueryName, key.TargetName, key.Serotype, key.Strand),
		Hits:                  hits,
		NumberOfContributions: len(hits),
	}

	// Use the first hit to set read length (should be same for all hits from same query)
	if len(hits) > 0 {
		score.TotalReadLength = hits[0].QueryLength
	}

	// Calculate aggregated values
	for _, hit := range hits {
		score.TotalAlignmentLength += hit.GetAlignmentLength()
		score.TotalMatches += hit.NumResidueMatches
	}

	// Calculate ANI (Average Nucleotide Identity)
	if score.TotalAlignmentLength > 0 {
		score.ANI = float64(score.TotalMatches) / float64(score.TotalAlignmentLength)
	}

	// Calculate AFI (Alignment Fraction Index)
	if score.TotalReadLength > 0 {
		score.AFI = float64(score.TotalAlignmentLength) / float64(score.TotalReadLength)
	}

	// Calculate combined alignment score
	score.AlignmentScore = score.ANI * score.AFI

	c.logger.Debug("Calculated group score",
		utils.String("query", key.QueryName),
		utils.String("serotype", string(key.Serotype)),
		utils.Float64("score", score.AlignmentScore),
		utils.Int("hits", len(hits)),
	)

	return score
}

// GetMetrics returns calculator metrics
func (c *DefaultScoreCalculator) GetMetrics() (recordsProcessed, scoresCalculated int64) {
	return atomic.LoadInt64(&c.metrics.recordsProcessed), 
		   atomic.LoadInt64(&c.metrics.scoresCalculated)
}

// BatchScoreCalculator processes scores in batches for better performance
type BatchScoreCalculator struct {
	calculator ScoreCalculator
	batchSize  int
	logger     utils.Logger
}

// NewBatchScoreCalculator creates a new batch score calculator
func NewBatchScoreCalculator(calculator ScoreCalculator, batchSize int, logger utils.Logger) *BatchScoreCalculator {
	if batchSize <= 0 {
		batchSize = 100
	}

	return &BatchScoreCalculator{
		calculator: calculator,
		batchSize:  batchSize,
		logger:     logger,
	}
}

// Calculate processes records in batches
func (b *BatchScoreCalculator) Calculate(ctx context.Context, records <-chan []models.PafHit) <-chan []models.PafScore {
	scores := make(chan []models.PafScore, 100)

	go func() {
		defer close(scores)

		// Create batched channel
		batched := make(chan []models.PafHit, b.batchSize)
		
		// Batch collector
		go func() {
			defer close(batched)
			batch := make([][]models.PafHit, 0, b.batchSize)

			for record := range records {
				batch = append(batch, record)
				
				if len(batch) >= b.batchSize {
					// Send batch
					for _, r := range batch {
						select {
						case <-ctx.Done():
							return
						case batched <- r:
						}
					}
					batch = batch[:0]
				}
			}

			// Send remaining records
			for _, r := range batch {
				select {
				case <-ctx.Done():
					return
				case batched <- r:
				}
			}
		}()

		// Process batched records
		calculated := b.calculator.Calculate(ctx, batched)
		for scoreSet := range calculated {
			select {
			case <-ctx.Done():
				return
			case scores <- scoreSet:
			}
		}
	}()

	return scores
}