package app

import (
	"archsetup/internal/assert"
	"archsetup/internal/dotfiles"
	"archsetup/internal/github"
	"archsetup/internal/layout"
	"archsetup/internal/menu"
	"archsetup/internal/navigator"
	"archsetup/internal/nvidia"
	"archsetup/internal/profiles"
	"archsetup/internal/styles"
	"archsetup/internal/types"
	"log"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	nav          navigator.Navigator[types.Phase]
	models       map[types.Phase]tea.Model
	height       int
	width        int
	keys         types.KeyMap
	dotfilesPath string
}

func New(
	initialPhase types.Phase,
	models map[types.Phase]tea.Model,
	keys types.KeyMap,
) *model {
	log.Printf("app: New received: %+v", initialPhase)

	return &model{
		keys:   keys,
		nav:    navigator.New(initialPhase),
		models: models,
	}
}

func (m *model) Init() tea.Cmd {
	log.Printf("app: Init received")

	nvidiaModel, ok := m.models[types.NvidiaDriversPhase].(*nvidia.Model)
	if !ok {
		assert.Fail("Nvidia model is not of the correct type.")
	}

	initialChecks := tea.Batch(
		github.CheckAuthCmd(),
		nvidiaModel.CheckGpuCmd(),
	)

	return tea.Batch(
		m.models[m.nav.Current()].Init(),
		initialChecks,
	)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case github.AuthStatusMsg:
		return m.handleGithubAuthStatusMsg(msg)

	case types.PhaseFinished:
		return m.handlePhaseFinished(msg)

	case types.PhaseCancelled:
		return m.handlePhaseCancelled()

	case dotfiles.DotfilesFinished:
		return m.handleDotFilesFinishedMsg(msg)

	case types.PhaseBack:
		return m.previousPhase()

	case types.MenuItemSelected:
		return m.handleMenuItemSelected(msg)

	default:
		return m.handleDefaultMsg(msg)
	}
}

func (m *model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	log.Printf("app: WindowSizeMsg received: %+v", msg)

	var cmd tea.Cmd
	var cmds []tea.Cmd

	m.width = msg.Width
	m.height = msg.Height

	childWidth := m.width - styles.AppStyle.GetHorizontalPadding()
	childHeight := m.height - styles.AppStyle.GetVerticalPadding()
	childMsg := tea.WindowSizeMsg{Width: childWidth, Height: childHeight}

	for phase, modelInstance := range m.models {
		m.models[phase], cmd = modelInstance.Update(childMsg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	log.Printf("app: KeyMsg received: %+v", msg)

	if key.Matches(msg, m.keys.HardQuit) {
		return m.handleQuit()
	}

	return m.delegateToActive(msg)
}

func (m *model) handleGithubAuthStatusMsg(
	msg github.AuthStatusMsg,
) (tea.Model, tea.Cmd) {
	log.Printf("app: AuthStatusMsg received: %+v", msg)

	var cmds []tea.Cmd

	m.updateAndCollectCmd(types.DotfilesPhase, msg, &cmds)

	var activeCmd tea.Cmd
	_, activeCmd = m.delegateToActive(msg)
	cmds = append(cmds, activeCmd)

	return m, tea.Batch(cmds...)
}

func (m *model) handlePhaseFinished(msg types.PhaseFinished) (tea.Model, tea.Cmd) {
	log.Println("app: PhaseFinishedMsg received")

	finished := m.nav.Current()

	var cmds []tea.Cmd

	m.updateAndCollectCmd(
		types.MenuPhase,
		menu.PhaseDoneMsg{Phase: finished},
		&cmds,
	)

	cmds = append(cmds, m.popNavAndInit())
	return m, tea.Batch(cmds...)
}

func (m *model) handlePhaseCancelled() (tea.Model, tea.Cmd) {
	log.Println("app: PhaseCancelledMsg received")

	return m, m.popNavAndInit()
}

func (m *model) handleDotFilesFinishedMsg(
	msg dotfiles.DotfilesFinished,
) (tea.Model, tea.Cmd) {
	log.Printf("app: DotfilesFinishedMsg received: %+v", msg)

	var cmd tea.Cmd
	var cmds []tea.Cmd

	m.dotfilesPath = msg.Path

	m.updateAndCollectCmd(
		types.ProfilesPhase,
		profiles.DotfilesPathUpdatedMsg{Path: msg.Path},
		&cmds,
	)

	m.updateAndCollectCmd(
		types.MenuPhase,
		menu.PhaseDoneMsg{Phase: types.DotfilesPhase},
		&cmds,
	)

	m.updateAndCollectCmd(
		types.MenuPhase,
		menu.DotfilesPathUpdatedMsg{Path: msg.Path},
		&cmds,
	)

	cmd = m.popNavAndInit()
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *model) popNavAndInit() tea.Cmd {
	log.Println("app: popNavigatorAndInit received")
	m.nav.Pop()
	return m.getActiveComponentModel().Init()
}

func (m *model) previousPhase() (tea.Model, tea.Cmd) {
	log.Println("app: PhaseBack received")

	if m.nav.Pop() {
		return m, nil
	}

	log.Println("app: no phase to go back to, quitting.")
	return m.handleQuit()
}

func (m *model) handleMenuItemSelected(
	msg types.MenuItemSelected,
) (tea.Model, tea.Cmd) {
	log.Printf("app: MenuItemSelected received: %+v", msg)

	m.nav.Push(msg.Phase)
	return m, m.getActiveComponentModel().Init()
}

func (m *model) handleDefaultMsg(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("app: tea.Msg received: %+v", msg)

	return m.delegateToActive(msg)
}

func (m *model) handleQuit() (tea.Model, tea.Cmd) {
	log.Println("app: quitting...")
	return m, tea.Quit
}

func (m *model) delegateToActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	updatedModel, cmd := m.getActiveComponentModel().Update(msg)
	m.setActiveComponentModel(updatedModel)
	return m, cmd
}

func (m *model) View() string {
	content := m.getActiveComponentModel().View()

	return layout.View(content, m.width, m.height)
}

func (m *model) setActiveComponentModel(updatedModel tea.Model) {
	m.models[m.nav.Current()] = updatedModel
}

func (m *model) getActiveComponentModel() tea.Model {
	model, found := m.models[m.nav.Current()]
	if !found {
		log.Fatalf("app: no model found for phase: %v", m.nav.Current())
	}

	return model
}

func (m *model) updateAndCollectCmd(
	phase types.Phase,
	msg tea.Msg,
	cmds *[]tea.Cmd,
) {
	updatedModel, cmd := m.models[phase].Update(msg)
	m.models[phase] = updatedModel
	*cmds = append(*cmds, cmd)
}
