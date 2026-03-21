package report

import (
	"fmt"
	"html"
	"html/template"
	"math"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/entro314-labs/techtell/internal/analysis"
)

// ChartItem represents a single data point for charts
type ChartItem struct {
	Name    string
	Count   int
	Percent float64
	Color   string
}

// HTMLReport generates a shareable, screenshot-friendly HTML report
func GenerateHTML(r *analysis.WorkspaceReport, rootPath string) string {

	type ReportData struct {
		RootPath        string
		Date            string
		TotalProjects   int
		TopLanguage     string
		TopStack        string
		TopPkgMgr       string
		Languages       []ChartItem
		Stacks          []ChartItem
		PkgManagers     []ChartItem
		PieChartPaths   template.HTML
		RadarChartPaths template.HTML
		StackedBars     template.HTML
	}

	// Dev-oriented color palette (vibrant but professional)
	colors := []string{
		"#22d3ee", // Cyan
		"#a78bfa", // Violet
		"#f472b6", // Pink
		"#fb923c", // Orange
		"#4ade80", // Green
		"#facc15", // Yellow
		"#60a5fa", // Blue
		"#f87171", // Red
		"#34d399", // Emerald
		"#c084fc", // Purple
	}

	// Helper to get top item
	getTop := func(m map[string]int) string {
		topK := "N/A"
		maxV := 0
		for k, v := range m {
			if v > maxV {
				maxV = v
				topK = k
			}
		}
		return topK
	}

	// Sort and prepare languages
	langItems := make([]ChartItem, 0, len(r.LangCounts))
	totalLang := 0
	for _, v := range r.LangCounts {
		totalLang += v
	}
	for k, v := range r.LangCounts {
		if k == "" {
			k = "Unknown"
		}
		langItems = append(langItems, ChartItem{
			Name:    k,
			Count:   v,
			Percent: float64(v) / float64(totalLang) * 100,
		})
	}
	sort.Slice(langItems, func(i, j int) bool {
		return langItems[i].Count > langItems[j].Count
	})
	if len(langItems) > 8 {
		langItems = langItems[:8]
	}
	for i := range langItems {
		langItems[i].Color = colors[i%len(colors)]
	}

	// Sort and prepare stacks
	stackItems := make([]ChartItem, 0, len(r.StackCounts))
	maxStack := 0
	for _, v := range r.StackCounts {
		if v > maxStack {
			maxStack = v
		}
	}
	for k, v := range r.StackCounts {
		if k == "" {
			k = "Unknown"
		}
		stackItems = append(stackItems, ChartItem{
			Name:    k,
			Count:   v,
			Percent: float64(v) / float64(maxStack) * 100,
		})
	}
	sort.Slice(stackItems, func(i, j int) bool {
		return stackItems[i].Count > stackItems[j].Count
	})
	if len(stackItems) > 10 {
		stackItems = stackItems[:10]
	}
	for i := range stackItems {
		stackItems[i].Color = colors[i%len(colors)]
	}

	// Prepare package managers
	pkgItems := make([]ChartItem, 0, len(r.PkgMgrCounts))
	maxPkg := 0
	for _, v := range r.PkgMgrCounts {
		if v > maxPkg {
			maxPkg = v
		}
	}
	for k, v := range r.PkgMgrCounts {
		if k == "" {
			k = "Unknown"
		}
		pkgItems = append(pkgItems, ChartItem{
			Name:    k,
			Count:   v,
			Percent: float64(v) / float64(maxPkg) * 100,
		})
	}
	sort.Slice(pkgItems, func(i, j int) bool {
		return pkgItems[i].Count > pkgItems[j].Count
	})
	if len(pkgItems) > 6 {
		pkgItems = pkgItems[:6]
	}
	for i := range pkgItems {
		pkgItems[i].Color = colors[i%len(colors)]
	}

	// Generate SVG pie chart paths
	pieChartPaths := generatePieChart(langItems, 100, 100, 80)

	// Generate radar chart for tech categories
	radarChartPaths := generateRadarChart(langItems, 100, 100, 70)

	// Generate stacked horizontal bars
	stackedBars := generateStackedBars(langItems, stackItems)

	data := ReportData{
		RootPath:        rootPath,
		Date:            time.Now().Format("2006-01-02 15:04"),
		TotalProjects:   r.TotalProjects,
		TopLanguage:     getTop(r.LangCounts),
		TopStack:        getTop(r.StackCounts),
		TopPkgMgr:       getTop(r.PkgMgrCounts),
		Languages:       langItems,
		Stacks:          stackItems,
		PkgManagers:     pkgItems,
		PieChartPaths:   template.HTML(pieChartPaths),
		RadarChartPaths: template.HTML(radarChartPaths),
		StackedBars:     template.HTML(stackedBars),
	}

	tmpl := template.Must(template.New("report").Parse(htmlTemplate))
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Sprintf("Error generating HTML: %v", err)
	}

	return buf.String()
}

func generatePieChart(items []ChartItem, cx, cy, r float64) string {
	var sb strings.Builder
	startAngle := -90.0

	for _, item := range items {
		angle := item.Percent / 100 * 360
		endAngle := startAngle + angle

		startRad := startAngle * math.Pi / 180
		endRad := endAngle * math.Pi / 180

		x1 := cx + r*math.Cos(startRad)
		y1 := cy + r*math.Sin(startRad)
		x2 := cx + r*math.Cos(endRad)
		y2 := cy + r*math.Sin(endRad)

		largeArc := "0"
		if angle > 180 {
			largeArc = "1"
		}

        fmt.Fprintf(&sb,
			`<path d="M %.1f %.1f L %.1f %.1f A %.1f %.1f 0 %s 1 %.1f %.1f Z" fill="%s" stroke="#0a0a0f" stroke-width="2"/>`,
			cx, cy, x1, y1, r, r, largeArc, x2, y2, item.Color,
        )

		startAngle = endAngle
	}

	return sb.String()
}

func generateRadarChart(items []ChartItem, cx, cy, maxR float64) string {
	if len(items) < 3 {
		return ""
	}

	var sb strings.Builder
	n := len(items)
	angleStep := 2 * math.Pi / float64(n)

	// Draw grid circles
	for i := 1; i <= 4; i++ {
		r := maxR * float64(i) / 4
        fmt.Fprintf(&sb,
			`<circle cx="%.1f" cy="%.1f" r="%.1f" fill="none" stroke="#1e293b" stroke-width="1"/>`,
			cx, cy, r,
        )
	}

	// Draw axis lines
	for i := 0; i < n; i++ {
		angle := float64(i)*angleStep - math.Pi/2
		x := cx + maxR*math.Cos(angle)
		y := cy + maxR*math.Sin(angle)
        fmt.Fprintf(&sb,
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#1e293b" stroke-width="1"/>`,
			cx, cy, x, y,
        )
	}

	// Find max for scaling
	maxVal := 0.0
	for _, item := range items {
		if item.Percent > maxVal {
			maxVal = item.Percent
		}
	}

	// Draw data polygon
	var points strings.Builder
	for i, item := range items {
		angle := float64(i)*angleStep - math.Pi/2
		r := (item.Percent / maxVal) * maxR * 0.9
		x := cx + r*math.Cos(angle)
		y := cy + r*math.Sin(angle)
		if i > 0 {
			points.WriteString(" ")
		}
        fmt.Fprintf(&points, "%.1f,%.1f", x, y)
	}
    fmt.Fprintf(&sb,
		`<polygon points="%s" fill="rgba(34, 211, 238, 0.3)" stroke="#22d3ee" stroke-width="2"/>`,
		points.String(),
    )

	// Draw data points
	for i, item := range items {
		angle := float64(i)*angleStep - math.Pi/2
		r := (item.Percent / maxVal) * maxR * 0.9
		x := cx + r*math.Cos(angle)
		y := cy + r*math.Sin(angle)
        fmt.Fprintf(&sb,
			`<circle cx="%.1f" cy="%.1f" r="4" fill="%s"/>`,
			x, y, item.Color,
        )
	}

	// Draw labels
	for i, item := range items {
		angle := float64(i)*angleStep - math.Pi/2
		labelR := maxR + 15
		x := cx + labelR*math.Cos(angle)
		y := cy + labelR*math.Sin(angle)
		anchor := "middle"
		if x < cx-10 {
			anchor = "end"
		} else if x > cx+10 {
			anchor = "start"
		}
        fmt.Fprintf(&sb,
			`<text x="%.1f" y="%.1f" fill="#94a3b8" font-size="10" font-family="JetBrains Mono, monospace" text-anchor="%s">%s</text>`,
			x, y+3, anchor, html.EscapeString(truncateLabel(item.Name, 12)),
        )
	}

	return sb.String()
}

func generateStackedBars(langs []ChartItem, stacks []ChartItem) string {
	var sb strings.Builder

	// Stacked bar for languages
	totalWidth := 0.0
	for _, item := range langs {
		totalWidth += item.Percent
	}

	x := 0.0
	for _, item := range langs {
		width := (item.Percent / totalWidth) * 100
        fmt.Fprintf(&sb,
			`<rect x="%.2f%%" y="0" width="%.2f%%" height="24" fill="%s"/>`,
			x, width, item.Color,
        )
		x += width
	}

	return sb.String()
}

func truncateLabel(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-2] + ".."
}

// SaveHTMLToFile writes HTML content to a file
func SaveHTMLToFile(filename string, content string) error {
	return os.WriteFile(filename, []byte(content), 0644)
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>TechTell // Dev Stack Report</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
            background: #0a0a0f;
            color: #e2e8f0;
            min-height: 100vh;
            padding: 32px;
        }

        .container {
            max-width: 1000px;
            margin: 0 auto;
        }

        .header {
            border: 1px solid #1e293b;
            padding: 24px;
            margin-bottom: 24px;
            background: linear-gradient(180deg, #0f172a 0%, #0a0a0f 100%);
        }

        .header-top {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 16px;
        }

        .logo {
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .logo-icon {
            font-size: 1.5rem;
        }

        .logo-text {
            font-size: 1.25rem;
            font-weight: 700;
            color: #22d3ee;
            letter-spacing: -0.02em;
        }

        .logo-text span {
            color: #64748b;
            font-weight: 400;
        }

        .timestamp {
            font-size: 0.75rem;
            color: #64748b;
        }

        .title {
            font-size: 2rem;
            font-weight: 700;
            color: #f8fafc;
            margin-bottom: 8px;
            letter-spacing: -0.03em;
        }

        .subtitle {
            font-size: 0.8rem;
            color: #64748b;
        }

        .subtitle code {
            color: #94a3b8;
            background: #1e293b;
            padding: 2px 6px;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(4, 1fr);
            gap: 2px;
            margin-bottom: 24px;
            background: #1e293b;
            border: 1px solid #1e293b;
        }

        .stat-card {
            background: #0a0a0f;
            padding: 20px;
            text-align: center;
        }

        .stat-card .value {
            font-size: 2rem;
            font-weight: 700;
            color: #22d3ee;
            margin-bottom: 4px;
        }

        .stat-card .label {
            font-size: 0.65rem;
            text-transform: uppercase;
            letter-spacing: 0.1em;
            color: #64748b;
        }

        .stat-card.highlight .value {
            color: #a78bfa;
        }

        .charts-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 2px;
            margin-bottom: 24px;
            background: #1e293b;
            border: 1px solid #1e293b;
        }

        .chart-card {
            background: #0a0a0f;
            padding: 20px;
        }

        .chart-card h3 {
            font-size: 0.7rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.1em;
            color: #64748b;
            margin-bottom: 16px;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .chart-card h3::before {
            content: '//';
            color: #22d3ee;
        }

        .chart-container {
            display: flex;
            align-items: center;
            gap: 20px;
        }

        .pie-chart, .radar-chart {
            width: 160px;
            height: 160px;
            flex-shrink: 0;
        }

        .pie-chart svg, .radar-chart svg {
            width: 100%;
            height: 100%;
        }

        .legend {
            flex: 1;
        }

        .legend-item {
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 4px 0;
            font-size: 0.75rem;
        }

        .legend-dot {
            width: 8px;
            height: 8px;
            flex-shrink: 0;
        }

        .legend-name {
            flex: 1;
            color: #94a3b8;
        }

        .legend-count {
            color: #64748b;
            font-weight: 500;
        }

        .bar-chart {
            display: flex;
            flex-direction: column;
            gap: 8px;
        }

        .bar-item {
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .bar-label {
            width: 100px;
            font-size: 0.7rem;
            color: #64748b;
            text-align: right;
            flex-shrink: 0;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
        }

        .bar-wrapper {
            flex: 1;
            height: 20px;
            background: #1e293b;
            position: relative;
        }

        .bar-fill {
            height: 100%;
        }

        .bar-count {
            width: 40px;
            font-size: 0.7rem;
            color: #64748b;
            text-align: left;
        }

        .stacked-section {
            border: 1px solid #1e293b;
            padding: 20px;
            margin-bottom: 24px;
            background: #0a0a0f;
        }

        .stacked-section h3 {
            font-size: 0.7rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.1em;
            color: #64748b;
            margin-bottom: 12px;
        }

        .stacked-section h3::before {
            content: '// ';
            color: #22d3ee;
        }

        .stacked-bar {
            height: 24px;
            display: flex;
            margin-bottom: 8px;
        }

        .stacked-legend {
            display: flex;
            flex-wrap: wrap;
            gap: 12px 24px;
        }

        .footer {
            border: 1px solid #1e293b;
            padding: 20px;
            text-align: center;
            background: #0a0a0f;
        }

        .footer .tagline {
            font-size: 0.9rem;
            color: #94a3b8;
            margin-bottom: 8px;
        }

        .footer .credits {
            font-size: 0.65rem;
            color: #475569;
        }

        .footer .credits a {
            color: #22d3ee;
            text-decoration: none;
        }

        @media (max-width: 768px) {
            .stats-grid {
                grid-template-columns: repeat(2, 1fr);
            }
            .charts-grid {
                grid-template-columns: 1fr;
            }
            .chart-container {
                flex-direction: column;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <header class="header">
            <div class="header-top">
                <div class="logo">
                    <span class="logo-icon">⌘</span>
                    <span class="logo-text">techtell<span>.report</span></span>
                </div>
                <span class="timestamp">{{.Date}}</span>
            </div>
            <h1 class="title">Your Tech DNA</h1>
            <p class="subtitle">scanned: <code>{{.RootPath}}</code></p>
        </header>

        <div class="stats-grid">
            <div class="stat-card">
                <div class="value">{{.TotalProjects}}</div>
                <div class="label">Projects</div>
            </div>
            <div class="stat-card highlight">
                <div class="value">{{.TopLanguage}}</div>
                <div class="label">Top Language</div>
            </div>
            <div class="stat-card">
                <div class="value">{{.TopStack}}</div>
                <div class="label">Top Stack</div>
            </div>
            <div class="stat-card">
                <div class="value">{{.TopPkgMgr}}</div>
                <div class="label">Pkg Manager</div>
            </div>
        </div>

        <div class="stacked-section">
            <h3>Language Distribution</h3>
            <svg width="100%" height="24" preserveAspectRatio="none">
                {{.StackedBars}}
            </svg>
            <div class="stacked-legend">
                {{range .Languages}}
                <div class="legend-item">
                    <div class="legend-dot" style="background-color: {{.Color}}"></div>
                    <span class="legend-name">{{.Name}}</span>
                    <span class="legend-count">({{.Count}})</span>
                </div>
                {{end}}
            </div>
        </div>

        <div class="charts-grid">
            <div class="chart-card">
                <h3>Languages</h3>
                <div class="chart-container">
                    <div class="pie-chart">
                        <svg viewBox="0 0 200 200">
                            {{.PieChartPaths}}
                        </svg>
                    </div>
                    <div class="legend">
                        {{range .Languages}}
                        <div class="legend-item">
                            <div class="legend-dot" style="background-color: {{.Color}}"></div>
                            <span class="legend-name">{{.Name}}</span>
                            <span class="legend-count">{{.Count}}</span>
                        </div>
                        {{end}}
                    </div>
                </div>
            </div>

            <div class="chart-card">
                <h3>Tech Radar</h3>
                <div class="chart-container">
                    <div class="radar-chart">
                        <svg viewBox="0 0 200 200">
                            {{.RadarChartPaths}}
                        </svg>
                    </div>
                </div>
            </div>
        </div>

        <div class="charts-grid">
            <div class="chart-card">
                <h3>Stacks & Frameworks</h3>
                <div class="bar-chart">
                    {{range .Stacks}}
                    <div class="bar-item">
                        <span class="bar-label">{{.Name}}</span>
                        <div class="bar-wrapper">
                            <div class="bar-fill" style="width: {{printf "%.1f" .Percent}}%; background: {{.Color}};"></div>
                        </div>
                        <span class="bar-count">{{.Count}}</span>
                    </div>
                    {{end}}
                </div>
            </div>

            <div class="chart-card">
                <h3>Package Managers</h3>
                <div class="bar-chart">
                    {{range .PkgManagers}}
                    <div class="bar-item">
                        <span class="bar-label">{{.Name}}</span>
                        <div class="bar-wrapper">
                            <div class="bar-fill" style="width: {{printf "%.1f" .Percent}}%; background: {{.Color}};"></div>
                        </div>
                        <span class="bar-count">{{.Count}}</span>
                    </div>
                    {{end}}
                </div>
            </div>
        </div>

        <footer class="footer">
            <p class="tagline">"hey, that's mostly my stacks and langs — what's yours?"</p>
            <p class="credits">generated by <a href="#">techtell</a> // share your coding dna</p>
        </footer>
    </div>
</body>
</html>
`
