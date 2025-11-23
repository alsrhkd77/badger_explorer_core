package ui

import (
	"fmt"
	"time"

	"badger_explorer_core/config"
	"badger_explorer_core/db"
	"badger_explorer_core/locale"
	"badger_explorer_core/pkg"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DBMainModel struct {
	dbClient *db.DBClient
	cfg      *config.Config
	styles   pkg.Styles

	table    table.Model
	searchIn textinput.Model

	keys      []db.KeyItem
	offset    int
	hasMore   bool
	isLoading bool

	searchMode string // "prefix", "substring", "regex"
	sortDesc   bool

	width  int
	height int

	err error

	// Debounce state
	searchID int
}

func NewDBMainModel(client *db.DBClient, cfg *config.Config) DBMainModel {
	styles := pkg.DefaultStyles()

	// Table init
	columns := []table.Column{
		{Title: "Key", Width: 30},
		{Title: "Preview", Width: 50},
		{Title: "Size", Width: 10},
		{Title: "Expires", Width: 20},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(pkg.ColorPurple)).
		BorderBottom(true).
		Bold(true).
		Foreground(lipgloss.Color(pkg.ColorCyan))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color(pkg.ColorBackground)).
		Background(lipgloss.Color(pkg.ColorPink)).
		Bold(true)
	t.SetStyles(s)

	// Search input init
	ti := textinput.New()
	ti.Placeholder = locale.T("search_placeholder")
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 50
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(pkg.ColorOrange))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(pkg.ColorForeground))

	return DBMainModel{
		dbClient:   client,
		cfg:        cfg,
		styles:     styles,
		table:      t,
		searchIn:   ti,
		searchMode: cfg.Search.DefaultMode,
		sortDesc:   false,
	}
}

func (m DBMainModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.fetchKeysCmd(),
	)
}

// SearchTickMsg is sent after debounce duration
type SearchTickMsg struct {
	ID int
}

func (m DBMainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			if m.searchIn.Focused() {
				m.searchIn.Blur()
				m.table.Focus()
			} else {
				// Back to Welcome?
				// Or close DB?
				return m, func() tea.Msg { return BackToWelcomeMsg{} }
			}
		case "/":
			if !m.searchIn.Focused() {
				m.searchIn.Focus()
				m.table.Blur()
				return m, textinput.Blink
			}
		case "enter":
			if m.table.Focused() {
				// Open detail
				selected := m.table.SelectedRow()
				if len(selected) > 0 {
					key := selected[0]
					return m, func() tea.Msg { return OpenDetailMsg{Key: key} }
				}
			} else if m.searchIn.Focused() {
				// Trigger search immediately (force)
				m.offset = 0
				m.searchID++ // Invalidate pending ticks
				cmds = append(cmds, m.fetchKeysCmd())
				m.searchIn.Blur()
				m.table.Focus()
			}
		case "s":
			if !m.searchIn.Focused() {
				m.sortDesc = !m.sortDesc
				m.offset = 0
				cmds = append(cmds, m.fetchKeysCmd())
			}
		case "ctrl+f":
			// Toggle search mode
			modes := []string{"prefix", "substring", "regex"}
			for i, mode := range modes {
				if m.searchMode == mode {
					m.searchMode = modes[(i+1)%len(modes)]
					break
				}
			}
			// Re-fetch?
			m.offset = 0
			cmds = append(cmds, m.fetchKeysCmd())
		case "i":
			if !m.searchIn.Focused() {
				return m, func() tea.Msg { return OpenInsertMsg{} }
			}
		case "right", "l":
			if !m.searchIn.Focused() && m.hasMore {
				m.offset += m.cfg.DB.OpenBatchSize
				cmds = append(cmds, m.fetchKeysCmd())
			}
		case "left", "h":
			if !m.searchIn.Focused() && m.offset > 0 {
				m.offset -= m.cfg.DB.OpenBatchSize
				if m.offset < 0 {
					m.offset = 0
				}
				cmds = append(cmds, m.fetchKeysCmd())
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate available height for table
		// Header (2) + SearchBar (3) + Footer (2) + Container Padding (2) = 9
		// Let's use 10 to be safe
		availableHeight := msg.Height - 10
		if availableHeight < 1 {
			availableHeight = 1
		}

		m.table.SetWidth(msg.Width - 4) // Container padding
		m.table.SetHeight(availableHeight)

	case KeysFetchedMsg:
		m.isLoading = false
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.keys = msg.Keys
			m.hasMore = msg.HasMore
			m.updateTable()
		}

	case SearchTickMsg:
		if msg.ID == m.searchID {
			m.offset = 0
			cmds = append(cmds, m.fetchKeysCmd())
		}
	}

	// 컴포넌트 처리
	if m.searchIn.Focused() {
		oldValue := m.searchIn.Value()
		m.searchIn, cmd = m.searchIn.Update(msg)
		cmds = append(cmds, cmd)

		newValue := m.searchIn.Value()
		if oldValue != newValue {
			m.searchID++
			// Debounce 400ms
			id := m.searchID
			cmds = append(cmds, tea.Tick(400*time.Millisecond, func(t time.Time) tea.Msg {
				return SearchTickMsg{ID: id}
			}))
		}
	} else {
		m.table, cmd = m.table.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *DBMainModel) updateTable() {
	rows := make([]table.Row, len(m.keys))
	for i, k := range m.keys {
		rows[i] = table.Row{
			k.Key,
			k.ValuePreview,
			fmt.Sprintf("%d", k.Size),
			fmt.Sprintf("%d", k.ExpiresAt),
		}
	}
	m.table.SetRows(rows)
}

func (m DBMainModel) View() string {
	// Header
	header := m.styles.Title.Render(fmt.Sprintf("DB: %s", m.dbClient.GetPath()))

	// Search Bar
	modeStr := fmt.Sprintf("[%s]", m.searchMode)
	searchBar := lipgloss.JoinHorizontal(lipgloss.Left,
		m.searchIn.View(),
		" ",
		m.styles.Dimmed.Render(modeStr),
	)
	searchBar = m.styles.Container.Copy().Padding(0, 1).Render(searchBar)

	// Table
	tableView := m.styles.Border.Render(m.table.View())

	// Footer
	helpText := "Enter: Detail | /: Search | s: Sort | i: Insert | ←/→: Page | Ctrl+F: Mode | Esc: Back"
	if m.isLoading {
		helpText += " | Loading..."
	}
	footer := m.styles.Help.Render(helpText)

	content := lipgloss.JoinVertical(lipgloss.Left,
		header,
		searchBar,
		tableView,
		footer,
	)

	return m.styles.Container.Render(content)
}

// Commands & Messages

type KeysFetchedMsg struct {
	Keys    []db.KeyItem
	HasMore bool
	Err     error
}

func (m DBMainModel) fetchKeysCmd() tea.Cmd {
	return func() tea.Msg {
		opts := db.ListKeysOptions{
			Prefix:       m.searchIn.Value(),
			Mode:         m.searchMode,
			SortDesc:     m.sortDesc,
			Limit:        m.cfg.DB.OpenBatchSize,
			Offset:       m.offset,
			PreviewChars: m.cfg.UI.PreviewChars,
		}

		// Simulate delay for spinner? No need.
		keys, hasMore, err := m.dbClient.ListKeys(opts)
		return KeysFetchedMsg{Keys: keys, HasMore: hasMore, Err: err}
	}
}

type OpenDetailMsg struct {
	Key string
}

type OpenInsertMsg struct{}
