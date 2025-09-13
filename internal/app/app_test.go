package app

import (
	"archsetup/internal/dotfiles"
	"archsetup/internal/github"
	"archsetup/internal/menu"
	"archsetup/internal/profiles"
	"archsetup/internal/types"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// mockModel satisfies the tea.Model interface and helps us track interactions.
type mockModel struct {
	lastMsgReceived tea.Msg
}

func (m *mockModel) Init() tea.Cmd {
	return nil
}

func (m *mockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.lastMsgReceived = msg
	return m, nil
}

func (m *mockModel) View() string {
	return "mock view"
}

// updateTrackingMock returns a new instance of itself on Update.
type updateTrackingMock struct {
	mockModel
}

// myTestCmd is a command we can check for in our tests.
func myTestCmd() tea.Msg { return nil }

func (m *updateTrackingMock) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.lastMsgReceived = msg
	// Return a new instance to prove the original was replaced
	return &updateTrackingMock{}, myTestCmd
}

// setupTestModel is a helper function to create a standard test setup.
func setupTestModel() (*model, map[types.Phase]tea.Model) {
	keys := types.DefaultKeys()
	mockModels := map[types.Phase]tea.Model{
		types.MenuPhase:          &mockModel{},
		types.GithubAuthPhase:    &mockModel{},
		types.DotfilesPhase:      &mockModel{},
		types.NvidiaDriversPhase: &mockModel{},
		types.ProfilesPhase:      &mockModel{},
	}
	appModel := New(types.MenuPhase, mockModels, keys)
	return appModel, mockModels
}

func TestAppModel_Navigation_OnMenuItemSelected(t *testing.T) {
	// ARRANGE
	m, _ := setupTestModel()
	selectDotfilesMsg := types.MenuItemSelected{Phase: types.DotfilesPhase}

	// ACT
	updatedModel, _ := m.Update(selectDotfilesMsg)
	m = updatedModel.(*model)

	// ASSERT
	expectedPhase := types.DotfilesPhase
	actualPhase := m.nav.Current()
	if actualPhase != expectedPhase {
		t.Errorf("navigation failed: expected phase %v, but got %v", expectedPhase, actualPhase)
	}
}

func TestAppModel_AuthStatusMsg_UpdatesCorrectModels(t *testing.T) {
	// ARRANGE
	keys := types.DefaultKeys()
	originalDotfilesModel := &updateTrackingMock{}
	activeMenuModel := &mockModel{}

	mockModels := map[types.Phase]tea.Model{
		types.MenuPhase:     activeMenuModel,
		types.DotfilesPhase: originalDotfilesModel,
	}

	m := New(types.MenuPhase, mockModels, keys)
	authMsg := github.AuthStatusMsg{IsAuthenticated: true}

	// ACT
	updatedModel, cmd := m.Update(authMsg)
	m = updatedModel.(*model)

	// ASSERT
	// 1. Check that the specific dotfiles model was updated and replaced.
	if m.models[types.DotfilesPhase] == originalDotfilesModel {
		t.Error("FAIL: The dotfiles model in the map was not updated with the new instance.")
	}

	// 2. Check that the active menu model also received the message.
	if activeMenuModel.lastMsgReceived != authMsg {
		t.Error("FAIL: The active menu model did not receive the auth status message.")
	}

	// 3. Check that a command was returned (from the combination of both updates).
	if cmd == nil {
		t.Error("FAIL: The command returned from the model updates was ignored.")
	}
}

func TestAppModel_PhaseBack_PopsNavigator(t *testing.T) {
	// ARRANGE
	m, _ := setupTestModel()
	m.nav.Push(types.DotfilesPhase) // Push a phase so we can pop it

	// ACT
	m.Update(types.PhaseBack{})

	// ASSERT
	if m.nav.Current() != types.MenuPhase {
		t.Error("expected to be back at MenuPhase")
	}
}

func TestAppModel_DotfilesFinished_UpdatesStateAndModels(t *testing.T) {
	// ARRANGE
	m, mockModels := setupTestModel()
	m.nav.Push(types.DotfilesPhase) // Start on a different phase
	finishMsg := dotfiles.DotfilesFinished{Path: "/test/path"}

	// ACT
	m.Update(finishMsg)

	// ASSERT
	// 1. Check that the internal state was updated.
	if m.dotfilesPath != "/test/path" {
		t.Errorf("expected dotfilesPath to be '/test/path', got %s", m.dotfilesPath)
	}

	// 2. Check that the profiles and menu models received the correct update message.
	profilesMock := mockModels[types.ProfilesPhase].(*mockModel)
	if _, ok := profilesMock.lastMsgReceived.(profiles.DotfilesPathUpdatedMsg); !ok {
		t.Error("profiles model did not receive DotfilesPathUpdatedMsg")
	}
	menuMock := mockModels[types.MenuPhase].(*mockModel)
	if _, ok := menuMock.lastMsgReceived.(menu.DotfilesPathUpdatedMsg); !ok {
		t.Error("menu model did not receive DotfilesPathUpdatedMsg")
	}

	// 3. Check that the navigator popped back to the menu.
	if m.nav.Current() != types.MenuPhase {
		t.Errorf("expected navigator to pop to MenuPhase, got %v", m.nav.Current())
	}
}

func TestAppModel_WindowSizeMsg_BroadcastsToAllModels(t *testing.T) {
	// ARRANGE
	m, mockModels := setupTestModel()
	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 50}

	// ACT
	m.Update(sizeMsg)

	// ASSERT
	for phase, modelInstance := range mockModels {
		mock, ok := modelInstance.(*mockModel)
		if !ok { // Skip the updateTrackingMock if it's in the map
			continue
		}
		if _, ok := mock.lastMsgReceived.(tea.WindowSizeMsg); !ok {
			t.Errorf("model for phase %v did not receive WindowSizeMsg", phase)
		}
	}
}

func TestAppModel_KeyMsg_HandlesQuit(t *testing.T) {
	// ARRANGE
	m, _ := setupTestModel()

	// Use the less ambiguous "ctrl+c" key for the test
	quitKeyMsg := tea.KeyMsg{Type: tea.KeyCtrlC}

	// ACT
	_, cmd := m.Update(quitKeyMsg)

	// ASSERT
	if cmd == nil {
		t.Fatal("Update did not return a command on quit key")
	}

	isQuitCmd := false
	if _, ok := cmd().(tea.QuitMsg); ok {
		isQuitCmd = true
	}

	if !isQuitCmd {
		t.Error("expected tea.Quit command but got something else")
	}
}
