package main

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	colorPrimary   = lipgloss.Color("#7D56F4")
	colorSecondary = lipgloss.Color("#6C6C6C")
	colorSuccess   = lipgloss.Color("#04B575")
	colorWarning   = lipgloss.Color("#FFCC00")
	colorError     = lipgloss.Color("#FF4444")
	colorInfo      = lipgloss.Color("#00AAFF")
	colorMuted     = lipgloss.Color("#626262")
	colorWhite     = lipgloss.Color("#FAFAFA")
	colorDim       = lipgloss.Color("#999999")
	colorCyan      = lipgloss.Color("#00CCCC")
	colorGreen     = lipgloss.Color("#04B575")
	colorYellow    = lipgloss.Color("#FFCC00")
	colorRed       = lipgloss.Color("#FF4444")
	colorBlue      = lipgloss.Color("#5B9BD5")
	colorMagenta   = lipgloss.Color("#CC66CC")
)

// Text styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorCyan)

	labelStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(18).
			Align(lipgloss.Right).
			PaddingRight(1)

	valueStyle = lipgloss.NewStyle().
			Foreground(colorWhite)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorDim)

	boldStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	successStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorWarning)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError)

	infoStyle = lipgloss.NewStyle().
			Foreground(colorInfo)
)

// Component styles
var (
	cursorStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(colorSuccess).
				Bold(true)

	unselectedItemStyle = lipgloss.NewStyle().
				Foreground(colorWhite)

	previewStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	checkedStyle = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	uncheckedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)
)

// Box / panel styles
var (
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPrimary).
			Padding(1, 2)

	summaryBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorCyan).
			Padding(1, 2)

	warningBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorWarning).
			Padding(0, 2)

	errorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorError).
			Padding(0, 2)

	successBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorSuccess).
			Padding(1, 2)
)

// Status indicator glyphs (ASCII only per project rules)
const (
	glyphSelected   = ">"
	glyphUnselected = " "
	glyphChecked    = "[x]"
	glyphUnchecked  = "[ ]"
	glyphSuccess    = "[ok]"
	glyphFailed     = "[!!]"
	glyphPending    = "[..]"
	glyphArrow      = "->"
	glyphBullet     = "*"
	glyphWarning    = "/!\\"
	glyphInfo       = "(i)"
)

// Helper to render a key-value info line for summaries
func renderInfoLine(label, value string) string {
	return labelStyle.Render(label+":") + " " + valueStyle.Render(value)
}

// Helper to render a warning line
func renderWarning(msg string) string {
	return warningStyle.Render(glyphWarning+" "+msg)
}

// Helper to render an error line
func renderError(msg string) string {
	return errorStyle.Render(glyphFailed+" "+msg)
}

// Helper to render a success line
func renderSuccess(msg string) string {
	return successStyle.Render(glyphSuccess+" "+msg)
}

// Helper to render a muted info line
func renderInfo(msg string) string {
	return infoStyle.Render(glyphInfo+" "+msg)
}

// Helper for step progress during execution
func renderExecStep(glyph, label string, style lipgloss.Style) string {
	return style.Render(glyph) + " " + valueStyle.Render(label)
}