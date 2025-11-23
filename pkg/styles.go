package pkg

import (
	"github.com/charmbracelet/lipgloss"
)

// Color Palette (Dracula-inspired)
const (
	ColorBackground  = "#282A36"
	ColorCurrentLine = "#44475A"
	ColorForeground  = "#F8F8F2"
	ColorComment     = "#6272A4"
	ColorCyan        = "#8BE9FD"
	ColorGreen       = "#50FA7B"
	ColorOrange      = "#FFB86C"
	ColorPink        = "#FF79C6"
	ColorPurple      = "#BD93F9"
	ColorRed         = "#FF5555"
	ColorYellow      = "#F1FA8C"
)

// Styles defines the application styles.
type Styles struct {
	Title        lipgloss.Style
	Help         lipgloss.Style
	HelpKey      lipgloss.Style
	HelpDesc     lipgloss.Style
	Error        lipgloss.Style
	Success      lipgloss.Style
	Highlight    lipgloss.Style
	Dimmed       lipgloss.Style
	Border       lipgloss.Style
	Focused      lipgloss.Style
	Normal       lipgloss.Style
	SelectedItem lipgloss.Style
	Container    lipgloss.Style
	Logo         lipgloss.Style
}

// DefaultStyles returns the default styles.
func DefaultStyles() Styles {
	return Styles{
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorPurple)).
			Background(lipgloss.Color(ColorCurrentLine)).
			Padding(0, 1).
			MarginBottom(1),
		Help: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorComment)).
			MarginTop(1),
		HelpKey: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorCyan)).
			Bold(true),
		HelpDesc: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorComment)),
		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorRed)).
			Bold(true),
		Success: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGreen)).
			Bold(true),
		Highlight: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorOrange)),
		Dimmed: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorComment)),
		Border: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorPurple)),
		Focused: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorPink)),
		Normal: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorComment)),
		SelectedItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorBackground)).
			Background(lipgloss.Color(ColorPink)).
			Bold(true).
			Padding(0, 1),
		Container: lipgloss.NewStyle().
			Padding(1, 2),
		Logo: lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorOrange)).
			Bold(true).
			MarginBottom(1),
	}
}
