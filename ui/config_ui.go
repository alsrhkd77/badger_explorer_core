package ui

import (
	"strconv"
	"strings"

	"badger_explorer_core/config"
	"badger_explorer_core/locale"
	"badger_explorer_core/pkg"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type ConfigModel struct {
	cfg    *config.Config
	styles pkg.Styles

	inputs []textinput.Model
	cursor int

	err error
	msg string
}

func NewConfigModel(cfg *config.Config) ConfigModel {
	inputs := make([]textinput.Model, 5)

	inputs[0] = textinput.New()
	inputs[0].Placeholder = "Theme (dark/light)"
	inputs[0].SetValue(cfg.Theme)
	inputs[0].Focus()
	inputs[0].Prompt = "Theme: "

	inputs[1] = textinput.New()
	inputs[1].Placeholder = "Preview Chars"
	inputs[1].SetValue(strconv.Itoa(cfg.UI.PreviewChars))
	inputs[1].Prompt = "Preview Chars: "

	inputs[2] = textinput.New()
	inputs[2].Placeholder = "Page Size"
	inputs[2].SetValue(strconv.Itoa(cfg.UI.ValuePageSize))
	inputs[2].Prompt = "Page Size: "

	inputs[3] = textinput.New()
	inputs[3].Placeholder = "Auto Backup (true/false)"
	inputs[3].SetValue(strconv.FormatBool(cfg.DB.AutoBackupOnWrite))
	inputs[3].Prompt = "Auto Backup: "

	inputs[4] = textinput.New()
	inputs[4].Placeholder = "Backup Path"
	inputs[4].SetValue(cfg.DB.BackupPath)
	inputs[4].Prompt = "Backup Path: "

	return ConfigModel{
		cfg:    cfg,
		styles: pkg.DefaultStyles(),
		inputs: inputs,
		cursor: 0,
	}
}

func (m ConfigModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return BackToWelcomeMsg{} }
		case "tab", "down":
			m.inputs[m.cursor].Blur()
			m.cursor = (m.cursor + 1) % len(m.inputs)
			m.inputs[m.cursor].Focus()
		case "shift+tab", "up":
			m.inputs[m.cursor].Blur()
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(m.inputs) - 1
			}
			m.inputs[m.cursor].Focus()
		case "enter":
			// Save
			return m, m.saveCmd()
		}
	}

	m.inputs[m.cursor], cmd = m.inputs[m.cursor].Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m ConfigModel) View() string {
	s := strings.Builder{}

	s.WriteString(m.styles.Title.Render(locale.T("config")) + "\n\n")

	if m.err != nil {
		s.WriteString(m.styles.Error.Render(m.err.Error()) + "\n")
	}
	if m.msg != "" {
		s.WriteString(m.styles.Success.Render(m.msg) + "\n")
	}

	for i := range m.inputs {
		s.WriteString(m.inputs[i].View() + "\n")
	}

	s.WriteString("\n" + m.styles.Help.Render("Enter: Save | Tab/Arrows: Navigate | Esc: Back"))

	return s.String()
}

func (m *ConfigModel) saveCmd() tea.Cmd {
	return func() tea.Msg {
		// Parse inputs
		m.cfg.Theme = m.inputs[0].Value()

		pc, err := strconv.Atoi(m.inputs[1].Value())
		if err == nil {
			m.cfg.UI.PreviewChars = pc
		}

		ps, err := strconv.Atoi(m.inputs[2].Value())
		if err == nil {
			m.cfg.UI.ValuePageSize = ps
		}

		ab, err := strconv.ParseBool(m.inputs[3].Value())
		if err == nil {
			m.cfg.DB.AutoBackupOnWrite = ab
		}

		m.cfg.DB.BackupPath = m.inputs[4].Value()

		// Save to file
		if err := m.cfg.Save(); err != nil {
			return OperationResultMsg{Op: "config", Err: err}
		}

		return OperationResultMsg{Op: "config", Message: "Configuration saved"}
	}
}
