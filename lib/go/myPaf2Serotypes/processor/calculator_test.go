package processor

import (
	"context"
	"testing"
	"time"

	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/models"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestCalculatorMapping() models.Mapping {
	return models.Mapping{
		"ref1": models.MappingEntry{
			Serotype: "H1N1",
			Segment:  "1",
		},
		"ref2": models.MappingEntry{
			Serotype: "H3N2",
			Segment:  "2",
		},
		"ref3": models.MappingEntry{
			Serotype: "H1N1",
			Segment:  "4",
		},
	}
}

func TestNewDefaultScoreCalculator(t *testing.T) {
	mapping := createTestCalculatorMapping()
	logger := utils.NewLogger("info", "text", false)

	tests := []struct {
		name       string
		mapping    models.Mapping
		numWorkers int
		logger     utils.Logger
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "Valid parameters",
			mapping:    mapping,
			numWorkers: 4,
			logger:     logger,
			wantErr:    false,
		},
		{
			name:       "Nil mapping",
			mapping:    nil,
			numWorkers: 4,
			logger:     logger,
			wantErr:    true,
			errMsg:     "mapping cannot be nil",
		},
		{
			name:       "Nil logger",
			mapping:    mapping,
			numWorkers: 4,
			logger:     nil,
			wantErr:    true,
			errMsg:     "logger cannot be nil",
		},
		{
			name:       "Zero workers (should use default)",
			mapping:    mapping,
			numWorkers: 0,
			logger:     logger,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calc, err := NewDefaultScoreCalculator(tt.mapping, tt.numWorkers, tt.logger)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, calc)
				if tt.numWorkers == 0 {
					assert.Equal(t, 4, calc.numWorkers) // Default
				}
			}
		})
	}
}

func TestDefaultScoreCalculator_Calculate(t *testing.T) {
	mapping := createTestCalculatorMapping()
	logger := utils.NewLogger("error", "text", false)
	calc, err := NewDefaultScoreCalculator(mapping, 2, logger)
	require.NoError(t, err)

	tests := []struct {
		name           string
		hits           [][]models.PafHit
		expectedScores int
		validate       func(t *testing.T, scores []models.PafScore)
	}{
		{
			name: "Single query single hit",
			hits: [][]models.PafHit{
				{
					{
						QueryName:            "read1",
						QueryLength:          100,
						QueryStart:           10,
						QueryEnd:             90,
						Strand:               "+",
						TargetName:           "ref1",
						TargetLength:         1000,
						TargetStart:          100,
						TargetEnd:            180,
						NumResidueMatches:    75,
						AlignmentBlockLength: 80,
						MappingQuality:       60,
						Serotype:             "H1N1",
					},
				},
			},
			expectedScores: 1,
			validate: func(t *testing.T, scores []models.PafScore) {
				assert.Len(t, scores, 1)
				score := scores[0]
				assert.Equal(t, "read1", score.QueryName)
				assert.Equal(t, models.Serotype("H1N1"), score.Serotype)
				assert.Equal(t, 80, score.TotalAlignmentLength)
				assert.Equal(t, 75, score.TotalMatches)
				assert.Equal(t, 75.0/80.0, score.ANI)
				assert.Equal(t, 80.0/100.0, score.AFI)
				assert.Equal(t, (75.0/80.0)*(80.0/100.0), score.AlignmentScore)
			},
		},
		{
			name: "Single query multiple hits same target",
			hits: [][]models.PafHit{
				{
					{
						QueryName:            "read1",
						QueryLength:          100,
						QueryStart:           10,
						QueryEnd:             50,
						Strand:               "+",
						TargetName:           "ref1",
						TargetLength:         1000,
						TargetStart:          100,
						TargetEnd:            140,
						NumResidueMatches:    35,
						AlignmentBlockLength: 40,
						MappingQuality:       60,
						Serotype:             "H1N1",
					},
					{
						QueryName:            "read1",
						QueryLength:          100,
						QueryStart:           50,
						QueryEnd:             90,
						Strand:               "+",
						TargetName:           "ref1",
						TargetLength:         1000,
						TargetStart:          200,
						TargetEnd:            240,
						NumResidueMatches:    38,
						AlignmentBlockLength: 40,
						MappingQuality:       60,
						Serotype:             "H1N1",
					},
				},
			},
			expectedScores: 1,
			validate: func(t *testing.T, scores []models.PafScore) {
				assert.Len(t, scores, 1)
				score := scores[0]
				assert.Equal(t, 80, score.TotalAlignmentLength) // 40 + 40
				assert.Equal(t, 73, score.TotalMatches)          // 35 + 38
				assert.Equal(t, 2, score.NumberOfContributions)
			},
		},
		{
			name: "Single query hits to different serotypes",
			hits: [][]models.PafHit{
				{
					{
						QueryName:            "read1",
						QueryLength:          100,
						QueryStart:           10,
						QueryEnd:             90,
						Strand:               "+",
						TargetName:           "ref1",
						TargetLength:         1000,
						TargetStart:          100,
						TargetEnd:            180,
						NumResidueMatches:    75,
						AlignmentBlockLength: 80,
						MappingQuality:       60,
						Serotype:             "H1N1",
					},
					{
						QueryName:            "read1",
						QueryLength:          100,
						QueryStart:           10,
						QueryEnd:             90,
						Strand:               "+",
						TargetName:           "ref2",
						TargetLength:         2000,
						TargetStart:          200,
						TargetEnd:            280,
						NumResidueMatches:    70,
						AlignmentBlockLength: 80,
						MappingQuality:       55,
						Serotype:             "H3N2",
					},
				},
			},
			expectedScores: 2,
			validate: func(t *testing.T, scores []models.PafScore) {
				assert.Len(t, scores, 2)
				// Find H1N1 and H3N2 scores
				var h1n1Score, h3n2Score *models.PafScore
				for i := range scores {
					if scores[i].Serotype == "H1N1" {
						h1n1Score = &scores[i]
					} else if scores[i].Serotype == "H3N2" {
						h3n2Score = &scores[i]
					}
				}
				assert.NotNil(t, h1n1Score)
				assert.NotNil(t, h3n2Score)
			},
		},
		{
			name: "Different strands grouped separately",
			hits: [][]models.PafHit{
				{
					{
						QueryName:            "read1",
						QueryLength:          100,
						QueryStart:           10,
						QueryEnd:             90,
						Strand:               "+",
						TargetName:           "ref1",
						TargetLength:         1000,
						TargetStart:          100,
						TargetEnd:            180,
						NumResidueMatches:    75,
						AlignmentBlockLength: 80,
						MappingQuality:       60,
						Serotype:             "H1N1",
					},
					{
						QueryName:            "read1",
						QueryLength:          100,
						QueryStart:           10,
						QueryEnd:             90,
						Strand:               "-",
						TargetName:           "ref1",
						TargetLength:         1000,
						TargetStart:          100,
						TargetEnd:            180,
						NumResidueMatches:    75,
						AlignmentBlockLength: 80,
						MappingQuality:       60,
						Serotype:             "H1N1",
					},
				},
			},
			expectedScores: 2, // Different strands = different groups
		},
		{
			name:           "Empty hits",
			hits:           [][]models.PafHit{},
			expectedScores: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			records := make(chan []models.PafHit, 10)

			// Send test data
			go func() {
				for _, hits := range tt.hits {
					records <- hits
				}
				close(records)
			}()

			// Calculate scores
			scoresChan := calc.Calculate(ctx, records)

			// Collect results
			var allScores []models.PafScore
			for scores := range scoresChan {
				allScores = append(allScores, scores...)
			}

			// Verify
			assert.Len(t, allScores, tt.expectedScores)
			if tt.validate != nil {
				tt.validate(t, allScores)
			}
		})
	}
}

func TestDefaultScoreCalculator_ContextCancellation(t *testing.T) {
	mapping := createTestCalculatorMapping()
	logger := utils.NewLogger("error", "text", false)
	calc, err := NewDefaultScoreCalculator(mapping, 2, logger)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	records := make(chan []models.PafHit, 10)

	// Start calculation
	scores := calc.Calculate(ctx, records)

	// Send some data
	records <- []models.PafHit{{
		QueryName: "read1",
		Serotype:  "H1N1",
	}}

	// Cancel context
	cancel()
	close(records)

	// Verify that scores channel is closed
	select {
	case _, ok := <-scores:
		assert.False(t, ok, "scores channel should be closed")
	case <-time.After(time.Second):
		t.Error("scores channel not closed after cancellation")
	}
}

func TestDefaultScoreCalculator_GetMetrics(t *testing.T) {
	mapping := createTestCalculatorMapping()
	logger := utils.NewLogger("error", "text", false)
	calc, err := NewDefaultScoreCalculator(mapping, 2, logger)
	require.NoError(t, err)

	// Initial metrics should be zero
	recordsProcessed, scoresCalculated := calc.GetMetrics()
	assert.Equal(t, int64(0), recordsProcessed)
	assert.Equal(t, int64(0), scoresCalculated)

	// Process some data
	ctx := context.Background()
	records := make(chan []models.PafHit, 10)

	scores := calc.Calculate(ctx, records)

	// Send test data
	records <- []models.PafHit{{
		QueryName:            "read1",
		QueryLength:          100,
		NumResidueMatches:    75,
		AlignmentBlockLength: 80,
		QueryEnd:             90,
		QueryStart:           10,
		TargetName:           "ref1",
		Serotype:             "H1N1",
	}}
	close(records)

	// Consume results
	for range scores {
	}

	// Check updated metrics
	recordsProcessed, scoresCalculated = calc.GetMetrics()
	assert.Equal(t, int64(1), recordsProcessed)
	assert.Equal(t, int64(1), scoresCalculated)
}

func BenchmarkDefaultScoreCalculator(b *testing.B) {
	mapping := createTestCalculatorMapping()
	logger := utils.NewLogger("error", "text", false)
	calc, err := NewDefaultScoreCalculator(mapping, 4, logger)
	require.NoError(b, err)

	// Create test data
	testHits := []models.PafHit{
		{
			QueryName:            "read1",
			QueryLength:          100,
			QueryStart:           10,
			QueryEnd:             90,
			Strand:               "+",
			TargetName:           "ref1",
			TargetLength:         1000,
			TargetStart:          100,
			TargetEnd:            180,
			NumResidueMatches:    75,
			AlignmentBlockLength: 80,
			MappingQuality:       60,
			Serotype:             "H1N1",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		records := make(chan []models.PafHit, 100)

		scores := calc.Calculate(ctx, records)

		// Send data
		go func() {
			for j := 0; j < 100; j++ {
				records <- testHits
			}
			close(records)
		}()

		// Consume results
		count := 0
		for range scores {
			count++
		}
	}
}