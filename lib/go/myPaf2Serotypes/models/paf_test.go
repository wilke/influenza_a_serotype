package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePafLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    *PafHit
		wantErr bool
	}{
		{
			name: "Valid PAF line",
			line: "read1\t100\t10\t90\t+\tref1\t1000\t100\t180\t75\t80\t60",
			want: &PafHit{
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
			},
			wantErr: false,
		},
		{
			name: "Reverse strand",
			line: "read2\t150\t20\t120\t-\tref2\t2000\t200\t300\t90\t100\t50",
			want: &PafHit{
				QueryName:            "read2",
				QueryLength:          150,
				QueryStart:           20,
				QueryEnd:             120,
				Strand:               "-",
				TargetName:           "ref2",
				TargetLength:         2000,
				TargetStart:          200,
				TargetEnd:            300,
				NumResidueMatches:    90,
				AlignmentBlockLength: 100,
				MappingQuality:       50,
			},
			wantErr: false,
		},
		{
			name:    "Too few fields",
			line:    "read1\t100\t10\t90\t+\tref1",
			wantErr: true,
		},
		{
			name:    "Invalid number",
			line:    "read1\tabc\t10\t90\t+\tref1\t1000\t100\t180\t75\t80\t60",
			wantErr: true,
		},
		{
			name:    "Empty line",
			line:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePafLine(tt.line)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestPafHitCalculations(t *testing.T) {
	paf := &PafHit{
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
	}

	t.Run("CalculateANI", func(t *testing.T) {
		expected := 75.0 / 80.0
		assert.Equal(t, expected, paf.CalculateANI())
	})

	t.Run("CalculateAF", func(t *testing.T) {
		expected := 80.0 / 100.0
		assert.Equal(t, expected, paf.CalculateAF())
	})

	t.Run("CalculateScore", func(t *testing.T) {
		expectedANI := 75.0 / 80.0
		expectedAF := 80.0 / 100.0
		expected := expectedANI * expectedAF
		assert.Equal(t, expected, paf.CalculateScore())
	})

	t.Run("GetAlignmentLength", func(t *testing.T) {
		assert.Equal(t, 80, paf.GetAlignmentLength())
	})

	t.Run("GetTargetAlignmentLength", func(t *testing.T) {
		assert.Equal(t, 80, paf.GetTargetAlignmentLength())
	})

	t.Run("IsForwardStrand", func(t *testing.T) {
		assert.True(t, paf.IsForwardStrand())
		
		paf.Strand = "-"
		assert.False(t, paf.IsForwardStrand())
	})
}

func TestPafHitEdgeCases(t *testing.T) {
	t.Run("Zero alignment length", func(t *testing.T) {
		paf := &PafHit{
			AlignmentBlockLength: 0,
			QueryLength:          100,
		}
		assert.Equal(t, 0.0, paf.CalculateANI())
	})

	t.Run("Zero query length", func(t *testing.T) {
		paf := &PafHit{
			QueryLength: 0,
			QueryEnd:    50,
			QueryStart:  0,
		}
		assert.Equal(t, 0.0, paf.CalculateAF())
	})
}

func TestPafHitValidate(t *testing.T) {
	tests := []struct {
		name    string
		paf     *PafHit
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid PAF hit",
			paf: &PafHit{
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
			},
			wantErr: false,
		},
		{
			name: "Empty query name",
			paf: &PafHit{
				QueryName:    "",
				TargetName:   "ref1",
				QueryLength:  100,
				TargetLength: 1000,
				Strand:       "+",
			},
			wantErr: true,
			errMsg:  "query name is empty",
		},
		{
			name: "Invalid strand",
			paf: &PafHit{
				QueryName:    "read1",
				TargetName:   "ref1",
				QueryLength:  100,
				TargetLength: 1000,
				Strand:       "x",
			},
			wantErr: true,
			errMsg:  "invalid strand",
		},
		{
			name: "Invalid query coordinates",
			paf: &PafHit{
				QueryName:    "read1",
				TargetName:   "ref1",
				QueryLength:  100,
				TargetLength: 1000,
				QueryStart:   50,
				QueryEnd:     30, // End before start
				Strand:       "+",
			},
			wantErr: true,
			errMsg:  "invalid query coordinates",
		},
		{
			name: "Negative mapping quality",
			paf: &PafHit{
				QueryName:      "read1",
				TargetName:     "ref1",
				QueryLength:    100,
				TargetLength:   1000,
				QueryEnd:       50,
				TargetEnd:      500,
				Strand:         "+",
				MappingQuality: -1,
			},
			wantErr: true,
			errMsg:  "mapping quality out of range",
		},
		{
			name: "Too many matches",
			paf: &PafHit{
				QueryName:            "read1",
				TargetName:           "ref1",
				QueryLength:          100,
				TargetLength:         1000,
				QueryEnd:             50,
				TargetEnd:            500,
				Strand:               "+",
				NumResidueMatches:    100,
				AlignmentBlockLength: 80,
			},
			wantErr: true,
			errMsg:  "invalid number of matches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.paf.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPafHitString(t *testing.T) {
	paf := &PafHit{
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
	}

	expected := "read1\t100\t10\t90\t+\tref1\t1000\t100\t180\t75\t80\t60"
	assert.Equal(t, expected, paf.String())
}

func BenchmarkParsePafLine(b *testing.B) {
	line := "read1\t100\t10\t90\t+\tref1\t1000\t100\t180\t75\t80\t60"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParsePafLine(line)
	}
}

func BenchmarkPafCalculations(b *testing.B) {
	paf := &PafHit{
		QueryLength:          100,
		QueryStart:           10,
		QueryEnd:             90,
		NumResidueMatches:    75,
		AlignmentBlockLength: 80,
	}

	b.Run("ANI", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = paf.CalculateANI()
		}
	})

	b.Run("AF", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = paf.CalculateAF()
		}
	})

	b.Run("Score", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = paf.CalculateScore()
		}
	})
}