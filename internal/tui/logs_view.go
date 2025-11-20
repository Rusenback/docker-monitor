package tui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/rusenback/docker-monitor/internal/model"
)

var (
	// Log level patterns
	errorPattern   = regexp.MustCompile(`(?i)\b(error|err|fatal|fail|failed|exception|panic)\b`)
	warningPattern = regexp.MustCompile(`(?i)\b(warn|warning|caution)\b`)
	infoPattern    = regexp.MustCompile(`(?i)\b(info|information)\b`)
	debugPattern   = regexp.MustCompile(`(?i)\b(debug|trace)\b`)

	// Pattern highlighting
	ipPattern   = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`)
	urlPattern  = regexp.MustCompile(`https?://[^\s]+`)
	pathPattern = regexp.MustCompile(`(/[\w\-./]+)+`)

	// Styles for log levels
	timestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086")) // Dim gray

	errorLogStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8"))   // Red
	warningLogStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FAB387"))   // Orange
	infoLogStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA"))   // Blue
	debugLogStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C7086"))   // Dim
	defaultLogStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#CDD6F4"))   // Normal

	// Stream indicators
	stdoutIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#A6E3A1")).Render("○") // Green circle
	stderrIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8")).Render("●") // Red circle

	// Highlight styles
	ipStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#F9E2AF"))   // Yellow
	urlStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#89DCEB"))   // Cyan
	pathStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#CBA6F7"))   // Purple
)

// styleLogEntry applies styling to a log entry
func styleLogEntry(entry model.LogEntry, maxWidth int) string {
	// Format timestamp (dimmed)
	timestamp := timestampStyle.Render(entry.Timestamp.Format("15:04:05"))

	// Stream indicator
	streamIndicator := stdoutIndicator
	if entry.Stream == "stderr" {
		streamIndicator = stderrIndicator
	}

	// Style the message based on log level
	message := entry.Message
	var styledMessage string

	// Detect log level and apply appropriate style
	switch {
	case errorPattern.MatchString(message):
		styledMessage = styleMessage(message, errorLogStyle)
	case warningPattern.MatchString(message):
		styledMessage = styleMessage(message, warningLogStyle)
	case infoPattern.MatchString(message):
		styledMessage = styleMessage(message, infoLogStyle)
	case debugPattern.MatchString(message):
		styledMessage = styleMessage(message, debugLogStyle)
	default:
		styledMessage = styleMessage(message, defaultLogStyle)
	}

	// Combine all parts
	logLine := timestamp + " " + streamIndicator + " " + styledMessage

	// Truncate if needed (accounting for ANSI codes)
	if lipgloss.Width(logLine) > maxWidth {
		// Calculate how much to keep
		overhead := lipgloss.Width(timestamp) + lipgloss.Width(streamIndicator) + 5 // spaces + "..."
		keepLength := maxWidth - overhead
		if keepLength > 0 {
			styledMessage = truncateStyled(styledMessage, keepLength)
			logLine = timestamp + " " + streamIndicator + " " + styledMessage + "..."
		}
	}

	return logLine
}

// styleMessage applies base style and highlights patterns
func styleMessage(message string, baseStyle lipgloss.Style) string {
	result := message

	// Highlight IPs
	result = ipPattern.ReplaceAllStringFunc(result, func(match string) string {
		return ipStyle.Render(match)
	})

	// Highlight URLs
	result = urlPattern.ReplaceAllStringFunc(result, func(match string) string {
		return urlStyle.Render(match)
	})

	// Highlight paths (but avoid over-highlighting)
	if strings.Count(result, "/") >= 2 {
		result = pathPattern.ReplaceAllStringFunc(result, func(match string) string {
			// Only highlight if it looks like a real path (has at least 2 segments)
			if strings.Count(match, "/") >= 2 {
				return pathStyle.Render(match)
			}
			return match
		})
	}

	return result
}

// truncateStyled truncates a styled string to a maximum visible width
func truncateStyled(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	// Simple truncation - just take first maxWidth runes
	// This isn't perfect with ANSI codes but works for our case
	runes := []rune(s)
	if len(runes) > maxWidth {
		return string(runes[:maxWidth])
	}
	return s
}
