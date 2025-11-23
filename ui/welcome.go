package ui

import (
	"strings"

	"badger_explorer_core/config"
	"badger_explorer_core/locale"
	"badger_explorer_core/pkg"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type WelcomeModel struct {
	cfg       *config.Config
	styles    pkg.Styles
	cursor    int
	recentDBs []string
	width     int
	height    int
}

func NewWelcomeModel(cfg *config.Config) WelcomeModel {
	return WelcomeModel{
		cfg:       cfg,
		styles:    pkg.DefaultStyles(),
		recentDBs: cfg.GetRecentDBs(),
	}
}

func (m WelcomeModel) Init() tea.Cmd {
	return nil
}

func (m WelcomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			// Menu items: Open DB, Config, Exit (3 items) + Recent DBs
			totalItems := 3 + len(m.recentDBs)
			if m.cursor < totalItems-1 {
				m.cursor++
			}
		case "enter":
			if m.cursor == 0 {
				// Open DB Picker
				return m, func() tea.Msg { return OpenPickerMsg{} }
			} else if m.cursor == 1 {
				// Config
				return m, func() tea.Msg { return OpenConfigMsg{} }
			} else if m.cursor == 2 {
				// Exit
				return m, tea.Quit
			} else {
				// Recent DB
				idx := m.cursor - 3
				if idx >= 0 && idx < len(m.recentDBs) {
					return m, func() tea.Msg { return OpenDBMsg{Path: m.recentDBs[idx]} }
				}
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m WelcomeModel) View() string {
	logo := `Badger Explorer Core`
	logoView := m.styles.Logo.Render(logo)

	// Menu
	menuItems := []string{
		locale.T("open_db"),
		locale.T("config"),
		locale.T("exit"),
	}

	var menuView strings.Builder
	for i, item := range menuItems {
		if m.cursor == i {
			// Selected: "> Item" (No background effect)
			menuView.WriteString(m.styles.Highlight.Render("> "+item) + "\n")
		} else {
			// Normal: "  Item"
			menuView.WriteString("  " + item + "\n")
		}
	}

	var recentView strings.Builder
	if len(m.recentDBs) > 0 {
		recentView.WriteString("\n" + m.styles.Dimmed.Render(locale.T("recent_dbs")) + ":\n")
		for i, dbPath := range m.recentDBs {
			if m.cursor == i+3 {
				recentView.WriteString(m.styles.Highlight.Render("> "+dbPath) + "\n")
			} else {
				recentView.WriteString("  " + dbPath + "\n")
			}
		}
	}

	helpView := m.styles.Help.Render("Use " + m.styles.HelpKey.Render("↑/↓") + " to navigate, " + m.styles.HelpKey.Render("Enter") + " to select")

	content := lipgloss.JoinVertical(lipgloss.Center,
		logoView,
		"",
		menuView.String(),
		recentView.String(),
		"",
		helpView,
	)

	// Center the content in the window
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// Messages
type OpenPickerMsg struct{}
type OpenConfigMsg struct{}
type OpenDBMsg struct{ Path string }
