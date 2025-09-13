package profiles

import (
	"archsetup/internal/navigator"
	"archsetup/internal/styles"
	"archsetup/internal/system"
	"archsetup/internal/types"
	"archsetup/internal/utils"
	"bufio"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type phase int

const (
	checkingConfigurationPhase phase = iota
	selectOptionPhase
	loadingPackagesPhase
	confirmationPhase
	checkingYayPhase
	installingYayPhase
	installingPackagesPhase
	postInstallConfirmationPhase
	postInstallRunningPhase
	installCompletePhase
	errorPhase
)

type DotfilesPathUpdatedMsg struct {
	Path string
}

type tickMsg struct{}

type Model struct {
	dotfilesPath        string
	list                list.Model
	viewport            viewport.Model
	selectedProfile     profileItem
	packagesToInstall   []string
	packagesSucceeded   []string
	packagesFailed      []string
	currentPackageIndex int
	nav                 navigator.Navigator[phase]
	keys                types.KeyMap
	spinner             spinner.Model
	width               int
	height              int
	service             *Service
	err                 error

	// install process state
	execCmd *exec.Cmd
	logChan chan string
	logBuf  strings.Builder
}

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 2 }
func (d itemDelegate) Spacing() int                              { return 1 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(profileItem)
	if !ok {
		return
	}

	var title, desc string
	if index == m.Index() {
		title = styles.TitleStyle.Render("» " + i.Title())
		desc = styles.SubtleTextStyle.Render("  " + i.Description())
	} else {
		title = styles.NormalTextStyle.Render("  " + i.Title())
		desc = styles.SubtleTextStyle.Render("  " + i.Description())
	}
	fmt.Fprintf(w, "%s\n%s", title, desc)
}

func New(keys types.KeyMap, service *Service) tea.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.SpinnerStyle

	// Set up the list
	delegate := itemDelegate{}
	profileList := list.New([]list.Item{}, delegate, 0, 0)
	profileList.Title = "Select a Machine Profile"
	profileList.Styles.Title = styles.TitleStyle.Border(
		lipgloss.NormalBorder(),
		false,
		false,
		true,
		false,
	)
	profileList.SetShowHelp(true)
	profileList.SetShowStatusBar(false)
	profileList.SetShowPagination(false)
	profileList.SetFilteringEnabled(false)

	vp := viewport.New(0, 0)

	return &Model{
		keys:     keys,
		nav:      navigator.New(checkingConfigurationPhase),
		spinner:  s,
		list:     profileList,
		viewport: vp,
		service:  service,
	}
}

func (m *Model) readLogCmd() tea.Cmd {
	return func() tea.Msg {
		log.Printf("readLogCmd waiting…")
		if line, ok := <-m.logChan; ok {
			return installLogMsg{line: line}
		}
		if m.execCmd != nil {
			return installationFinishedMsg{err: m.execCmd.Wait()}
		}
		return installationFinishedMsg{err: nil}
	}
}

func (m *Model) Init() tea.Cmd {
	m.nav.Reset(checkingConfigurationPhase)
	m.err = nil

	return tea.Batch(
		m.spinner.Tick,
		m.service.getProfilesCmd(m.dotfilesPath),
	)
}

// streamCmdOutput is a tea.Cmd that starts a command, streams its output,
// and sends a final message when it's done.
func (m *Model) streamCmdOutput(
	cmd *exec.Cmd,
	stdout, stderr io.ReadCloser,
) tea.Cmd {
	return func() tea.Msg {
		// We need to store the command on the model so we can Wait() for it later
		m.execCmd = cmd
		m.logChan = make(chan string, 128)

		if err := cmd.Start(); err != nil {
			return installationFinishedMsg{err: fmt.Errorf("failed to start cmd: %w", err)}
		}

		// Signal that the process has started
		// We send a separate message so we can start listening for log messages
		m.logChan <- "Process started..."

		var wg sync.WaitGroup
		wg.Add(2)

		// Goroutine to stream stdout and stderr to the log channel
		stream := func(r io.Reader) {
			defer wg.Done()
			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				m.logChan <- scanner.Text()
			}
		}

		go stream(stdout)
		go stream(stderr)

		// Goroutine to wait for streaming to finish and then close the channel
		go func() {
			wg.Wait()
			close(m.logChan)
		}()

		return installStartedMsg{}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	// Messages with global effects or don't depend on current phase.
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	case DotfilesPathUpdatedMsg:
		return m.handleDotfilesPathUpdatedMsg(msg)
	case errMsg:
		return m.handleErrMsg(msg)
	case yayCheckResultMsg:
		return m.handleYayCheckResult(msg)
	case yayInstallResultMsg:
		return m.handleYayInstallResult(msg)

	// Messages that trigger the installation process.
	case startStreamingCmdMsg:
		return m.handleStartStreamingMsg(msg)
	case installStartedMsg:
		return m.handleInstallStartedMsg(msg)
	case installLogMsg:
		return m.handleInstallLogMsg(msg)
	case installationFinishedMsg:
		return m.handleInstallationFinishedMsg(msg)
	case packageInstallResultMsg:
		return m.handlePackageInstallResult(msg)
	case postInstallLogMsg:
		return m.handlePostInstallLogMsg(msg)
	case postInstallCompleteMsg:
		return m.handlePostInstallFinishedMsg(msg)

	// Messages that load data and change phase.
	case profilesLoadedMsg:
		return m.handleProfilesLoadedMsg(msg)
	case profilesNotFoundMsg:
		return m.handleProfilesNotFoundMsg(msg)
	case packagesLoadedMsg:
		return m.handlePackagesLoadedMsg(msg)
	case stowResultMsg:
		return m.handleStowResultMsg(msg)

	// Keyboard input, dependent on the current phase.
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	// For other messages (like spinner ticks), update the relevant component.
	switch m.nav.Current() {
	case checkingConfigurationPhase, loadingPackagesPhase:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	case selectOptionPhase:
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	case confirmationPhase, installingPackagesPhase:
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleWindowSizeMsg(
	msg tea.WindowSizeMsg,
) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.viewport.Width = msg.Width
	m.viewport.Height = msg.Height - 8
	return m, nil
}

func (m *Model) handleStartStreamingMsg(
	msg startStreamingCmdMsg,
) (tea.Model, tea.Cmd) {
	return m, m.streamCmdOutput(msg.cmd, msg.stdout, msg.stderr)
}

func (m *Model) handleStowResultMsg(msg stowResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		log.Printf("stow command failed: %v", msg.err)
		m.err = msg.err
		m.nav.Push(errorPhase)
		return m, nil
	}

	log.Println("stow command finished successfully")
	return m, nil
}

func (m *Model) handleDotfilesPathUpdatedMsg(
	msg DotfilesPathUpdatedMsg,
) (tea.Model, tea.Cmd) {
	log.Printf("dotfiles path updated: %s", msg.Path)
	m.dotfilesPath = msg.Path

	return m, nil
}

func (m *Model) handleErrMsg(msg errMsg) (tea.Model, tea.Cmd) {
	log.Printf("profiles.model received error: %v", msg.err)
	m.err = msg.err
	m.nav.Push(errorPhase)

	return m, nil
}

func (m *Model) handlePostInstallLogMsg(
	msg postInstallLogMsg,
) (tea.Model, tea.Cmd) {
	m.logBuf.WriteString(msg.line + "\n")
	m.viewport.SetContent(m.logBuf.String())
	m.viewport.GotoBottom()
	return m, m.readLogCmd()
}

func (m *Model) handlePostInstallFinishedMsg(
	msg postInstallCompleteMsg,
) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		log.Printf("profiles: post-install script failed: %v", msg.err)
		m.err = msg.err
		m.nav.Push(errorPhase)
		return m, nil
	}

	log.Printf("profiles: post-install script succeeded")
	m.nav.Push(installCompletePhase)
	return m, nil
}

func (m *Model) handleInstallStartedMsg(
	msg installStartedMsg,
) (tea.Model, tea.Cmd) {
	return m, m.readLogCmd()
}

func (m *Model) handleInstallLogMsg(msg installLogMsg) (tea.Model, tea.Cmd) {
	log.Printf("installLogMsg received: %s", msg.line)
	m.logBuf.WriteString(msg.line)
	m.logBuf.WriteByte('\n')
	m.viewport.SetContent(m.logBuf.String())
	m.viewport.GotoBottom()
	return m, m.readLogCmd()
}

func (m *Model) handleInstallationFinishedMsg(
	msg installationFinishedMsg,
) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		log.Printf("install finished with error: %v", msg.err)
		m.err = msg.err
		m.nav.Push(errorPhase)
		return m, nil
	}

	log.Println("install finished successfully")
	m.nav.Push(installCompletePhase)
	return m, nil
}

func (m *Model) handlePackageInstallResult(
	msg packageInstallResultMsg,
) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		log.Printf("Failed to install package %s: %v", msg.pkg, msg.err)
		m.packagesFailed = append(m.packagesFailed, msg.pkg)
		m.logBuf.WriteString(fmt.Sprintf("\n❌ Failed to install %s\n", msg.pkg))
	} else {
		log.Printf("Successfully installed package %s", msg.pkg)
		m.packagesSucceeded = append(m.packagesSucceeded, msg.pkg)
		m.logBuf.WriteString(fmt.Sprintf("\n✓ Successfully installed %s\n", msg.pkg))
	}

	m.currentPackageIndex++

	// If there are more packages, install the next one.
	if m.currentPackageIndex < len(m.packagesToInstall) {
		nextPackage := m.packagesToInstall[m.currentPackageIndex]
		return m, m.service.installPackageCmd(nextPackage)
	}

	log.Println("All packages processed.")

	if m.selectedProfile.PostInstall == nil {
		m.nav.Push(installCompletePhase)
		return m, nil
	}

	m.nav.Push(postInstallConfirmationPhase)
	m.logBuf.Reset()
	m.viewport.SetContent("")
	return m, nil
}

func (m *Model) handleProfilesLoadedMsg(
	msg profilesLoadedMsg,
) (tea.Model, tea.Cmd) {
	log.Printf("Custom profiles loaded: %d found", len(msg.Config.Profiles))

	info := system.CurrentOSInfo()
	var items []list.Item

	for _, p := range msg.Config.Profiles {
		famOk := p.OsFamily == "" || p.OsFamily == info.Family
		if !famOk {
			continue
		}

		distOk := p.OsDistro == "" || p.OsDistro == info.Distro
		if !distOk {
			continue
		}

		items = append(items, profileItem{Profile: p})
	}

	m.list.SetItems(items)
	m.nav.Reset(selectOptionPhase)

	return m, nil
}

func (m *Model) handleProfilesNotFoundMsg(
	msg profilesNotFoundMsg,
) (tea.Model, tea.Cmd) {
	log.Println("No custom profiles file found. Using defaults.")
	profiles := m.service.getDefaultProfiles()
	items := make([]list.Item, len(profiles))
	for i, p := range profiles {
		items[i] = profileItem{Profile: p}
	}
	m.list.SetItems(items)
	m.nav.Reset(selectOptionPhase)

	return m, nil
}

func (m *Model) handlePackagesLoadedMsg(
	msg packagesLoadedMsg,
) (tea.Model, tea.Cmd) {
	m.packagesToInstall = msg.packages
	content := "The following packages will be installed:\n\n" +
		strings.Join(m.packagesToInstall, "\n")
	m.viewport.SetContent(content)
	m.nav.Push(confirmationPhase)

	return m, nil
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.nav.Current() {
	case selectOptionPhase:
		return m.handleSelectOptionKeys(msg)
	case confirmationPhase:
		return m.handleConfirmationKeys(msg)
	case postInstallConfirmationPhase:
		return m.handlePostInstallConfirmationKeys(msg)
	case installCompletePhase, errorPhase:
		return m.handleFinalPhaseKeys(msg)
	}
	return m, nil
}

func (m *Model) handleSelectOptionKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.list.Items()) == 0 {
		if key.Matches(msg, m.keys.Enter) || key.Matches(msg, m.keys.Back) {
			return m, func() tea.Msg {
				return types.PhaseCancelled{}
			}
		}

		return m, nil
	}

	if key.Matches(msg, m.keys.Enter) {
		selectedProfile, ok := m.list.SelectedItem().(profileItem)
		if !ok {
			return m, nil
		}
		m.selectedProfile = selectedProfile
		m.nav.Push(loadingPackagesPhase)
		return m, tea.Batch(
			m.spinner.Tick,
			m.service.loadPackagesCmd(m.dotfilesPath, m.selectedProfile.Path),
		)
	}

	// Delegate other keys to the list component
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) handleConfirmationKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Enter):
		log.Printf(
			"Confirmed installation for profile: %s",
			m.selectedProfile.Name,
		)
		// Instead of starting the install directly, check for yay first.
		m.nav.Push(checkingYayPhase)
		return m, m.service.CheckPkgMgrCmd()

	case key.Matches(msg, m.keys.Back):
		m.nav.Reset(selectOptionPhase)
		return m, nil
	}

	// Delegate other keys to the viewport for scrolling
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *Model) handlePostInstallConfirmationKeys(
	msg tea.KeyMsg,
) (tea.Model, tea.Cmd) {
	log.Println("profiles: handlePostInstallConfirmationKeys started")
	switch {
	case key.Matches(msg, m.keys.Enter):
		log.Println("profiles: handlePostInstallConfirmationKeys: Accept")

		m.nav.Push(postInstallRunningPhase)

		roles := make([]string, 0, len(m.selectedProfile.Roles))
		for _, r := range m.selectedProfile.Roles {
			r = strings.TrimSpace(r)
			if r != "" {
				roles = append(roles, r)
			}
		}
		log.Printf("profiles: handlePostInstallConfirmationKeys: roles: %v", roles)
		env := map[string]string{
			"MACHINE_PROFILES": strings.Join(roles, ","),
		}

		return m, m.service.RunPostInstallCmd(
			m.dotfilesPath,
			*m.selectedProfile.PostInstall,
			env,
		)

	case key.Matches(msg, m.keys.Back):
		log.Println("profiles: handlePostInstallConfirmationKeys: Decline")

		// User chose to skip, go to the final screen.
		m.nav.Push(installCompletePhase)
		return m, nil
	}
	return m, nil
}

func (m *Model) handleFinalPhaseKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.nav.Current() {
	case installCompletePhase:
		if key.Matches(msg, m.keys.Enter) {
			return m, func() tea.Msg { return types.PhaseFinished{} }
		}
	case errorPhase:
		if key.Matches(msg, m.keys.Enter, m.keys.Back) {
			m.nav.Reset(selectOptionPhase)
		}
	}
	return m, nil
}

func (m *Model) handleYayCheckResult(msg yayCheckResultMsg) (tea.Model, tea.Cmd) {
	if msg.isInstalled {
		log.Println("profiles: yay is already installed.")
		// Yay exists, proceed directly to package installation
		return m.startPackageInstallation()
	}

	log.Println("profiles: yay not found, starting installation.")
	m.nav.Push(installingYayPhase)
	// Reset the log buffer for the yay installation view
	m.logBuf.Reset()
	m.viewport.SetContent("")
	return m, m.service.InstallPkgMgrCmd()
}

// Add this new handler function
func (m *Model) handleYayInstallResult(msg yayInstallResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.err = fmt.Errorf("failed to install yay: %w", msg.err)
		m.nav.Push(errorPhase)
		return m, nil
	}
	log.Println("profiles: yay installed successfully.")
	// Yay is now installed, proceed to package installation
	return m.startPackageInstallation()
}

// Create a new helper function to start the actual package installation.
// This avoids duplicating code.
func (m *Model) startPackageInstallation() (tea.Model, tea.Cmd) {
	m.nav.Reset(installingPackagesPhase) // Use Reset to clear nav history like "installing yay"
	m.currentPackageIndex = 0
	m.packagesSucceeded = nil
	m.packagesFailed = nil
	m.logBuf.Reset()

	// Stow dotfiles first
	stowCmd := m.service.stowCmd(m.dotfilesPath, m.selectedProfile.StowDirs)

	if len(m.packagesToInstall) == 0 {
		m.nav.Push(installCompletePhase)
		return m, stowCmd
	}

	firstPackage := m.packagesToInstall[0]
	return m, tea.Batch(
		stowCmd,
		m.service.installPackageCmd(firstPackage),
	)
}

func (m *Model) View() string {
	switch m.nav.Current() {
	case checkingConfigurationPhase:
		return m.spinner.View() + " Checking for custom profiles configuration..."

	case selectOptionPhase:
		if len(m.list.Items()) == 0 {
			instructions := "No profiles found\n"
			instructions += "" // TODO: Instructions for adding profiles
			instructions += "\nPress Enter or Escape to go back"
			return instructions
		}

		m.list.SetSize(m.width, utils.CalculateListHeight(m.list))
		return m.list.View()

	case loadingPackagesPhase:
		return m.spinner.View() + fmt.Sprintf(" Loading packages for %s...", m.selectedProfile.Name)

	case confirmationPhase:
		header := fmt.Sprintf("Ready to install profile '%s'?", m.selectedProfile.Name)
		help := styles.SubtleTextStyle.Render("Press Enter to confirm, Esc to go back, ↑/↓ to scroll.")

		return lipgloss.JoinVertical(lipgloss.Left,
			styles.TitleStyle.Render(header),
			styles.BlurredBorderStyle.Render(m.viewport.View()),
			help,
		)

	case checkingYayPhase:
		return m.spinner.View() + " Checking for AUR helper (yay)..."

	case installingYayPhase:
		header := styles.TitleStyle.Render("Installing AUR Helper (yay)")
		help := styles.SubtleTextStyle.Render("Please wait, this may take a while...")

		// The command output from InstallYayCmd will be in the viewport via tea.ExecProcess
		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			styles.BlurredBorderStyle.Render(m.viewport.View()),
			help,
		)

	case installingPackagesPhase:
		// Show which package is currently being installed and the progress.
		total := len(m.packagesToInstall)
		current := m.currentPackageIndex + 1
		if current > total {
			current = total
		}

		var pkg string
		if m.currentPackageIndex < total && total > 0 {
			pkg = m.packagesToInstall[m.currentPackageIndex]
		} else {
			pkg = "(finalizing)"
		}

		header := styles.TitleStyle.Render(
			fmt.Sprintf("Installing (%d/%d): %s", current, total, pkg),
		)
		help := styles.SubtleTextStyle.Render("Please wait, this may take a while...")

		// We can reuse the viewport to show the log of previous installs in the loop.
		m.viewport.SetContent(m.logBuf.String())
		m.viewport.GotoBottom()

		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			styles.BlurredBorderStyle.Render(m.viewport.View()),
			help,
		)

	case postInstallConfirmationPhase:
		desc := m.selectedProfile.PostInstall.Description
		header := styles.TitleStyle.Render("Run Post-Install Script?")
		help := styles.SubtleTextStyle.Render("Press Enter to run, Esc to skip.")

		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			"\n",
			desc,
			"\n\n",
			help,
		)

	case postInstallRunningPhase:
		header := styles.TitleStyle.Render(fmt.Sprintf(
			"Running: %s",
			m.selectedProfile.PostInstall.Command,
		))
		help := styles.SubtleTextStyle.Render("Please wait, this may take a while...")

		return lipgloss.JoinVertical(
			lipgloss.Left,
			header,
			styles.BlurredBorderStyle.Render(m.viewport.View()),
			help,
		)

	case installCompletePhase:
		var summary strings.Builder
		summary.WriteString(styles.SuccessStyle.Render("✅ Installation Complete!"))
		summary.WriteString(fmt.Sprintf("\n\nSucceeded: %d, Failed: %d\n", len(m.packagesSucceeded), len(m.packagesFailed)))

		if len(m.packagesFailed) > 0 {
			summary.WriteString("\nFailed packages:\n")
			for _, pkg := range m.packagesFailed {
				summary.WriteString(fmt.Sprintf("  - %s\n", pkg))
			}
		}

		summary.WriteString("\n" + styles.SubtleTextStyle.Render("Press Enter to return to the main menu."))
		return summary.String()

	case errorPhase:
		return styles.ErrorStyle.Width(m.width).Render(fmt.Sprintf("Error: %v", m.err))

	default:
		return "Unknown state."
	}
}
