package menu

import (
	"archsetup/internal/constants"
	"archsetup/internal/github"
	"archsetup/internal/nvidia"
	"archsetup/internal/styles"
	"archsetup/internal/types"
	"archsetup/internal/utils"
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	list   list.Model
	keys   types.KeyMap
	width  int
	height int
}

type DotfilesPathUpdatedMsg struct {
	Path string
}

type PhaseDoneMsg struct {
	Phase types.Phase
}

func New(keys types.KeyMap) tea.Model {
	items := GetMenuItems()
	delegate := utils.ItemDelegate{}
	menuList := list.New(items, delegate, 0, 0)

	menuList.Title = constants.Menu.Title
	menuStyle := styles.TitleStyle.Border(
		lipgloss.RoundedBorder(),
		false,
		false,
		true,
		false,
	).BorderForeground(lipgloss.Color("63"))

	menuList.Styles.Title = menuStyle
	menuList.SetShowHelp(true)
	menuList.SetShowStatusBar(false)
	menuList.SetShowPagination(false)
	menuList.SetFilteringEnabled(false)

	return &Model{
		keys: keys,
		list: menuList,
	}
}

func (m *Model) Init() tea.Cmd {
	return github.CheckAuthCmd()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("menu: Update received: %T: %+v", msg, msg)
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)

	case nvidia.GpuCheckResultMsg:
		return m.handleGpuCheckMsg(msg)

	case github.AuthStatusMsg:
		return m.handleGithubAuthMsg(msg)

	case DotfilesPathUpdatedMsg:
		return m.handleDotfilesPathMsg(msg)

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case PhaseDoneMsg:
		return m.handlePhaseDoneMsg(msg)
	}

	m.updateList(msg, &cmds)
	return m, tea.Batch(cmds...)
}

func (m *Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	return m, nil
}

func (m *Model) handleGpuCheckMsg(msg nvidia.GpuCheckResultMsg) (tea.Model, tea.Cmd) {
	log.Println("menu: received nvidia.GpuCheckResultMsg")
	items := m.list.Items()
	for i, li := range items {
		item, ok := li.(MenuItem)
		if !ok {
			continue
		}

		if item.Phase == types.NvidiaDriversPhase {
			if msg.HasNvidiaGpu {
				item.Enabled = true
				item.item.Description = constants.Menu.NvidiaMenuDesc
			} else {
				item.Enabled = false
				item.item.Description = constants.Menu.NvidiaDisDesc
			}
			items[i] = item
		}
	}
	cmd := m.list.SetItems(items)
	return m, cmd
}

func (m *Model) handleGithubAuthMsg(msg github.AuthStatusMsg) (tea.Model, tea.Cmd) {
	log.Println("menu: received github.AuthStatusMsg")
	items := m.list.Items()
	for i, li := range items {
		item, ok := li.(MenuItem)
		if !ok {
			continue
		}

		authenticated := constants.Menu.GithubAuthed
		unauthenticated := constants.Menu.GithubUnAuthed
		connected := constants.Menu.GithubConnected
		disconnected := constants.Menu.GithubDisconnected

		utils.PadRightToSameLength(
			&authenticated,
			&unauthenticated,
			&connected,
			&disconnected,
		)

		switch item.Phase {
		case types.GithubAuthPhase:
			if msg.IsAuthenticated {
				log.Printf("menu: github is authenticated with: %s", msg.Username)
				item.Done = true
				item.item.Description = fmt.Sprintf(GithubAuthDesc, authenticated)
			} else {
				log.Println("menu: github failed to authenticate")
				item.Done = false
				item.item.Description = fmt.Sprintf(GithubAuthDesc, unauthenticated)
			}

		case types.DotfilesPhase:
			item.Enabled = msg.IsAuthenticated

			if item.Enabled {
				item.item.Description = fmt.Sprintf(DotfilesDesc, connected)
			} else {
				item.item.Description = fmt.Sprintf(DotfilesDesc, disconnected)
			}
		}
		items[i] = item
	}

	cmd := m.list.SetItems(items)

	return m, cmd

}

func (m *Model) handleDotfilesPathMsg(
	msg DotfilesPathUpdatedMsg,
) (tea.Model, tea.Cmd) {
	log.Printf("menu: dotfiles finished with path: %+v", msg)
	items := m.list.Items()
	for i, li := range items {
		item, ok := li.(MenuItem)
		if !ok {
			continue
		}

		switch item.Phase {
		case types.ProfilesPhase:
			item.Enabled = len(msg.Path) > 0
		}
		items[i] = item
	}
	cmd := m.list.SetItems(items)
	return m, cmd
}

func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Enter) {
		selectedItem, ok := m.list.SelectedItem().(MenuItem)

		if !ok {
			return m, nil
		}

		if !selectedItem.Enabled {
			return m, nil
		}

		return m, func() tea.Msg {
			return types.MenuItemSelected{Phase: selectedItem.Phase}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *Model) updateList(msg tea.Msg, cmds *[]tea.Cmd) {
	updatedList, cmd := m.list.Update(msg)
	m.list = updatedList
	*cmds = append(*cmds, cmd)
}

func (m *Model) View() string {
	m.list.SetSize(m.width, utils.CalculateListHeight(m.list))

	return m.list.View()
}

func (m *Model) handlePhaseDoneMsg(msg PhaseDoneMsg) (tea.Model, tea.Cmd) {
	log.Printf("menu: handlePhaseDoneMsg: %+v", msg)

	items := m.list.Items()
	for i, li := range items {
		item, ok := li.(MenuItem)
		if !ok {
			continue
		}
		if item.Phase == msg.Phase {
			item.Done = true
			items[i] = item
			break
		}
	}
	cmd := m.list.SetItems(items)
	return m, cmd
}
