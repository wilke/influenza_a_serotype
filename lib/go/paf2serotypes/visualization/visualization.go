package visualization

import (
	"fmt"
	"image/color"
	"path/filepath"
	"sort"

	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/log"
	"github.com/me/influenza_a_serotype/lib/go/paf2serotypes/model"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"gonum.org/v1/plot/vg/draw"
)

// SerotypeCounts represents a serotype and its read count
type SerotypeCounts struct {
	Serotype string
	Count    int
}

// GenerateBarChart creates a bar chart of read assignments by serotype
// and saves it as a PDF file
func GenerateBarChart(summaries []model.SerotypeSummary, outDir, sampleName string) error {
	// Count reads by serotype
	counts := make(map[string]int)
	for _, summary := range summaries {
		if summary.ReadAssignment != "" {
			counts[summary.ReadAssignment]++
		}
	}

	// If no data, return early
	if len(counts) == 0 {
		log.Warn("No data available for visualization")
		return nil
	}

	// Create a new plot
	p := plot.New()

	p.Title.Text = fmt.Sprintf("Read Serotype Assignments: %s", sampleName)
	p.X.Label.Text = "Serotype"
	p.Y.Label.Text = "Count"

	// Rotate x-axis labels for better readability
	p.X.Tick.Label.Rotation = 1.5708 // 90 degrees in radians (π/2)
	p.X.Tick.Label.YAlign = draw.YCenter
	p.X.Tick.Label.XAlign = draw.XRight

	// Add some padding
	p.X.Padding = vg.Points(10)
	p.Y.Padding = vg.Points(10)

	// Convert map to sorted slice for consistent ordering
	var sortedCounts []SerotypeCounts
	for serotype, count := range counts {
		sortedCounts = append(sortedCounts, SerotypeCounts{
			Serotype: serotype,
			Count:    count,
		})
	}
	sort.Slice(sortedCounts, func(i, j int) bool {
		return sortedCounts[i].Serotype < sortedCounts[j].Serotype
	})

	// Create the bars
	w := vg.Points(20)

	// Create values for the bar chart
	values := make(plotter.Values, len(sortedCounts))
	for i, sc := range sortedCounts {
		values[i] = float64(sc.Count)
	}

	// Create the bar chart
	bars, err := plotter.NewBarChart(values, w)
	if err != nil {
		return fmt.Errorf("failed to create bar chart: %w", err)
	}

	// Set the positions
	bars.Offset = -0.5 * w

	// Customize the bars
	bars.Width = w
	bars.Color = color.RGBA{R: 31, G: 119, B: 180, A: 255}
	bars.LineStyle.Width = vg.Points(1)

	// Add the bars to the plot
	p.Add(bars)

	// Set the X axis labels
	p.NominalX(getSerotypes(sortedCounts)...)

	// Save the plot to a PDF file
	outPath := filepath.Join(outDir, fmt.Sprintf("%s_read_serotype_assignment.pdf", sampleName))
	if err := p.Save(6*vg.Inch, 4*vg.Inch, outPath); err != nil {
		return fmt.Errorf("failed to save visualization: %w", err)
	}

	log.Info("Visualization saved to %s", outPath)
	return nil
}

// getSerotypes extracts serotype names from the sorted counts
func getSerotypes(counts []SerotypeCounts) []string {
	serotypes := make([]string, len(counts))
	for i, sc := range counts {
		serotypes[i] = sc.Serotype
	}
	return serotypes
}
