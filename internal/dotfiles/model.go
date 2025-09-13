package dotfiles

import (
	"archsetup/internal/assert"
	"archsetup/internal/github"
	"archsetup/internal/navigator"
	"archsetup/internal/styles"
	"archsetup/internal/types"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type phase int

const (
	inputPhase phase = iota
	verifyingPhase
	dirExistsPhase
	confirmationPhase
	cloningPhase
	cloneCompletePhase
)

type DotfilesFinished struct {
	Path string
}

const maxInputWidth = 100

type Model struct {
	Username     string
	nav          navigator.Navigator[phase]
	keys         types.KeyMap
	spinner      spinner.Model
	repoInput    textinput.Model
	destInput    textinput.Model
	focusedInput int
	width        int
	height       int
	service      *Service
	err          error
}

func New(
	keys types.KeyMap,
	service *Service,
	defaultDest string,
) tea.Model {
	repo := textinput.New()
	repo.Placeholder = "username/dotfiles-repo"
	repo.Focus()
	repo.CharLimit = 100

	dest := textinput.New()
	dest.Placeholder = defaultDest
	dest.SetValue(defaultDest)
	dest.CharLimit = 200

	s := spinner.New()
	s.Spinner = spinner.Dot

	return &Model{
		keys:         types.InputNavKeys(keys),
		nav:          navigator.New(inputPhase),
		spinner:      s,
		repoInput:    repo,
		destInput:    dest,
		focusedInput: 0,
		service:      service,
	}
}

func (m *Model) Init() tea.Cmd {
	m.nav.Reset(inputPhase)

	return textinput.Blink
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if m.nav.Current() == verifyingPhase || m.nav.Current() == cloningPhase {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)

	case github.AuthStatusMsg:
		return m.handleGithubAuthStatusMsg(msg)

	case validationResultMsg:
		return m.handleValidationResultMsg(msg)

	case cloneResultMsg:
		return m.handleCloneResultMsg(msg)

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleWindowSizeMsg(
	msg tea.WindowSizeMsg,
) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.repoInput.Width = m.getInputWidth()
	m.destInput.Width = m.getInputWidth()
	return m, nil
}

func (m *Model) handleGithubAuthStatusMsg(
	msg github.AuthStatusMsg,
) (tea.Model, tea.Cmd) {
	m.Username = msg.Username
	m.repoInput.SetValue(fmt.Sprintf("%s/dotfiles", m.Username))
	return m, nil
}

func (m *Model) handleValidationResultMsg(
	msg validationResultMsg,
) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.nav.Pop()
		return m, nil
	}

	nextPhase := confirmationPhase
	if msg.DirAlreadyExists {
		nextPhase = dirExistsPhase
	}

	m.nav.Push(nextPhase)

	return m, nil
}

func (m *Model) handleCloneResultMsg(
	msg cloneResultMsg,
) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.nav.Reset(inputPhase)
		return m, nil
	}
	m.nav.Push(cloneCompletePhase)
	return m, nil
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.nav.Current() {
	case inputPhase:
		return m.handleInputKeys(msg)
	case confirmationPhase:
		return m.handleConfirmationKeys(msg)
	case cloneCompletePhase, dirExistsPhase:
		return m.handleCloneCompleteKeys(msg)
	case verifyingPhase, cloningPhase:
		return m, nil
	default:
		switch {
		case key.Matches(msg, m.keys.Back):
			return m.previousPhase()
		}

		return m, nil
	}

}

func (m *Model) handleInputKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Enter):
		repoPath := m.repoPath()
		destPath := m.destPath()

		if repoPath == "" || destPath == "" || strings.HasSuffix(repoPath, "/") {
			m.err = errors.New("paths cannot be empty or incomplete")
			return m, nil
		}
		m.err = nil
		m.nav.Push(verifyingPhase)
		return m, tea.Batch(
			m.spinner.Tick,
			m.service.ValidateCmd(repoPath, destPath),
		)

	case key.Matches(msg, m.keys.Back):
		return m.previousPhase()

	case key.Matches(msg, m.keys.Tab):
		m.focusedInput = (m.focusedInput + 1) % 2

	case key.Matches(msg, m.keys.ShiftTab):
		m.focusedInput = (m.focusedInput - 1 + 2) % 2

	case key.Matches(msg, m.keys.Up):
		if m.focusedInput > 0 {
			m.focusedInput--
		}

	case key.Matches(msg, m.keys.Down):
		if m.focusedInput < 1 {
			m.focusedInput++
		}

	default:
		var cmd tea.Cmd

		m.err = nil
		if m.focusedInput == 0 {
			m.repoInput, cmd = m.repoInput.Update(msg)
		} else {
			m.destInput, cmd = m.destInput.Update(msg)
		}

		cmds = append(cmds, cmd)
	}

	if m.focusedInput == 0 {
		m.repoInput.Focus()
		m.destInput.Blur()
	} else {
		m.repoInput.Blur()
		m.destInput.Focus()
	}

	cmds = append(cmds, textinput.Blink)

	return m, tea.Batch(cmds...)
}

func (m *Model) handleConfirmationKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.nav.Reset(inputPhase)
		return m, nil
	case key.Matches(msg, m.keys.Enter):
		m.nav.Push(cloningPhase)
		repoPath := strings.TrimSpace(m.repoInput.Value())
		destPath := strings.TrimSpace(m.destInput.Value())
		return m, tea.Batch(
			m.spinner.Tick,
			m.service.CloneRepoCmd(repoPath, destPath),
		)
	}

	return m, nil
}

func (m *Model) handleCloneCompleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Enter):
		return m, func() tea.Msg { return DotfilesFinished{Path: m.destPath()} }

	case key.Matches(msg, m.keys.Back):
		m.nav.Reset(inputPhase)
		return m, nil
	}

	return m, nil
}

func (m *Model) destPath() string {
	return strings.TrimSpace(m.destInput.Value())
}

func (m *Model) repoPath() string {
	return strings.TrimSpace(m.repoInput.Value())
}

func (m *Model) nextPhase() (tea.Model, tea.Cmd) {
	return m, func() tea.Msg { return types.PhaseFinished{} }
}

func (m *Model) previousPhase() (tea.Model, tea.Cmd) {
	if m.nav.Pop() {
		return m, nil
	}

	return m, func() tea.Msg { return types.PhaseBack{} }
}

func (m *Model) getInputWidth() int {
	w := m.width - styles.AppStyle.GetHorizontalPadding() - 4
	if w > maxInputWidth {
		return maxInputWidth
	}
	return w
}

func (m *Model) View() string {
	if m.width == 0 {
		return ""
	}

	switch m.nav.Current() {
	case inputPhase:
		return m.viewInput()

	case verifyingPhase:
		return m.viewVerifying()

	case confirmationPhase:
		return m.viewConfirmation()

	case dirExistsPhase:
		return m.viewDirExists()

	case cloningPhase:
		return m.viewCloning()

	case cloneCompletePhase:
		return m.viewCloneComplete()

	default:
		assert.Fail("Unknown phase in dotfiles model")
		return ""
	}
}

func (m *Model) viewInput() string {
	var repoBox, destBox string
	if m.focusedInput == 0 {
		repoBox = styles.FocusedBorderStyle.Render(m.repoInput.View())
		destBox = styles.BlurredBorderStyle.Render(m.destInput.View())
	} else {
		repoBox = styles.BlurredBorderStyle.Render(m.repoInput.View())
		destBox = styles.FocusedBorderStyle.Render(m.destInput.View())
	}

	var errorLine string
	if m.err != nil {
		errorLine = styles.ErrorStyle.Render(m.err.Error())
	}

	help := styles.SubtleTextStyle.Render(
		"Use Tab/Shift+Tab or ↑/↓ to switch. Press Enter to continue.",
	)

	return lipgloss.JoinVertical(lipgloss.Left,
		styles.TitleStyle.Render("Dotfiles Setup"),
		errorLine,
		"\nEnter the path to your dotfiles repository on GitHub.",
		styles.SubtleTextStyle.Render("(e.g., ansimb/dotfiles)"),
		repoBox,
		"\nWhere should the repository be cloned?",
		styles.SubtleTextStyle.Render("(e.g. /home/you/dotfiles)"),
		destBox,
		"\n",
		help,
	)
}

func (m *Model) viewVerifying() string {
	return m.spinner.View() + " Verifying repository and destination..."
}

func (m *Model) viewConfirmation() string {
	repo := styles.TitleStyle.Render(m.repoInput.Value())
	dest := styles.TitleStyle.Render(m.destInput.Value())
	return fmt.Sprintf("Ready to clone %s into %s?\n\n", repo, dest) +
		styles.SubtleTextStyle.Render(
			"Press Enter to confirm, or Esc to go back.",
		)
}

func (m *Model) viewDirExists() string {
	message := fmt.Sprintf(
		"✓ Dotfiles directory already found at %s.",
		styles.TitleStyle.Render(m.destPath()),
	)
	help := styles.SubtleTextStyle.Render(
		"\nPress Enter to continue, or Esc to go back.",
	)
	return lipgloss.JoinVertical(lipgloss.Left, message, "\n", help)
}

func (m *Model) viewCloning() string {
	return m.spinner.View() + " Cloning into " + m.destInput.Value() + "..."
}

func (m *Model) viewCloneComplete() string {
	message := "✓ Dotfiles cloned successfully!"
	help := styles.SubtleTextStyle.Render("\nPress Enter to finish.")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		message,
		"\n",
		help,
	)
}
