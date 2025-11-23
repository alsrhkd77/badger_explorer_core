package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"badger_explorer_core/pkg"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DBPickerModel struct {
	styles      pkg.Styles
	currentPath string
	files       []os.DirEntry
	cursor      int
	offset      int // Scroll offset
	height      int
	width       int
	err         error
}

func NewDBPickerModel() DBPickerModel {
	home, err := os.UserHomeDir()
	if err != nil {
		home, _ = os.Getwd()
	}

	m := DBPickerModel{
		styles:      pkg.DefaultStyles(),
		currentPath: home,
	}
	m.loadFiles()
	return m
}

func (m *DBPickerModel) loadFiles() {
	entries, err := os.ReadDir(m.currentPath)
	if err != nil {
		m.err = err
		return
	}

	// Filter and sort
	var dirs []os.DirEntry
	for _, e := range entries {
		if e.IsDir() {
			// Skip hidden directories if needed, but let's show them for now or filter .git
			if strings.HasPrefix(e.Name(), ".") && len(e.Name()) > 1 {
				// Optional: skip hidden
			}
			dirs = append(dirs, e)
		}
	}

	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name()) < strings.ToLower(dirs[j].Name())
	})

	m.files = dirs
	m.cursor = 0
	m.offset = 0
	m.err = nil
}

func (m DBPickerModel) Init() tea.Cmd {
	return nil
}

func (m DBPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.UpdateWithKey(msg)
}

func (m DBPickerModel) UpdateWithKey(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Adjust offset if needed

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}
		case "down", "j":
			if m.cursor < len(m.files)-1 {
				m.cursor++
				// Visible height calculation (approximate, title + help takes space)
				listHeight := m.height - 6 // Title(1) + Path(1) + Spacing(1) + Help(1) + Spacing(1) (No Border)
				if listHeight < 1 {
					listHeight = 1
				}
				if m.cursor >= m.offset+listHeight {
					m.offset++
				}
			}
		case "left", "h", "backspace":
			// Go up
			parent := filepath.Dir(m.currentPath)
			if parent != m.currentPath {
				m.currentPath = parent
				m.loadFiles()
			}
		case "right", "l":
			// Enter directory
			if len(m.files) > 0 {
				selected := m.files[m.cursor]
				newPath := filepath.Join(m.currentPath, selected.Name())
				// Check permission?
				m.currentPath = newPath
				m.loadFiles()
			}
		case "enter":
			// Pick current selection as DB
			if len(m.files) > 0 {
				selected := m.files[m.cursor]
				path := filepath.Join(m.currentPath, selected.Name())
				return m, func() tea.Msg { return OpenDBMsg{Path: path} }
			}
		case " ":
			// Select current directory
			return m, func() tea.Msg { return OpenDBMsg{Path: m.currentPath} }
		case "esc":
			return m, func() tea.Msg { return BackToWelcomeMsg{} }
		}
	}
	return m, nil
}

func (m DBPickerModel) View() string {
	// Title
	title := m.styles.Title.Render("Select DB Directory")

	// Current Path
	path := m.styles.Highlight.Render(m.currentPath)
	path = m.styles.Container.Copy().Padding(0, 1).Render(path)

	if m.err != nil {
		return lipgloss.JoinVertical(lipgloss.Left,
			title,
			path,
			m.styles.Error.Render(m.err.Error()),
		)
	}

	// List
	listHeight := m.height - 6 // Title + Path + Help + Spacing (No Border)
	if listHeight < 1 {
		listHeight = 1
	}

	end := m.offset + listHeight
	if end > len(m.files) {
		end = len(m.files)
	}

	var listContent strings.Builder
	for i := m.offset; i < end; i++ {
		f := m.files[i]
		name := "📁 " + f.Name()
		if i == m.cursor {
			// Selected: "> Name" (No background)
			name = m.styles.Highlight.Render("> " + name)
		} else {
			name = "  " + name
		}
		listContent.WriteString(name + "\n")
	}

	// Fill empty lines
	for i := end - m.offset; i < listHeight; i++ {
		listContent.WriteString("\n")
	}

	// No Border
	listView := listContent.String()

	// Help
	help := m.styles.Help.Render("↑/↓: Move | ←/→: Navigate | Enter: Select Item | Space: Select Current Dir | Esc: Back")

	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		path,
		listView,
		help,
	)

	return m.styles.Container.Render(content)
}

type BackToWelcomeMsg struct{}
