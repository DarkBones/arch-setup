package menu

import (
	"archsetup/internal/github"
	"archsetup/internal/types"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func setupTestModel() *Model {
	return New(types.DefaultKeys()).(*Model)
}

func findMenuItem(m *Model, phase types.Phase) (MenuItem, bool) {
	for _, item := range m.list.Items() {
		menuItem, ok := item.(MenuItem)
		if ok && menuItem.Phase == phase {
			return menuItem, true
		}
	}
	return MenuItem{}, false
}

func TestUpdate_AuthStatusAuthenticated(t *testing.T) {
	// Arrange
	m := setupTestModel()
	authMsg := github.AuthStatusMsg{IsAuthenticated: true, Username: "testuser"}

	// Act
	updatedModel, _ := m.Update(authMsg)
	m = updatedModel.(*Model)

	// Assert
	var dotfilesItem MenuItem
	for _, item := range m.list.Items() {
		menuItem := item.(MenuItem)
		if menuItem.Phase == types.DotfilesPhase {
			dotfilesItem = menuItem
			break
		}
	}

	if !dotfilesItem.Enabled {
		t.Error("expected Dotfiles item to be enabled after authentication, but it was not")
	}
}

func TestUpdate_AuthStatus(t *testing.T) {
	testCases := []struct {
		name              string
		authMsg           github.AuthStatusMsg
		expectedEnabled   bool
		expectedAuthDesc  string
		expectedFilesDesc string
	}{
		{
			name:              "Authenticated",
			authMsg:           github.AuthStatusMsg{IsAuthenticated: true},
			expectedEnabled:   true,
			expectedAuthDesc:  "Set up SSH keys for GitHub. (Authenticated)",
			expectedFilesDesc: "Clone and set up your dotfiles. (Connected)",
		},
		{
			name:              "Not Authenticated",
			authMsg:           github.AuthStatusMsg{IsAuthenticated: false},
			expectedEnabled:   false,
			expectedAuthDesc:  "Set up SSH keys for GitHub. (Not Authenticated)",
			expectedFilesDesc: "Clone and set up your dotfiles. (Not connected)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			m := setupTestModel()

			// Act
			updatedModel, _ := m.Update(tc.authMsg)
			m = updatedModel.(*Model)

			// Assert
			dotfilesItem, _ := findMenuItem(m, types.DotfilesPhase)
			if dotfilesItem.Enabled != tc.expectedEnabled {
				t.Errorf("expected Dotfiles item enabled status to be %v, but got %v", tc.expectedEnabled, dotfilesItem.Enabled)
			}

			authItem, _ := findMenuItem(m, types.GithubAuthPhase)
			if strings.TrimSpace(authItem.Description()) != tc.expectedAuthDesc {
				t.Errorf(
					"expected Auth item description to be %q, but got %q",
					tc.expectedAuthDesc,
					authItem.Description(),
				)
			}
			if strings.TrimSpace(dotfilesItem.Description()) != tc.expectedFilesDesc {
				t.Errorf(
					"expected Dotfiles item description to be %q, but got %q",
					tc.expectedFilesDesc,
					dotfilesItem.Description(),
				)
			}
		})
	}
}

func TestUpdate_DotfilesPathUpdated(t *testing.T) {
	// Arrange
	m := setupTestModel()
	msg := DotfilesPathUpdatedMsg{Path: "/path/to/dotfiles"}

	// Act
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(*Model)

	// Assert
	profilesItem, _ := findMenuItem(m, types.ProfilesPhase)
	if !profilesItem.Enabled {
		t.Error("expected Profiles item to be enabled after dotfiles path was updated, but it was not")
	}
}

func TestUpdate_EnterKeySelection(t *testing.T) {
	testCases := []struct {
		name          string
		phaseToSelect types.Phase
		setupModel    func(m *Model)
		expectCommand bool
	}{
		{
			name:          "Selects enabled item",
			phaseToSelect: types.DotfilesPhase,
			setupModel: func(m *Model) {
				m.Update(github.AuthStatusMsg{IsAuthenticated: true})
			},
			expectCommand: true,
		},
		{
			name:          "Ignores disabled item",
			phaseToSelect: types.DotfilesPhase,
			setupModel:    func(m *Model) {},
			expectCommand: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			m := setupTestModel()
			tc.setupModel(m)

			var itemIndex int
			for i, item := range m.list.Items() {
				if item.(MenuItem).Phase == tc.phaseToSelect {
					itemIndex = i
					break
				}
			}
			m.list.Select(itemIndex)
			enterMsg := tea.KeyMsg{Type: tea.KeyEnter}

			// Act
			_, cmd := m.Update(enterMsg)

			// Assert
			if tc.expectCommand {
				if cmd == nil {
					t.Fatal("expected a command to be returned, but got nil")
				}
				msg := cmd()
				if selectedMsg, ok := msg.(types.MenuItemSelected); !ok {
					t.Errorf("expected msg of type types.MenuItemSelected, but got %T", msg)
				} else if selectedMsg.Phase != tc.phaseToSelect {
					t.Errorf("expected selected phase to be %v, but got %v", tc.phaseToSelect, selectedMsg.Phase)
				}
			} else {
				if cmd != nil {
					t.Fatal("expected a nil command, but a command was returned")
				}
			}
		})
	}
}
