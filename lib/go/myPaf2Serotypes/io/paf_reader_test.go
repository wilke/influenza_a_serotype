package io

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/models"
	"github.com/me/influenza_a_serotype/lib/go/myPaf2Serotypes/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestMapping() models.Mapping {
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
			Serotype: "H5N1",
			Segment:  "4",
		},
	}
}

func createTestPAFFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	pafFile := filepath.Join(tmpDir, "test.paf")
	err := os.WriteFile(pafFile, []byte(content), 0644)
	require.NoError(t, err)
	return pafFile
}

func TestNewPAFFileReader(t *testing.T) {
	mapping := createTestMapping()
	logger := utils.NewLogger("info", "text", false)

	tests := []struct {
		name     string
		filename string
		mapping  models.Mapping
		logger   utils.Logger
		bufSize  int
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "Valid parameters",
			filename: "test.paf",
			mapping:  mapping,
			logger:   logger,
			bufSize:  4096,
			wantErr:  false,
		},
		{
			name:     "Empty filename",
			filename: "",
			mapping:  mapping,
			logger:   logger,
			bufSize:  4096,
			wantErr:  true,
			errMsg:   "filename cannot be empty",
		},
		{
			name:     "Nil mapping",
			filename: "test.paf",
			mapping:  nil,
			logger:   logger,
			bufSize:  4096,
			wantErr:  true,
			errMsg:   "mapping cannot be nil",
		},
		{
			name:     "Nil logger",
			filename: "test.paf",
			mapping:  mapping,
			logger:   nil,
			bufSize:  4096,
			wantErr:  true,
			errMsg:   "logger cannot be nil",
		},
		{
			name:     "Zero buffer size (should use default)",
			filename: "test.paf",
			mapping:  mapping,
			logger:   logger,
			bufSize:  0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader, err := NewPAFFileReader(tt.filename, tt.mapping, tt.logger, tt.bufSize)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, reader)
			}
		})
	}
}

func TestPAFFileReader_Read(t *testing.T) {
	tests := []struct {
		name           string
		pafContent     string
		expectedGroups int
		expectedHits   []int // hits per group
	}{
		{
			name: "Single query with multiple hits",
			pafContent: `read1	100	10	90	+	ref1	1000	100	180	75	80	60
read1	100	15	85	+	ref2	2000	200	270	65	70	55`,
			expectedGroups: 1,
			expectedHits:   []int{2},
		},
		{
			name: "Multiple queries",
			pafContent: `read1	100	10	90	+	ref1	1000	100	180	75	80	60
read2	150	20	120	-	ref2	2000	200	300	90	100	50
read3	200	0	180	+	ref3	3000	300	480	170	180	70`,
			expectedGroups: 3,
			expectedHits:   []int{1, 1, 1},
		},
		{
			name: "With invalid lines",
			pafContent: `read1	100	10	90	+	ref1	1000	100	180	75	80	60
invalid line
read2	150	20	120	-	ref2	2000	200	300	90	100	50

read3	200	0	180	+	ref3	3000	300	480	170	180	70`,
			expectedGroups: 3,
			expectedHits:   []int{1, 1, 1},
		},
		{
			name: "Unknown references (should be skipped)",
			pafContent: `read1	100	10	90	+	ref1	1000	100	180	75	80	60
read2	150	20	120	-	unknown_ref	2000	200	300	90	100	50
read3	200	0	180	+	ref3	3000	300	480	170	180	70`,
			expectedGroups: 2,
			expectedHits:   []int{1, 1},
		},
		{
			name:           "Empty file",
			pafContent:     "",
			expectedGroups: 0,
			expectedHits:   []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test PAF file
			pafFile := createTestPAFFile(t, tt.pafContent)
			
			// Create reader
			mapping := createTestMapping()
			logger := utils.NewLogger("error", "text", false)
			reader, err := NewPAFFileReader(pafFile, mapping, logger, 4096)
			require.NoError(t, err)
			defer reader.Close()

			// Read records
			ctx := context.Background()
			records, errChan := reader.Read(ctx)

			// Collect results
			var groups [][]models.PafHit
			for record := range records {
				groups = append(groups, record)
			}

			// Check for errors
			select {
			case err := <-errChan:
				assert.NoError(t, err)
			default:
			}

			// Verify results
			assert.Len(t, groups, tt.expectedGroups)
			for i, expectedHits := range tt.expectedHits {
				if i < len(groups) {
					assert.Len(t, groups[i], expectedHits)
				}
			}
		})
	}
}

func TestPAFFileReader_ContextCancellation(t *testing.T) {
	// Create a large PAF file
	var content strings.Builder
	for i := 0; i < 1000; i++ {
		content.WriteString("read")
		content.WriteString(string(rune(i)))
		content.WriteString("\t100\t10\t90\t+\tref1\t1000\t100\t180\t75\t80\t60\n")
	}
	pafFile := createTestPAFFile(t, content.String())

	// Create reader
	mapping := createTestMapping()
	logger := utils.NewLogger("error", "text", false)
	reader, err := NewPAFFileReader(pafFile, mapping, logger, 4096)
	require.NoError(t, err)
	defer reader.Close()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	records, errChan := reader.Read(ctx)

	// Read a few records
	count := 0
	for range records {
		count++
		if count >= 5 {
			cancel() // Cancel after 5 records
			break
		}
	}

	// Verify cancellation error
	select {
	case err := <-errChan:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "canceled")
	case <-time.After(time.Second):
		t.Error("Expected cancellation error")
	}
}

func TestPAFStreamReader(t *testing.T) {
	content := `read1	100	10	90	+	ref1	1000	100	180	75	80	60
read1	100	15	85	+	ref2	2000	200	270	65	70	55
read2	150	20	120	-	ref2	2000	200	300	90	100	50
`
	reader := bufio.NewReader(strings.NewReader(content))
	mapping := createTestMapping()
	logger := utils.NewLogger("error", "text", false)

	streamReader, err := NewPAFStreamReader(reader, mapping, logger)
	require.NoError(t, err)
	defer streamReader.Close()

	ctx := context.Background()

	// Read first batch (should be two hits for read1)
	batch1, err := streamReader.ReadBatch(ctx)
	assert.NoError(t, err)
	assert.Len(t, batch1, 2)
	assert.Equal(t, "read1", batch1[0].QueryName)
	assert.Equal(t, "read1", batch1[1].QueryName)

	// Read second batch (should be one hit for read2)
	batch2, err := streamReader.ReadBatch(ctx)
	assert.NoError(t, err)
	assert.Len(t, batch2, 1)
	assert.Equal(t, "read2", batch2[0].QueryName)

	// Read third batch (should be EOF)
	batch3, err := streamReader.ReadBatch(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "EOF")
	assert.Nil(t, batch3)
}

func TestPAFFileReader_NonExistentFile(t *testing.T) {
	mapping := createTestMapping()
	logger := utils.NewLogger("error", "text", false)
	reader, err := NewPAFFileReader("/non/existent/file.paf", mapping, logger, 4096)
	require.NoError(t, err)

	ctx := context.Background()
	records, errChan := reader.Read(ctx)

	// Try to read
	for range records {
		t.Error("Should not receive any records")
	}

	// Check for error
	select {
	case err := <-errChan:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open PAF file")
	default:
		t.Error("Expected error for non-existent file")
	}
}

func BenchmarkPAFFileReader(b *testing.B) {
	// Create test PAF content
	var content strings.Builder
	for i := 0; i < 10000; i++ {
		content.WriteString("read")
		content.WriteString(string(rune(i % 1000)))
		content.WriteString("\t100\t10\t90\t+\tref")
		content.WriteString(string(rune(i % 3 + 1)))
		content.WriteString("\t1000\t100\t180\t75\t80\t60\n")
	}

	tmpFile, err := os.CreateTemp("", "bench_*.paf")
	require.NoError(b, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(content.String())
	require.NoError(b, err)
	tmpFile.Close()

	mapping := createTestMapping()
	logger := utils.NewLogger("error", "text", false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader, err := NewPAFFileReader(tmpFile.Name(), mapping, logger, 4096)
		require.NoError(b, err)

		ctx := context.Background()
		records, _ := reader.Read(ctx)

		count := 0
		for range records {
			count++
		}

		reader.Close()
	}
}