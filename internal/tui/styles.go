package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors (Dev-oriented Terminal Palette)
	ColorPrimary   = lipgloss.AdaptiveColor{Light: "#0ea5e9", Dark: "#22d3ee"} // Cyan
	ColorSecondary = lipgloss.AdaptiveColor{Light: "#7c3aed", Dark: "#a78bfa"} // Violet
	ColorSuccess   = lipgloss.AdaptiveColor{Light: "#16a34a", Dark: "#4ade80"} // Green
	ColorError     = lipgloss.AdaptiveColor{Light: "#dc2626", Dark: "#f87171"} // Red
	ColorWarning   = lipgloss.AdaptiveColor{Light: "#d97706", Dark: "#fb923c"} // Orange
	ColorInfo      = lipgloss.AdaptiveColor{Light: "#0891b2", Dark: "#22d3ee"} // Cyan
	ColorGray      = lipgloss.AdaptiveColor{Light: "#64748b", Dark: "#475569"}
	ColorLightGray = lipgloss.AdaptiveColor{Light: "#94a3b8", Dark: "#94a3b8"}
	ColorDark      = lipgloss.AdaptiveColor{Light: "#f8fafc", Dark: "#0f172a"}
	ColorHighlight = lipgloss.AdaptiveColor{Light: "#e2e8f0", Dark: "#1e293b"}
	ColorBorder    = lipgloss.AdaptiveColor{Light: "#cbd5e1", Dark: "#334155"}

	// Styles - Dev-oriented with sharp corners

	// Base Title - Sharp edges, no rounded corners
	StyleTitle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Padding(0, 2).
			Align(lipgloss.Center).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			Width(60)

	StyleHelp      = lipgloss.NewStyle().Foreground(ColorGray).Italic(false)
	StyleError     = lipgloss.NewStyle().Foreground(ColorError).Bold(true)
	StyleSuccess   = lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true)
	StyleFocused   = lipgloss.NewStyle().Foreground(ColorPrimary).Bold(true)
	StyleUnfocused = lipgloss.NewStyle().Foreground(ColorGray)

	// Layout - Minimal padding
	StyleContainer = lipgloss.NewStyle().Padding(0, 1)
	StyleContent   = lipgloss.NewStyle().Padding(0)

	// Components - Sharp corners
	StyleItem = lipgloss.NewStyle().PaddingLeft(2)

	StyleSelectedItem = lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(ColorPrimary).
				Bold(true).
				BorderLeft(true).
				BorderStyle(lipgloss.ThickBorder()).
				BorderForeground(ColorPrimary)

	// Cards - Sharp corners for dev look
	StyleCard = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1).
			MarginRight(1)

	StyleHeader = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			MarginBottom(1)

	// Dashboard Specific - Sharp corners
	StyleStatBox = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 2).
			Align(lipgloss.Center).
			Width(20).
			MarginRight(1)

	StyleStatLabel = lipgloss.NewStyle().
			Foreground(ColorGray).
			Faint(true).
			Bold(false)

	StyleStatValue = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			PaddingTop(0)

	// Tabs - Sharp corners, terminal-inspired
	StyleTab = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorGray).
			Padding(0, 2).
			Foreground(ColorGray)

	StyleActiveTab = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorPrimary).
			BorderTop(true).
			BorderBottom(true).
			BorderLeft(true).
			BorderRight(true).
			Padding(0, 2).
			Foreground(ColorPrimary).
			Bold(true)

	// Progress Bar Style
	StyleProgressBar = lipgloss.NewStyle().
				Foreground(ColorPrimary)

	// Branding
	StyleBrand = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	StyleBrandDim = lipgloss.NewStyle().
			Foreground(ColorGray)
)
