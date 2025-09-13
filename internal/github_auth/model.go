package github_auth

import (
	"archsetup/internal/navigator"
	"archsetup/internal/styles"
	"archsetup/internal/types"
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/skip2/go-qrcode"
)

type phase int

const (
	checkingKey phase = iota
	generatingKey
	displayingKey
	verifyingConnection
	authComplete
	finalSuccessPhase
	authError
)

type Model struct {
	nav       navigator.Navigator[phase]
	keys      types.KeyMap
	spinner   spinner.Model
	viewport  viewport.Model
	publicKey string
	username  string
	width     int
	height    int
	err       error
	service   *Service
}

type keyCheckResultMsg struct {
	keyExists       bool
	isAuthenticated bool
	publicKey       string
	username        string
	err             error
}

type keyGeneratedMsg struct {
	publicKey string
}

type errMsg struct {
	err error
}

func New(keys types.KeyMap, service *Service) tea.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	return &Model{
		keys:     keys,
		nav:      navigator.New(checkingKey),
		spinner:  s,
		viewport: viewport.New(0, 0),
		service:  service,
	}
}

func (m *Model) Init() tea.Cmd {
	m.nav.Reset(checkingKey)
	m.err = nil
	m.username = ""
	m.publicKey = ""

	return tea.Batch(m.spinner.Tick, m.service.CheckKeyCmd())
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	case keyCheckResultMsg:
		return m.handleKeyCheckResultMsg(msg)
	case keyGeneratedMsg:
		return m.handleKeyGeneratedMsg(msg)
	case errMsg:
		return m.handleErrMsg(msg)
	case verificationSuccessMsg:
		return m.handleVerificationSuccessMsg(msg)
	case verificationFailedMsg:
		return m.handleVerificationFailedMsg(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	// Update components that run on tick, like the spinner.
	switch m.nav.Current() {
	case checkingKey, generatingKey, verifyingConnection:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.viewport.Width = m.width
	m.viewport.Height = m.height - 1 // Account for help text line
	return m, nil
}

func (m *Model) handleKeyCheckResultMsg(msg keyCheckResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		m.nav.Push(authError)
		return m, nil
	}

	if !msg.keyExists {
		log.Printf("github_auth: [checkingKey] No key found.")
		m.nav.Push(generatingKey)
		return m, m.service.GenerateKeyCmd()
	}

	if !msg.isAuthenticated {
		log.Printf("github_auth: [checkingKey] Key found, but not authenticated.")
		m.publicKey = msg.publicKey
		m.nav.Push(displayingKey)
		return m, nil
	}

	log.Printf("github_auth: [checkingKey] Already authenticated as '%s'.", msg.username)
	m.username = msg.username
	m.publicKey = msg.publicKey
	m.nav.Push(authComplete)
	return m, nil
}

func (m *Model) handleKeyGeneratedMsg(msg keyGeneratedMsg) (tea.Model, tea.Cmd) {
	log.Printf("github_auth: [generatingKey] Key generation complete.")
	m.publicKey = msg.publicKey
	m.nav.Push(displayingKey)
	return m, nil
}

func (m *Model) handleErrMsg(msg errMsg) (tea.Model, tea.Cmd) {
	log.Printf("github_auth: Received error: %v", msg.err)
	m.err = msg.err
	m.nav.Push(authError)
	return m, nil
}

func (m *Model) handleVerificationSuccessMsg(msg verificationSuccessMsg) (tea.Model, tea.Cmd) {
	log.Printf("github_auth: [verifyingConnection] Success! Authenticated as '%s'.", msg.username)
	m.username = msg.username
	m.nav.Push(finalSuccessPhase)
	return m, nil
}

func (m *Model) handleVerificationFailedMsg(msg verificationFailedMsg) (tea.Model, tea.Cmd) {
	log.Printf("github_auth: [verifyingConnection] Failure: %v", msg.err)
	m.err = msg.err
	m.nav.Push(authError)
	return m, nil
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.nav.Current() {
	case displayingKey, authError, authComplete:
		if key.Matches(msg, m.keys.Enter) {
			m.err = nil
			m.nav.Push(verifyingConnection)
			return m, tea.Batch(m.spinner.Tick, m.service.VerifyConnectionCmd())
		}
		if key.Matches(msg, m.keys.Back) {
			return m, func() tea.Msg { return types.PhaseCancelled{} }
		}

	case finalSuccessPhase:
		if key.Matches(msg, m.keys.Enter) {
			return m, func() tea.Msg { return types.PhaseFinished{} }
		}
	}
	return m, nil
}

func (m *Model) viewKeyAndQR(header, instructions string) string {
	contentWidth := m.width
	const maxWidth = 160
	if contentWidth > maxWidth {
		contentWidth = maxWidth
	}

	keyBoxStyle := styles.BlurredBorderStyle.Padding(0, 1)
	keyTextWidth := contentWidth - keyBoxStyle.GetHorizontalPadding() - keyBoxStyle.GetHorizontalBorderSize()
	wrappedKeyText := lipgloss.NewStyle().Width(keyTextWidth).Render(m.publicKey)
	keyBox := keyBoxStyle.Render(wrappedKeyText)

	qr, err := qrcode.New(m.publicKey, qrcode.Medium)
	if err != nil {
		return fmt.Sprintf("Error generating QR code: %v", err)
	}
	qrCodeString := qr.ToSmallString(true)
	centeredQrCode := lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, qrCodeString)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		styles.SubtleTextStyle.Render("https://github.com/settings/keys"),
		"",
		keyBox,
		"",
		centeredQrCode,
		"",
		styles.SubtleTextStyle.Render(instructions),
	)
}

func (m *Model) View() string {
	if m.width == 0 {
		return ""
	}

	var finalContent string
	switch m.nav.Current() {
	case checkingKey, generatingKey, verifyingConnection:
		text := map[phase]string{
			checkingKey:         "Checking for existing SSH key...",
			generatingKey:       "No key found, generating a new one...",
			verifyingConnection: "Verifying connection to GitHub...",
		}[m.nav.Current()]
		finalContent = m.spinner.View() + " " + text

	case displayingKey, authError, authComplete:
		var header, instructions string
		switch m.nav.Current() {
		case displayingKey:
			header = "Please add this public SSH key to your GitHub account:"
			instructions = "Press Enter when you're done."
		case authError:
			errorMsg := styles.ErrorStyle.Render("Verification Failed: " + m.err.Error())
			header = errorMsg + "\n\nPlease add this public SSH key to your GitHub account:"
			instructions = "Press Enter to retry, or Esc to go back."
		case authComplete:
			header = styles.SuccessStyle.Render(fmt.Sprintf("✅ Already authenticated as %s", m.username))
			instructions = "Press Enter to re-validate, or Esc to return to the menu."
		}
		finalContent = m.viewKeyAndQR(header, instructions)

	case finalSuccessPhase:
		successMsg := styles.SuccessStyle.Render(fmt.Sprintf("✅ Successfully authenticated as %s!", m.username))
		instructions := styles.SubtleTextStyle.Render("Press Enter to return to the menu.")
		finalContent = lipgloss.JoinVertical(lipgloss.Center, successMsg, "\n", instructions)
	}

	// If content is too tall for the window, make it scrollable.
	if lipgloss.Height(finalContent) > m.height {
		m.viewport.SetContent(finalContent)
		helpView := styles.SubtleTextStyle.Render("↓/↑ / j/k to scroll")
		return lipgloss.JoinVertical(lipgloss.Left, m.viewport.View(), helpView)
	}

	return finalContent
}
