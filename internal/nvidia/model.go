package nvidia

import (
	"archsetup/internal/assert"
	"archsetup/internal/navigator"
	"archsetup/internal/styles"
	"archsetup/internal/types"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type phase int

const (
	confirmationPhase phase = iota
	installingPhase
	successPhase
	errorPhase
)

type Model struct {
	nav       navigator.Navigator[phase]
	keys      types.KeyMap
	spinner   spinner.Model
	selection bool
	width     int
	height    int
	service   *Service
	err       error
}

func New(keys types.KeyMap, service *Service) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	return &Model{
		nav:       navigator.New(confirmationPhase),
		keys:      keys,
		spinner:   s,
		service:   service,
		selection: true,
	}
}

func (m *Model) Init() tea.Cmd {
	m.nav.Reset(confirmationPhase)
	m.selection = true
	m.err = nil
	return nil
}

func (m *Model) CheckGpuCmd() tea.Cmd {
	return m.service.CheckGpuCmd()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)

	case InstallResultMsg:
		return m.handleInstallResultMsg(msg)

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	default:
		return m.handleDefault(msg)
	}
}

func (m *Model) handleWindowSizeMsg(
	msg tea.WindowSizeMsg,
) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	return m, nil
}

func (m *Model) handleInstallResultMsg(
	msg InstallResultMsg,
) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.err = msg.Err
		m.nav.Push(errorPhase)
	} else {
		m.nav.Push(successPhase)
	}
	return m, nil
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	currentPhase := m.nav.Current()

	// Handle returning to the menu from final states
	if currentPhase == successPhase || currentPhase == errorPhase {
		if key.Matches(msg, m.keys.Enter, m.keys.Back) {
			return m, func() tea.Msg { return types.PhaseFinished{} }
		}
	}

	if currentPhase == confirmationPhase {
		switch {
		case key.Matches(msg, m.keys.Back):
			return m, func() tea.Msg { return types.PhaseCancelled{} }

		case key.Matches(msg, m.keys.Up), key.Matches(msg, m.keys.Down):
			m.selection = !m.selection
			return m, nil

		case key.Matches(msg, m.keys.Enter):
			if m.selection {
				m.nav.Push(installingPhase)
				return m, tea.Batch(m.spinner.Tick, m.service.InstallDriversCmd())
			}
			return m, func() tea.Msg { return types.PhaseCancelled{} }
		}
	}

	return m.handleDefault(msg)
}

func (m *Model) handleDefault(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.nav.Current() == installingPhase {
		m.spinner, cmd = m.spinner.Update(msg)
	}

	return m, cmd
}

func (m *Model) View() string {
	switch m.nav.Current() {
	case confirmationPhase:
		return m.viewConfirmation()

	case installingPhase:
		return m.viewInstalling()

	case successPhase:
		return m.viewSuccess()

	case errorPhase:
		return m.viewError()

	default:
		assert.Fail(fmt.Sprintf("unknown phase: %v", m.nav.Current()))
		return ""
	}
}

func (m *Model) viewConfirmation() string {
	question := "We detected an NVIDIA GPU.\n"
	question += "\nDo you want to install the proprietary drivers?\n"
	question += styles.SubtleTextStyle.Render(
		"(This will install nvidia-dkms, nvidia-utils, and lib32-nvidia-utils)",
	)

	yes := "[ ] Yes"
	no := "[ ] No"
	if m.selection {
		yes = styles.TitleStyle.Render("[•] Yes")
	} else {
		no = styles.TitleStyle.Render("[•] No")
	}

	options := lipgloss.JoinVertical(lipgloss.Top, "   ", yes, "   ", no)
	help := styles.SubtleTextStyle.Render(
		"\nUse ↑/↓ to select. Press Enter to confirm, Esc to go back.",
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		question,
		"\n",
		options,
		"\n",
		help,
	)
}

func (m *Model) viewInstalling() string {
	return fmt.Sprintf("%s Installing NVIDIA drivers...", m.spinner.View())
}

func (m *Model) viewSuccess() string {
	msg := styles.SuccessStyle.Render("✅ Drivers installed successfully!")
	reboot := "A reboot is required for the changes to take effect."
	help := styles.SubtleTextStyle.Render(
		"\nPress Enter to return to the main menu.",
	)
	return lipgloss.JoinVertical(
		lipgloss.Center,
		msg,
		"\n",
		reboot,
		"\n",
		help,
	)
}

func (m *Model) viewError() string {
	msg := styles.ErrorStyle.Render("❌ Installation Failed")
	errText := fmt.Sprintf("Error: %v", m.err)
	help := styles.SubtleTextStyle.Render(
		"\nPress Enter to return to the main menu.",
	)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		msg,
		"\n",
		errText,
		"\n",
		help,
	)
}
