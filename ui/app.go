package ui

import (
	"badger_explorer_core/config"
	"badger_explorer_core/db"

	tea "github.com/charmbracelet/bubbletea"
)

type sessionState int

const (
	stateWelcome sessionState = iota
	stateDBPicker
	stateDBMain
	stateDetail
	stateInsert
	stateConfig
)

type AppModel struct {
	state    sessionState
	cfg      *config.Config
	dbClient *db.DBClient

	welcome  WelcomeModel
	dbPicker DBPickerModel
	dbMain   DBMainModel
	detail   DetailModel
	insert   InsertModel
	config   ConfigModel

	width  int
	height int
}

func NewAppModel(cfg *config.Config, dbClient *db.DBClient) AppModel {
	return AppModel{
		state:    stateWelcome,
		cfg:      cfg,
		dbClient: dbClient,
		welcome:  NewWelcomeModel(cfg),
		dbPicker: NewDBPickerModel(),
		dbMain:   NewDBMainModel(dbClient, cfg),
		detail:   NewDetailModel(dbClient, cfg, ""), // Empty key initially
		insert:   NewInsertModel(dbClient, cfg),
		config:   NewConfigModel(cfg),
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.welcome.Init(),
	)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Propagate size
		updatedWelcome, _ := updateModel(m.welcome, msg)
		m.welcome = updatedWelcome.(WelcomeModel)

		updatedPicker, _ := updateModel(m.dbPicker, msg)
		m.dbPicker = updatedPicker.(DBPickerModel)

		updatedMain, _ := updateModel(m.dbMain, msg)
		m.dbMain = updatedMain.(DBMainModel)

		updatedDetail, _ := updateModel(m.detail, msg)
		m.detail = updatedDetail.(DetailModel)

		updatedInsert, _ := updateModel(m.insert, msg)
		m.insert = updatedInsert.(InsertModel)

		updatedConfig, _ := updateModel(m.config, msg)
		m.config = updatedConfig.(ConfigModel)

	// Navigation Messages
	case OpenPickerMsg:
		m.state = stateDBPicker
		m.dbPicker = NewDBPickerModel() // Reset
		updatedPicker, _ := updateModel(m.dbPicker, tea.WindowSizeMsg{Width: m.width, Height: m.height})
		m.dbPicker = updatedPicker.(DBPickerModel)
		return m, m.dbPicker.Init()

	case OpenConfigMsg:
		m.state = stateConfig
		m.config = NewConfigModel(m.cfg)
		return m, m.config.Init()

	case OpenDBMsg:
		// Try to open DB
		err := m.dbClient.Open(msg.Path) // Always RW
		if err != nil {
			// Show error in welcome?
			// For now, just print and exit or stay?
			// Ideally show error popup.
			// Let's go to main but with error?
			// Or stay in Welcome.
			// Let's assume success or panic for now (simple), or handle error properly.
			// We can pass error to Welcome model?
			// Show error in picker
			m.dbPicker.err = err
			// Force update picker view to show error
			return m, nil
		}

		// Add to recent
		m.cfg.AddRecentDB(msg.Path)
		m.cfg.Save()

		m.state = stateDBMain
		m.dbMain = NewDBMainModel(m.dbClient, m.cfg)
		updatedMain, _ := updateModel(m.dbMain, tea.WindowSizeMsg{Width: m.width, Height: m.height})
		m.dbMain = updatedMain.(DBMainModel)
		return m, m.dbMain.Init()

	case BackToWelcomeMsg:
		if m.dbClient.IsOpen() {
			m.dbClient.Close()
		}
		m.state = stateWelcome
		// Refresh recent DBs
		m.welcome = NewWelcomeModel(m.cfg)
		updatedWelcome, _ := updateModel(m.welcome, tea.WindowSizeMsg{Width: m.width, Height: m.height})
		m.welcome = updatedWelcome.(WelcomeModel)
		return m, nil

	case OpenDetailMsg:
		m.state = stateDetail
		m.detail = NewDetailModel(m.dbClient, m.cfg, msg.Key)
		updatedDetail, _ := updateModel(m.detail, tea.WindowSizeMsg{Width: m.width, Height: m.height})
		m.detail = updatedDetail.(DetailModel)
		return m, m.detail.Init()

	case BackToMainMsg:
		m.state = stateDBMain
		// Maybe refresh list?
		return m, nil // Main model keeps state

	case OpenInsertMsg:
		m.state = stateInsert
		m.insert = NewInsertModel(m.dbClient, m.cfg)
		updatedModel, _ := updateModel(m.insert, tea.WindowSizeMsg{Width: m.width, Height: m.height})
		m.insert = updatedModel.(InsertModel)
		return m, m.insert.Init()
	}

	// Delegate Update
	switch m.state {
	case stateWelcome:
		newModel, newCmd := m.welcome.Update(msg)
		m.welcome = newModel.(WelcomeModel)
		cmd = newCmd
	case stateDBPicker:
		newModel, newCmd := m.dbPicker.UpdateWithKey(msg)
		m.dbPicker = newModel.(DBPickerModel)
		cmd = newCmd
	case stateDBMain:
		newModel, newCmd := m.dbMain.Update(msg)
		m.dbMain = newModel.(DBMainModel)
		cmd = newCmd
	case stateDetail:
		newModel, newCmd := m.detail.Update(msg)
		m.detail = newModel.(DetailModel)
		cmd = newCmd
	case stateInsert:
		newModel, newCmd := m.insert.Update(msg)
		m.insert = newModel.(InsertModel)
		cmd = newCmd
	case stateConfig:
		newModel, newCmd := m.config.Update(msg)
		m.config = newModel.(ConfigModel)
		cmd = newCmd
	}

	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m AppModel) View() string {
	switch m.state {
	case stateWelcome:
		return m.welcome.View()
	case stateDBPicker:
		return m.dbPicker.View()
	case stateDBMain:
		return m.dbMain.View()
	case stateDetail:
		return m.detail.View()
	case stateInsert:
		return m.insert.View()
	case stateConfig:
		return m.config.View()
	}
	return "Unknown state"
}

// Helper to update sub-models with type assertion
func updateModel(m tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	return m.Update(msg)
}
