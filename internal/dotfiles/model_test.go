package dotfiles

import (
	"archsetup/internal/github"
	"archsetup/internal/types"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func setupTestModel() *Model {
	keys := types.DefaultKeys()
	mockExec := &mockExecutor{}
	mockFS := &mockFileSystem{}
	service := NewService(mockExec, mockFS)

	defaultPath := "/home/testuser/dotfiles"

	return New(keys, service, defaultPath).(*Model)
}

func TestUpdate_ValidationResultError(t *testing.T) {
	// Arrange
	m := setupTestModel()
	m.nav.Push(verifyingPhase)

	msg := validationResultMsg{err: errors.New("validation failed")}

	// Act
	updatedModel, cmd := m.Update(msg)
	m = updatedModel.(*Model)

	// Assert
	if m.err == nil {
		t.Error("expected model.err to be set, but it was nil")
	}
	if m.nav.Current() != inputPhase {
		t.Errorf("expected phase to be %v, but got %v", inputPhase, m.nav.Current())
	}
	if cmd != nil {
		t.Error("expected a nil command, but got one")
	}
}

func TestUpdate_EnterOnInputPhase_Submits(t *testing.T) {
	// Arrange
	m := setupTestModel()
	m.repoInput.SetValue("test/repo")

	enterKey := tea.KeyMsg{Type: tea.KeyEnter}

	// Act
	updatedModel, cmd := m.Update(enterKey)
	m = updatedModel.(*Model)

	// Assert
	if m.nav.Current() != verifyingPhase {
		t.Errorf("expected phase to be %v, but got %v", verifyingPhase, m.nav.Current())
	}
	if cmd == nil {
		t.Error("expected a command to be returned, but got nil")
	}
}

func TestUpdate_EnterOnCompletePhase_Finishes(t *testing.T) {
	// Arrange
	m := setupTestModel()
	m.nav.Push(cloneCompletePhase)

	enterKey := tea.KeyMsg{Type: tea.KeyEnter}

	// Act
	_, cmd := m.Update(enterKey)

	// Assert
	if cmd == nil {
		t.Fatal("expected a command but got nil")
	}

	msg := cmd()
	if _, ok := msg.(DotfilesFinished); !ok {
		t.Errorf("expected msg of type DotfilesFinished, but got %T", msg)
	}
}

func TestUpdate_ValidationResultSuccess(t *testing.T) {
	// Arrange
	m := setupTestModel()
	m.nav.Push(verifyingPhase)

	msg := validationResultMsg{err: nil}

	// Act
	updatedModel, cmd := m.Update(msg)
	m = updatedModel.(*Model)

	// Assert
	if m.nav.Current() != confirmationPhase {
		t.Errorf("expected phase to be %v, but got %v", confirmationPhase, m.nav.Current())
	}
	if cmd != nil {
		t.Errorf("expected a nil command, but got one")
	}
}

func TestUpdate_CloneResultSuccess(t *testing.T) {
	// Arrange
	m := setupTestModel()
	m.nav.Push(cloningPhase)

	msg := cloneResultMsg{err: nil}

	// Act
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(*Model)

	// Assert
	if m.nav.Current() != cloneCompletePhase {
		t.Errorf("expected phase to be %v, but got %v", cloneCompletePhase, m.nav.Current())
	}
}

func TestUpdate_CloneResultError(t *testing.T) {
	// Arrange
	m := setupTestModel()
	m.nav.Push(cloningPhase)

	msg := cloneResultMsg{err: errors.New("clone failed")}

	// Act
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(*Model)

	// Assert
	if m.err == nil {
		t.Error("expected model.err to be set, but it was nil")
	}
	if m.nav.Current() != inputPhase {
		t.Errorf("expected phase to be reset to %v, but got %v", inputPhase, m.nav.Current())
	}
}

func TestUpdate_EnterOnInputPhase_WithEmptyInput_SetsError(t *testing.T) {
	// Arrange
	m := setupTestModel()
	m.repoInput.SetValue("") // Empty repo input

	enterKey := tea.KeyMsg{Type: tea.KeyEnter}

	// Act
	updatedModel, _ := m.Update(enterKey)
	m = updatedModel.(*Model)

	// Assert
	if m.err == nil {
		t.Error("expected an error due to empty input, but got nil")
	}
	if m.nav.Current() != inputPhase {
		t.Errorf("expected to stay in the input phase, but got %v", m.nav.Current())
	}
}

func TestUpdate_EnterOnConfirmationPhase_StartsCloning(t *testing.T) {
	// Arrange
	m := setupTestModel()
	m.nav.Push(confirmationPhase)

	enterKey := tea.KeyMsg{Type: tea.KeyEnter}

	// Act
	updatedModel, cmd := m.Update(enterKey)
	m = updatedModel.(*Model)

	// Assert
	if m.nav.Current() != cloningPhase {
		t.Errorf("expected phase to be %v, but got %v", cloningPhase, m.nav.Current())
	}
	if cmd == nil {
		t.Error("expected a command to start cloning, but got nil")
	}
}

func TestUpdate_BackOnConfirmationPhase_ResetsToInput(t *testing.T) {
	// Arrange
	m := setupTestModel()
	m.nav.Push(confirmationPhase)

	backKey := tea.KeyMsg{Type: tea.KeyEscape}

	// Act
	updatedModel, _ := m.Update(backKey)
	m = updatedModel.(*Model)

	// Assert
	if m.nav.Current() != inputPhase {
		t.Errorf("expected phase to be %v, but got %v", inputPhase, m.nav.Current())
	}
}

func TestUpdate_TabOnInputPhase_TogglesFocus(t *testing.T) {
	// Arrange
	m := setupTestModel()
	tabKey := tea.KeyMsg{Type: tea.KeyTab}

	// Pre-condition assert
	if m.focusedInput != 0 {
		t.Fatalf("initial focused input should be 0, got %d", m.focusedInput)
	}

	// Act 1
	updatedModel, _ := m.Update(tabKey)
	m = updatedModel.(*Model)

	// Assert 1
	if m.focusedInput != 1 {
		t.Errorf("expected focused input to be 1 after tab, but got %d", m.focusedInput)
	}

	// Act 2
	updatedModel, _ = m.Update(tabKey)
	m = updatedModel.(*Model)

	// Assert 2
	if m.focusedInput != 0 {
		t.Errorf("expected focused input to be 0 after second tab, but got %d", m.focusedInput)
	}
}

func TestUpdate_WindowSizeMsg_SetsWidths(t *testing.T) {
	// Arrange
	m := setupTestModel()
	msg := tea.WindowSizeMsg{Width: 100, Height: 30}

	// Act
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(*Model)

	// Assert
	if m.width != 100 {
		t.Errorf("expected width to be 100, got %d", m.width)
	}
	if m.height != 30 {
		t.Errorf("expected height to be 30, got %d", m.height)
	}

	// The getInputWidth() func subtracts padding, so we check that
	// the input's width is less than the total width.
	if m.repoInput.Width >= 100 {
		t.Errorf("expected repoInput width to be less than 100, got %d", m.repoInput.Width)
	}
}

func TestUpdate_AuthStatusMsg_UpdatesUsernameAndInput(t *testing.T) {
	// Arrange
	m := setupTestModel()
	msg := github.AuthStatusMsg{Username: "gh-user"}

	// Act
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(*Model)

	// Assert
	if m.Username != "gh-user" {
		t.Errorf("expected Username to be 'gh-user', got '%s'", m.Username)
	}
	expectedRepo := "gh-user/dotfiles"
	if m.repoInput.Value() != expectedRepo {
		t.Errorf("expected repoInput value to be '%s', got '%s'", expectedRepo, m.repoInput.Value())
	}
}

func TestUpdate_BackKey_NoOpDuringAsync(t *testing.T) {
	tests := []struct {
		name  string
		phase phase
	}{
		{"VerifyingPhase", verifyingPhase},
		{"CloningPhase", cloningPhase},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			m := setupTestModel()
			m.nav.Push(tc.phase)
			backKey := tea.KeyMsg{Type: tea.KeyBackspace}

			// Act
			updatedModel, _ := m.Update(backKey)
			m = updatedModel.(*Model)

			// Assert
			if m.nav.Current() != tc.phase {
				t.Errorf("expected to remain in phase %v, but changed to %v", tc.phase, m.nav.Current())
			}
		})
	}
}

func TestUpdate_CharacterKeys_UpdatesFocusedInput(t *testing.T) {
	// Arrange
	m := setupTestModel()
	charKey := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")}

	// Act 1
	updatedModel, _ := m.Update(charKey)
	m = updatedModel.(*Model)

	// Assert 1
	if m.repoInput.Value() != "a" {
		t.Errorf("expected repoInput value to be 'a', but got '%s'", m.repoInput.Value())
	}

	// Arrange 2: Change focus to destInput
	m.focusedInput = 1
	m.repoInput.Blur()
	m.destInput.Focus()
	initialDestValue := m.destInput.Value()

	// Act 2: Type 'b' into destInput
	charKeyB := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")}
	updatedModel, _ = m.Update(charKeyB)
	m = updatedModel.(*Model)

	// Assert 2
	expectedDestValue := initialDestValue + "b"
	if m.destInput.Value() != expectedDestValue {
		t.Errorf("expected destInput value to be '%s', but got '%s'", expectedDestValue, m.destInput.Value())
	}
}

func TestUpdate_BackOnInputPhase_ReturnsPhaseBackCmd(t *testing.T) {
	// Arrange
	m := setupTestModel()
	backKey := tea.KeyMsg{Type: tea.KeyEscape}

	// Act
	_, cmd := m.Update(backKey)

	// Assert
	if cmd == nil {
		t.Fatal("expected a command but got nil")
	}
	msg := cmd()
	if _, ok := msg.(types.PhaseBack); !ok {
		t.Errorf("expected msg of type types.PhaseBack, but got %T", msg)
	}
}

func TestUpdate_NavKeysOnInputPhase_ChangesFocus(t *testing.T) {
	testCases := []struct {
		name          string
		initialFocus  int
		key           tea.KeyMsg
		expectedFocus int
	}{
		{
			name:          "ShiftTab from repo input focuses dest input",
			initialFocus:  0,
			key:           tea.KeyMsg{Type: tea.KeyShiftTab},
			expectedFocus: 1,
		},
		{
			name:          "ShiftTab from dest input focuses repo input",
			initialFocus:  1,
			key:           tea.KeyMsg{Type: tea.KeyShiftTab},
			expectedFocus: 0,
		},
		{
			name:          "Down key from repo input focuses dest input",
			initialFocus:  0,
			key:           tea.KeyMsg{Type: tea.KeyDown},
			expectedFocus: 1,
		},
		{
			name:          "Up key from dest input focuses repo input",
			initialFocus:  1,
			key:           tea.KeyMsg{Type: tea.KeyUp},
			expectedFocus: 0,
		},
		{
			name:          "Down key from dest input does not change focus",
			initialFocus:  1,
			key:           tea.KeyMsg{Type: tea.KeyDown},
			expectedFocus: 1,
		},
		{
			name:          "Up key from repo input does not change focus",
			initialFocus:  0,
			key:           tea.KeyMsg{Type: tea.KeyUp},
			expectedFocus: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			m := setupTestModel()
			m.focusedInput = tc.initialFocus

			// Act
			updatedModel, _ := m.Update(tc.key)
			m = updatedModel.(*Model)

			// Assert
			if m.focusedInput != tc.expectedFocus {
				t.Errorf("expected focusedInput to be %d, but got %d", tc.expectedFocus, m.focusedInput)
			}
		})
	}
}

func TestUpdate_FocusChanges_OnNavKeys(t *testing.T) {
	// Arrange
	m := setupTestModel()
	tabKey := tea.KeyMsg{Type: tea.KeyTab}

	// Assert initial state
	if !m.repoInput.Focused() {
		t.Fatal("repoInput should be focused initially")
	}
	if m.destInput.Focused() {
		t.Fatal("destInput should not be focused initially")
	}

	// Act: Press Tab to switch focus to destInput
	updatedModel, _ := m.Update(tabKey)
	m = updatedModel.(*Model)

	// Assert
	if m.repoInput.Focused() {
		t.Error("repoInput should be blurred after tabbing")
	}
	if !m.destInput.Focused() {
		t.Error("destInput should be focused after tabbing")
	}
}

func TestUpdate_ValidationResult_DirExists_NavigatesToDirExistsPhase(t *testing.T) {
	// Arrange
	m := setupTestModel()
	m.nav.Push(verifyingPhase)
	msg := validationResultMsg{err: nil, DirAlreadyExists: true}

	// Act
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(*Model)

	// Assert
	if m.nav.Current() != dirExistsPhase {
		t.Errorf("expected phase to be %v, but got %v", dirExistsPhase, m.nav.Current())
	}
}

func TestUpdate_EnterOnDirExistsPhase_Finishes(t *testing.T) {
	// Arrange
	m := setupTestModel()
	m.nav.Push(dirExistsPhase)
	enterKey := tea.KeyMsg{Type: tea.KeyEnter}

	// Act
	_, cmd := m.Update(enterKey)

	// Assert
	if cmd == nil {
		t.Fatal("expected a command but got nil")
	}

	msg := cmd()
	if _, ok := msg.(DotfilesFinished); !ok {
		t.Errorf("expected msg of type DotfilesFinished, but got %T", msg)
	}
}

func TestUpdate_BackOnDirExistsPhase_ResetsToInput(t *testing.T) {
	// Arrange
	m := setupTestModel()
	m.nav.Push(dirExistsPhase)
	backKey := tea.KeyMsg{Type: tea.KeyEscape}

	// Act
	updatedModel, _ := m.Update(backKey)
	m = updatedModel.(*Model)

	// Assert
	if m.nav.Current() != inputPhase {
		t.Errorf("expected phase to be %v, but got %v", inputPhase, m.nav.Current())
	}
}
