package nvidia

import (
	"archsetup/internal/system"
	"archsetup/internal/types"
	"errors"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

var exitErrorOne *exec.ExitError

func init() {
	// The "false" command is a standard utility that does nothing and exits with code 1.
	cmd := exec.Command("false")
	// cmd.Run() returns the ExitError we want. It does NOT populate cmd.Err.
	err := cmd.Run()

	// The error returned by Run is of type *exec.ExitError.
	var ok bool
	exitErrorOne, ok = err.(*exec.ExitError)
	if !ok {
		panic("failed to create a mock ExitError for testing")
	}
}

// --- Mocks ---

type mockExecutor struct {
	runPipedErr error
	runErr      error
	output      []byte
	outputErr   error
	combined    []byte
	combinedErr error
	isRoot      bool
	canSudo     bool
}

func (m *mockExecutor) Run(cmd *exec.Cmd) error {
	return m.runErr
}

func (m *mockExecutor) RunPiped(cmd1 *exec.Cmd, cmd2 *exec.Cmd) error {
	return m.runPipedErr
}

func (m *mockExecutor) Output(cmd *exec.Cmd) ([]byte, error) {
	return m.output, m.outputErr
}

func (m *mockExecutor) IsRoot() bool {
	return m.isRoot
}

func (m *mockExecutor) CanSudo() bool {
	return m.canSudo
}

func (m *mockExecutor) CombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	return m.combined, m.combinedErr
}

// --- Test Helpers ---

func setupTestService(exec system.Executor) *Service {
	return NewService(exec)
}

func setupTestModel(service *Service) *Model {
	return New(types.DefaultKeys(), service)
}

// --- Service Tests ---

func TestService_HasNvidiaGpu(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		mockErr       error
		wantHasGpu    bool
		wantShouldErr bool
	}{
		{
			name:          "GPU is found",
			mockErr:       nil, // A nil error means grep exited with 0
			wantHasGpu:    true,
			wantShouldErr: false,
		},
		{
			name:          "GPU is not found",
			mockErr:       exitErrorOne,
			wantHasGpu:    false,
			wantShouldErr: false,
		},
		{
			name:          "Command fails unexpectedly",
			mockErr:       errors.New("lspci command not found"),
			wantHasGpu:    false,
			wantShouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockExecutor{runPipedErr: tt.mockErr}
			service := setupTestService(mockExec)

			hasGpu, err := service.HasNvidiaGpu()

			if hasGpu != tt.wantHasGpu {
				t.Errorf("want hasGpu %v, got %v", tt.wantHasGpu, hasGpu)
			}
			if (err != nil) != tt.wantShouldErr {
				t.Errorf("want shouldErr %v, got err %v", tt.wantShouldErr, err)
			}
		})
	}
}

func TestService_BuildInstallCommand(t *testing.T) {
	t.Parallel()
	service := setupTestService(&mockExecutor{})

	cmd := service.BuildInstallCommand()

	if !strings.HasSuffix(cmd.Path, "sudo") {
		t.Errorf("expected command to be 'sudo', but got '%s'", cmd.Path)
	}

	args := cmd.Args
	expectedInitialArgs := []string{"sudo", "pacman", "-S", "--noconfirm"}
	if !reflect.DeepEqual(args[:4], expectedInitialArgs) {
		t.Errorf("expected initial args to be %v, but got %v", expectedInitialArgs, args[:4])
	}

	found := false
	for _, arg := range args {
		if arg == "nvidia-dkms" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find 'nvidia-dkms' in command arguments, but it was missing")
	}
}

// --- Model Tests ---

func TestModel_Update(t *testing.T) {
	t.Parallel()

	t.Run("ConfirmationPhase: Enter on Yes starts installation", func(t *testing.T) {
		service := setupTestService(&mockExecutor{})
		m := setupTestModel(service)
		m.selection = true // "Yes" is selected

		updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = updatedModel.(*Model)

		if m.nav.Current() != installingPhase {
			t.Errorf("expected phase to be %v, but got %v", installingPhase, m.nav.Current())
		}
		if cmd == nil {
			t.Error("expected a command to be returned for installation, but got nil")
		}
	})

	t.Run("ConfirmationPhase: Enter on No cancels the phase", func(t *testing.T) {
		service := setupTestService(&mockExecutor{})
		m := setupTestModel(service)
		m.selection = false // "No" is selected

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		msg := cmd()

		if _, ok := msg.(types.PhaseCancelled); !ok {
			t.Errorf("expected msg of type types.PhaseCancelled, but got %T", msg)
		}
	})

	t.Run("ConfirmationPhase: Up/Down toggles selection", func(t *testing.T) {
		service := setupTestService(&mockExecutor{})
		m := setupTestModel(service)

		if m.selection != true {
			t.Fatal("initial selection should be true (Yes)")
		}

		updatedModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updatedModel.(*Model)

		if m.selection != false {
			t.Error("expected selection to be false (No) after KeyDown, but it was true")
		}

		updatedModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
		m = updatedModel.(*Model)

		if m.selection != true {
			t.Error("expected selection to be true (Yes) after KeyUp, but it was false")
		}
	})

	t.Run("InstallResultMsg: Success moves to successPhase", func(t *testing.T) {
		service := setupTestService(&mockExecutor{})
		m := setupTestModel(service)
		m.nav.Push(installingPhase)

		updatedModel, _ := m.Update(InstallResultMsg{Err: nil})
		m = updatedModel.(*Model)

		if m.nav.Current() != successPhase {
			t.Errorf("expected phase to be %v, but got %v", successPhase, m.nav.Current())
		}
		if m.err != nil {
			t.Errorf("expected model error to be nil, but got %v", m.err)
		}
	})

	t.Run("InstallResultMsg: Failure moves to errorPhase", func(t *testing.T) {
		service := setupTestService(&mockExecutor{})
		m := setupTestModel(service)
		m.nav.Push(installingPhase)
		installErr := errors.New("pacman failed")

		updatedModel, _ := m.Update(InstallResultMsg{Err: installErr})
		m = updatedModel.(*Model)

		if m.nav.Current() != errorPhase {
			t.Errorf("expected phase to be %v, but got %v", errorPhase, m.nav.Current())
		}
		if m.err != installErr {
			t.Errorf("expected model error to be '%v', but got '%v'", installErr, m.err)
		}
	})

	t.Run("FinalPhases: Enter or Back finishes the phase", func(t *testing.T) {
		phases := []phase{successPhase, errorPhase}
		keys := []tea.KeyType{tea.KeyEnter, tea.KeyBackspace}

		for _, p := range phases {
			for _, k := range keys {
				service := setupTestService(&mockExecutor{})
				m := setupTestModel(service)
				m.nav.Push(p)

				_, cmd := m.Update(tea.KeyMsg{Type: k})
				msg := cmd()

				if _, ok := msg.(types.PhaseFinished); !ok {
					t.Errorf("expected PhaseFinished on phase %v with key %v, but got %T", p, k, msg)
				}
			}
		}
	})
}
