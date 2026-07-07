package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// BarChart renders a simple unicode bar chart
func BarChart(data map[string]int, width int) string {
	if len(data) == 0 {
		return lipgloss.NewStyle().Faint(true).Render("No category data available.")
	}

	displayData := make(map[string]int, len(data))
	for k, v := range data {
		name := k
		if strings.TrimSpace(name) == "" {
			name = "Other"
		}
		displayData[name] += v
	}

	keys := make([]string, 0, len(displayData))
	maxVal := 0
	for k, v := range displayData {
		keys = append(keys, k)
		if v > maxVal {
			maxVal = v
		}
	}

	sort.Slice(keys, func(i, j int) bool {
		return displayData[keys[i]] > displayData[keys[j]]
	})

	// Limit to top 5 categories for space
	if len(keys) > 5 {
		keys = keys[:5]
	}

	var sb strings.Builder

	// Calculate max label width
	maxLabelWidth := 0
	for _, k := range keys {
		if len(k) > maxLabelWidth {
			maxLabelWidth = len(k)
		}
	}

	// Safety for small screens
	availableWidth := width - maxLabelWidth - 8
	if availableWidth < 5 {
		availableWidth = 5
	}

	for i, k := range keys {
		val := displayData[k]

		// Bar calculation
		barWidth := 0
		if maxVal > 0 {
			barWidth = int(float64(val) / float64(maxVal) * float64(availableWidth))
		}
		if barWidth < 1 {
			barWidth = 1
		}

		bar := strings.Repeat("▇", barWidth)

		// Cycle colors
		var style lipgloss.Style
		switch i % 3 {
		case 0:
			style = lipgloss.NewStyle().Foreground(ColorPrimary)
		case 1:
			style = lipgloss.NewStyle().Foreground(ColorSecondary)
		case 2:
			style = lipgloss.NewStyle().Foreground(ColorInfo)
		}

		fmt.Fprintf(&sb, "%-*s %s %s\n",
			maxLabelWidth, k,
			style.Render(bar),
			StyleStatValue.Render(fmt.Sprintf("%d", val)))
	}

	return sb.String()
}
