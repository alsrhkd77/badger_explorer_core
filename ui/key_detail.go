package ui

import (
	"encoding/hex"
	"fmt"

	"badger_explorer_core/config"
	"badger_explorer_core/db"
	"badger_explorer_core/locale"
	"badger_explorer_core/pkg"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DetailModel struct {
	dbClient *db.DBClient
	cfg      *config.Config
	styles   pkg.Styles

	key       string
	value     []byte
	isHex     bool
	isEditing bool

	viewport viewport.Model
	textarea textarea.Model

	err error
	msg string // Success/Status message

	width  int
	height int
}

func NewDetailModel(client *db.DBClient, cfg *config.Config, key string) DetailModel {
	ta := textarea.New()
	ta.Placeholder = "Value..."
	ta.Focus()
	ta.ShowLineNumbers = true

	vp := viewport.New(0, 0)

	return DetailModel{
		dbClient: client,
		cfg:      cfg,
		styles:   pkg.DefaultStyles(),
		key:      key,
		textarea: ta,
		viewport: vp,
	}
}

func (m DetailModel) Init() tea.Cmd {
	return m.fetchValueCmd()
}

func (m DetailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.isEditing {
			switch msg.String() {
			case "esc":
				m.isEditing = false
				m.textarea.Blur()
				return m, nil
			case "ctrl+s":
				// Save
				return m, m.saveValueCmd()
			}
		} else {
			switch msg.String() {
			case "esc":
				return m, func() tea.Msg { return BackToMainMsg{} }
			case "e":
				m.isEditing = true
				m.textarea.SetValue(string(m.value)) // Assuming UTF-8 for edit
				m.textarea.Focus()
				return m, textarea.Blink
			case "d":
				// Delete confirmation?
				// For now, just delete
				return m, m.deleteKeyCmd()
			case "h":
				m.isHex = !m.isHex
				m.updateContent()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 4 // Title + Status
		footerHeight := 2 // Help
		verticalMarginHeight := headerHeight + footerHeight

		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - verticalMarginHeight - 2 // Border
		m.textarea.SetWidth(msg.Width - 4)
		m.textarea.SetHeight(msg.Height - verticalMarginHeight - 2)

	case ValueFetchedMsg:
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.value = msg.Value
			m.updateContent()
		}

	case OperationResultMsg:
		if msg.Err != nil {
			m.err = msg.Err
		} else {
			m.msg = msg.Message
			if msg.Op == "delete" {
				return m, func() tea.Msg { return BackToMainMsg{} }
			}
			if msg.Op == "save" {
				m.isEditing = false
				m.fetchValueCmd() // Reload
			}
		}
	}

	if m.isEditing {
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *DetailModel) updateContent() {
	content := ""
	if m.isHex {
		content = hex.Dump(m.value)
	} else {
		content = string(m.value)
	}
	m.viewport.SetContent(content)
}

func (m DetailModel) View() string {
	// Title
	title := m.styles.Title.Render(fmt.Sprintf("Key: %s", m.key))

	// Status Message
	status := ""
	if m.err != nil {
		status = m.styles.Error.Render(m.err.Error())
	} else if m.msg != "" {
		status = m.styles.Success.Render(m.msg)
	}

	// Content
	var content string
	if m.isEditing {
		content = m.textarea.View()
		content = m.styles.Focused.Render(content)
	} else {
		content = m.viewport.View()
		content = m.styles.Border.Render(content)
	}

	// Footer
	var help string
	if m.isEditing {
		help = m.styles.Help.Render("Ctrl+S: Save | Esc: Cancel")
	} else {
		help = m.styles.Help.Render("e: Edit | d: Delete | h: Toggle Hex | Esc: Back")
	}

	view := lipgloss.JoinVertical(lipgloss.Left,
		title,
		status,
		content,
		help,
	)

	return m.styles.Container.Render(view)
}

// Commands

type ValueFetchedMsg struct {
	Value []byte
	Err   error
}

func (m DetailModel) fetchValueCmd() tea.Cmd {
	return func() tea.Msg {
		val, err := m.dbClient.GetValue(m.key)
		return ValueFetchedMsg{Value: val, Err: err}
	}
}

type OperationResultMsg struct {
	Op      string
	Message string
	Err     error
}

func (m DetailModel) saveValueCmd() tea.Cmd {
	return func() tea.Msg {
		// Auto backup
		if m.cfg.DB.AutoBackupOnWrite {
			_, err := m.dbClient.BackupValue(m.key, m.cfg.DB.BackupPath)
			if err != nil {
				return OperationResultMsg{Op: "save", Err: fmt.Errorf("backup failed: %w", err)}
			}
		}

		// Save
		err := m.dbClient.SetValue(m.key, []byte(m.textarea.Value()), 0) // TTL 0 for now
		if err != nil {
			return OperationResultMsg{Op: "save", Err: err}
		}
		return OperationResultMsg{Op: "save", Message: locale.T("save_success")}
	}
}

func (m DetailModel) deleteKeyCmd() tea.Cmd {
	return func() tea.Msg {
		err := m.dbClient.DeleteKey(m.key)
		if err != nil {
			return OperationResultMsg{Op: "delete", Err: err}
		}
		return OperationResultMsg{Op: "delete", Message: locale.T("delete_success")}
	}
}

type BackToMainMsg struct{}
