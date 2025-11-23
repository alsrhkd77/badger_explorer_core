package ui

import (
	"fmt"
	"strings"

	"badger_explorer_core/config"
	"badger_explorer_core/db"
	"badger_explorer_core/locale"
	"badger_explorer_core/pkg"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type InsertModel struct {
	dbClient *db.DBClient
	cfg      *config.Config
	styles   pkg.Styles

	keyInput   textinput.Model
	valueInput textarea.Model

	focusIndex int // 0: key, 1: value

	err error
	msg string
}

func NewInsertModel(client *db.DBClient, cfg *config.Config) InsertModel {
	ki := textinput.New()
	ki.Placeholder = "Key"
	ki.Focus()

	vi := textarea.New()
	vi.Placeholder = "Value"

	return InsertModel{
		dbClient:   client,
		cfg:        cfg,
		styles:     pkg.DefaultStyles(),
		keyInput:   ki,
		valueInput: vi,
		focusIndex: 0,
	}
}

func (m InsertModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InsertModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return BackToMainMsg{} }
		case "tab":
			m.focusIndex = (m.focusIndex + 1) % 2
			if m.focusIndex == 0 {
				m.keyInput.Focus()
				m.valueInput.Blur()
			} else {
				m.keyInput.Blur()
				m.valueInput.Focus()
			}
			return m, nil
		case "ctrl+s":
			return m, m.saveCmd()
		}

	case OperationResultMsg:
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.msg = msg.Message
			// Clear inputs?
			m.keyInput.SetValue("")
			m.valueInput.SetValue("")
			m.keyInput.Focus()
			m.focusIndex = 0
		}
	}

	if m.focusIndex == 0 {
		m.keyInput, cmd = m.keyInput.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.valueInput, cmd = m.valueInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m InsertModel) View() string {
	s := strings.Builder{}

	s.WriteString(m.styles.Title.Render("Insert New Key") + "\n\n")

	if m.err != nil {
		s.WriteString(m.styles.Error.Render(m.err.Error()) + "\n")
	}
	if m.msg != "" {
		s.WriteString(m.styles.Success.Render(m.msg) + "\n")
	}

	s.WriteString("Key:\n")
	s.WriteString(m.keyInput.View() + "\n\n")

	s.WriteString("Value:\n")
	s.WriteString(m.valueInput.View() + "\n\n")

	s.WriteString(m.styles.Help.Render("Tab: Switch Focus | Ctrl+S: Save | Esc: Back"))

	return s.String()
}

func (m InsertModel) saveCmd() tea.Cmd {
	return func() tea.Msg {
		key := m.keyInput.Value()
		val := m.valueInput.Value()

		if key == "" {
			return OperationResultMsg{Op: "insert", Err: fmt.Errorf("key cannot be empty")}
		}

		// Check if exists?
		// Badger Set overwrites.
		// If we want to prevent overwrite, we should check first.
		// But spec says "Insert new key/value".

		// Auto backup if enabled (even for new key? No, backup is for existing value modification usually)
		// But if key exists, we might overwrite.
		// Let's check existence if backup is enabled.
		if m.cfg.DB.AutoBackupOnWrite {
			_, _ = m.dbClient.BackupValue(key, m.cfg.DB.BackupPath)
		}

		err := m.dbClient.SetValue(key, []byte(val), 0)
		if err != nil {
			return OperationResultMsg{Op: "insert", Err: err}
		}

		return OperationResultMsg{Op: "insert", Message: locale.T("save_success")}
	}
}
