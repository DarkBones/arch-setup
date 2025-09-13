package github_auth

import (
	"archsetup/internal/types"
	"errors"
	"io/fs"
	"os/exec"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// --- Mocks ---

type mockExecutor struct {
	runErr    error
	output    []byte
	outputErr error
}

func (m *mockExecutor) Run(cmd *exec.Cmd) error {
	return m.runErr
}
func (m *mockExecutor) Output(cmd *exec.Cmd) ([]byte, error) {
	return m.output, m.outputErr
}

type mockFileSystem struct {
	homeDir       string
	homeDirErr    error
	mkdirTempDir  string
	mkdirTempErr  error
	mkdirAllErr   error
	readFileData  []byte
	readFileErr   error
	appendFileErr error
	isNotExist    bool
}

func (m *mockFileSystem) UserHomeDir() (string, error) {
	return m.homeDir, m.homeDirErr
}
func (m *mockFileSystem) MkdirTemp(dir, pattern string) (string, error) {
	return m.mkdirTempDir, m.mkdirTempErr
}
func (m *mockFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return m.mkdirAllErr
}
func (m *mockFileSystem) ReadFile(name string) ([]byte, error) {
	return m.readFileData, m.readFileErr
}
func (m *mockFileSystem) AppendFile(name string, data []byte, perm fs.FileMode) error {
	return m.appendFileErr
}
func (m *mockFileSystem) IsNotExist(err error) bool {
	return m.isNotExist
}

type mockAuthenticator struct {
	isAuthenticated bool
	username        string
	output          string
}

func (m *mockAuthenticator) CheckConnection() (bool, string, string) {
	return m.isAuthenticated, m.username, m.output
}

// --- Test Setup ---

func setupTestService(fs FileSystem, exec Executor, auth Authenticator) *Service {
	return NewService(fs, exec, auth)
}

func setupTestModel(service *Service) *Model {
	return New(types.DefaultKeys(), service).(*Model)
}

// --- Service Tests ---

func TestService_GetSshPath(t *testing.T) {
	t.Parallel()

	t.Run("Normal mode", func(t *testing.T) {
		t.Parallel()
		fs := &mockFileSystem{homeDir: "/home/user", mkdirAllErr: nil}
		service := NewService(fs, nil, nil)

		path, err := service.getSshPath()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasSuffix(path, ".ssh") {
			t.Errorf("expected path to end with '.ssh', got '%s'", path)
		}
	})

	t.Run("Normal mode home dir error", func(t *testing.T) {
		t.Parallel()
		fs := &mockFileSystem{homeDirErr: errors.New("home dir error")}
		service := NewService(fs, nil, nil)

		_, err := service.getSshPath()
		if err == nil {
			t.Fatal("expected an error but got nil")
		}
	})
}

func TestService_GenerateKeyCmd(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		fs := &mockFileSystem{
			homeDir:      "/home/user",
			readFileData: []byte("ssh-key-content"),
		}
		exec := &mockExecutor{}
		service := setupTestService(fs, exec, nil)

		cmd := service.GenerateKeyCmd()
		msg := cmd()

		res, ok := msg.(keyGeneratedMsg)
		if !ok {
			t.Fatalf("expected keyGeneratedMsg, got %T", msg)
		}
		if res.publicKey != "ssh-key-content" {
			t.Errorf("unexpected public key content")
		}
	})

	t.Run("ssh-keygen fails", func(t *testing.T) {
		t.Parallel()
		fs := &mockFileSystem{homeDir: "/home/user"}
		exec := &mockExecutor{runErr: errors.New("keygen failed")}
		service := setupTestService(fs, exec, nil)

		msg := service.GenerateKeyCmd()()
		_, ok := msg.(errMsg)
		if !ok {
			t.Fatalf("expected errMsg, got %T", msg)
		}
	})

	t.Run("getSshPath fails", func(t *testing.T) {
		t.Parallel()
		fs := &mockFileSystem{homeDirErr: errors.New("home dir error")}
		service := setupTestService(fs, &mockExecutor{}, nil)

		msg := service.GenerateKeyCmd()()
		_, ok := msg.(errMsg)
		if !ok {
			t.Fatalf("expected errMsg, got %T", msg)
		}
	})

	t.Run("ReadFile fails", func(t *testing.T) {
		t.Parallel()
		fs := &mockFileSystem{
			homeDir:     "/home/user",
			readFileErr: errors.New("read error"),
		}
		service := setupTestService(fs, &mockExecutor{}, nil)

		msg := service.GenerateKeyCmd()()
		_, ok := msg.(errMsg)
		if !ok {
			t.Fatalf("expected errMsg, got %T", msg)
		}
	})

	t.Run("Empty public key", func(t *testing.T) {
		t.Parallel()
		fs := &mockFileSystem{
			homeDir:      "/home/user",
			readFileData: []byte{}, // Empty key
		}
		service := setupTestService(fs, &mockExecutor{}, nil)

		msg := service.GenerateKeyCmd()()
		_, ok := msg.(errMsg)
		if !ok {
			t.Fatalf("expected errMsg for empty key, got %T", msg)
		}
	})
}

func TestService_CheckKeyCmd(t *testing.T) {
	t.Parallel()

	t.Run("Key exists and is authenticated", func(t *testing.T) {
		t.Parallel()
		fs := &mockFileSystem{homeDir: "/home/user", readFileData: []byte("key")}
		auth := &mockAuthenticator{isAuthenticated: true, username: "testuser"}
		service := setupTestService(fs, &mockExecutor{}, auth)

		msg := service.CheckKeyCmd()()
		res, ok := msg.(keyCheckResultMsg)
		if !ok {
			t.Fatalf("expected keyCheckResultMsg, got %T", msg)
		}
		if !res.keyExists || !res.isAuthenticated || res.username != "testuser" {
			t.Errorf("unexpected result: %+v", res)
		}
	})

	t.Run("Key exists but is not authenticated", func(t *testing.T) {
		t.Parallel()
		fs := &mockFileSystem{homeDir: "/home/user", readFileData: []byte("key")}
		auth := &mockAuthenticator{isAuthenticated: false}
		service := setupTestService(fs, &mockExecutor{}, auth)

		msg := service.CheckKeyCmd()()
		res, ok := msg.(keyCheckResultMsg)
		if !ok {
			t.Fatalf("expected keyCheckResultMsg, got %T", msg)
		}
		if !res.keyExists || res.isAuthenticated {
			t.Errorf("unexpected result: %+v", res)
		}
	})

	t.Run("Key does not exist", func(t *testing.T) {
		t.Parallel()
		fs := &mockFileSystem{
			homeDir:     "/home/user",
			readFileErr: errors.New("not exist"),
			isNotExist:  true,
		}
		service := setupTestService(fs, &mockExecutor{}, nil)

		msg := service.CheckKeyCmd()()
		res, ok := msg.(keyCheckResultMsg)
		if !ok {
			t.Fatalf("expected keyCheckResultMsg, got %T", msg)
		}
		if res.keyExists {
			t.Error("expected keyExists to be false")
		}
	})

	t.Run("ensureGitHubKnownHost fails", func(t *testing.T) {
		t.Parallel()
		fs := &mockFileSystem{homeDirErr: errors.New("getSshPath failed")}
		service := setupTestService(fs, &mockExecutor{}, nil)

		msg := service.CheckKeyCmd()()
		_, ok := msg.(errMsg)
		if !ok {
			t.Fatalf("expected errMsg, got %T", msg)
		}
	})
}

func TestService_VerifyConnectionCmd(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		auth := &mockAuthenticator{isAuthenticated: true, username: "testuser"}
		fs := &mockFileSystem{homeDir: "/home/user"}
		service := setupTestService(fs, &mockExecutor{}, auth)

		cmd := service.VerifyConnectionCmd()
		msg := cmd()

		res, ok := msg.(verificationSuccessMsg)
		if !ok {
			t.Fatalf("expected verificationSuccessMsg, got %T", msg)
		}
		if res.username != "testuser" {
			t.Errorf("expected username 'testuser', got '%s'", res.username)
		}
	})

	t.Run("Failure", func(t *testing.T) {
		t.Parallel()
		auth := &mockAuthenticator{isAuthenticated: false, output: "permission denied"}
		fs := &mockFileSystem{homeDir: "/home/user"}
		service := setupTestService(fs, &mockExecutor{}, auth)

		cmd := service.VerifyConnectionCmd()
		msg := cmd()

		res, ok := msg.(verificationFailedMsg)
		if !ok {
			t.Fatalf("expected verificationFailedMsg, got %T", msg)
		}
		if !strings.Contains(res.err.Error(), "permission denied") {
			t.Errorf("unexpected error message: %v", res.err)
		}
	})
}

// --- Model Tests ---

func TestModel_Update_KeyCheckResult(t *testing.T) {
	t.Parallel()
	service := setupTestService(&mockFileSystem{}, &mockExecutor{}, &mockAuthenticator{})

	t.Run("No key exists", func(t *testing.T) {
		t.Parallel()
		m := setupTestModel(service)
		msg := keyCheckResultMsg{keyExists: false}

		updatedModel, cmd := m.Update(msg)
		m = updatedModel.(*Model)

		if m.nav.Current() != generatingKey {
			t.Errorf("expected phase generatingKey, got %v", m.nav.Current())
		}
		if cmd == nil {
			t.Error("expected a command to be returned")
		}
	})

	t.Run("Key exists but not authenticated", func(t *testing.T) {
		t.Parallel()
		m := setupTestModel(service)
		msg := keyCheckResultMsg{
			keyExists:       true,
			isAuthenticated: false,
			publicKey:       "my-key",
		}

		updatedModel, cmd := m.Update(msg)
		m = updatedModel.(*Model)

		if m.nav.Current() != displayingKey {
			t.Errorf("expected phase displayingKey, got %v", m.nav.Current())
		}
		if m.publicKey != "my-key" {
			t.Errorf("public key was not set on the model")
		}
		if cmd != nil {
			t.Error("expected no command to be returned")
		}
	})

	t.Run("Key exists and is authenticated", func(t *testing.T) {
		t.Parallel()
		m := setupTestModel(service)
		msg := keyCheckResultMsg{
			keyExists:       true,
			isAuthenticated: true,
			username:        "testuser",
		}

		updatedModel, _ := m.Update(msg)
		m = updatedModel.(*Model)

		if m.nav.Current() != authComplete {
			t.Errorf("expected phase authComplete, got %v", m.nav.Current())
		}
		if m.username != "testuser" {
			t.Errorf("username was not set on the model")
		}
	})
}

func TestModel_Update_KeyGenerated(t *testing.T) {
	t.Parallel()
	service := setupTestService(&mockFileSystem{}, &mockExecutor{}, &mockAuthenticator{})
	m := setupTestModel(service)
	m.nav.Push(generatingKey)
	msg := keyGeneratedMsg{publicKey: "new-public-key"}

	updatedModel, _ := m.Update(msg)
	m = updatedModel.(*Model)

	if m.nav.Current() != displayingKey {
		t.Errorf("expected phase displayingKey, got %v", m.nav.Current())
	}
	if m.publicKey != "new-public-key" {
		t.Error("public key was not set on the model")
	}
}

func TestModel_Update_Verification(t *testing.T) {
	t.Parallel()
	service := setupTestService(&mockFileSystem{}, &mockExecutor{}, &mockAuthenticator{})

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		m := setupTestModel(service)
		m.nav.Push(verifyingConnection)
		msg := verificationSuccessMsg{username: "verified-user"}

		updatedModel, _ := m.Update(msg)
		m = updatedModel.(*Model)

		if m.nav.Current() != finalSuccessPhase {
			t.Errorf("expected phase finalSuccessPhase, got %v", m.nav.Current())
		}
		if m.username != "verified-user" {
			t.Errorf("username was not updated")
		}
	})

	t.Run("Failure", func(t *testing.T) {
		t.Parallel()
		m := setupTestModel(service)
		m.nav.Push(verifyingConnection)
		testErr := errors.New("verification failed")
		msg := verificationFailedMsg{err: testErr}

		updatedModel, _ := m.Update(msg)
		m = updatedModel.(*Model)

		if m.nav.Current() != authError {
			t.Errorf("expected phase authError, got %v", m.nav.Current())
		}
		if m.err != testErr {
			t.Errorf("error was not set on the model")
		}
	})
}

func TestModel_Update_KeyMsg(t *testing.T) {
	t.Parallel()
	service := setupTestService(&mockFileSystem{}, &mockExecutor{}, &mockAuthenticator{})

	t.Run("Enter on displayingKey triggers verification", func(t *testing.T) {
		t.Parallel()
		m := setupTestModel(service)
		m.nav.Push(displayingKey)
		msg := tea.KeyMsg{Type: tea.KeyEnter}

		updatedModel, cmd := m.Update(msg)
		m = updatedModel.(*Model)

		if m.nav.Current() != verifyingConnection {
			t.Errorf("expected phase verifyingConnection, got %v", m.nav.Current())
		}
		if cmd == nil {
			t.Error("expected a verification command")
		}
	})

	t.Run("Back on displayingKey cancels", func(t *testing.T) {
		t.Parallel()
		m := setupTestModel(service)
		m.nav.Push(displayingKey)
		msg := tea.KeyMsg{Type: tea.KeyBackspace}

		_, cmd := m.Update(msg)
		res := cmd()

		if _, ok := res.(types.PhaseCancelled); !ok {
			t.Errorf("expected types.PhaseCancelled, got %T", res)
		}
	})

	t.Run("Enter on finalSuccessPhase finishes", func(t *testing.T) {
		t.Parallel()
		m := setupTestModel(service)
		m.nav.Push(finalSuccessPhase)
		msg := tea.KeyMsg{Type: tea.KeyEnter}

		_, cmd := m.Update(msg)
		res := cmd()

		if _, ok := res.(types.PhaseFinished); !ok {
			t.Errorf("expected types.PhaseFinished, got %T", res)
		}
	})
}

func TestService_EnsureGitHubKnownHost(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		fs := &mockFileSystem{homeDir: "/home/user"}
		exec := &mockExecutor{output: []byte("keyscan-output")}
		service := setupTestService(fs, exec, nil)

		err := service.ensureGitHubKnownHost()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ssh-keyscan fails", func(t *testing.T) {
		t.Parallel()
		fs := &mockFileSystem{homeDir: "/home/user"}
		exec := &mockExecutor{outputErr: errors.New("keyscan failed")}
		service := setupTestService(fs, exec, nil)

		err := service.ensureGitHubKnownHost()
		if err == nil {
			t.Fatal("expected an error but got nil")
		}
	})
}

func TestModel_Init(t *testing.T) {
	t.Parallel()
	service := setupTestService(&mockFileSystem{}, &mockExecutor{}, &mockAuthenticator{})
	m := New(types.DefaultKeys(), service).(*Model)

	cmd := m.Init()

	if m.nav.Current() != checkingKey {
		t.Errorf("expected initial phase to be checkingKey, got %v", m.nav.Current())
	}
	if m.err != nil || m.username != "" || m.publicKey != "" {
		t.Error("model state was not properly reset")
	}
	if cmd == nil {
		t.Error("expected a command to be returned from Init")
	}
}

func TestModel_Update_ErrMsg(t *testing.T) {
	t.Parallel()
	service := setupTestService(&mockFileSystem{}, &mockExecutor{}, &mockAuthenticator{})
	m := setupTestModel(service)
	testErr := errors.New("a test error")
	msg := errMsg{err: testErr}

	updatedModel, _ := m.Update(msg)
	m = updatedModel.(*Model)

	if m.nav.Current() != authError {
		t.Errorf("expected phase authError, got %v", m.nav.Current())
	}
	if m.err != testErr {
		t.Error("error was not set on the model")
	}
}

func TestModel_Update_WindowSizeMsg(t *testing.T) {
	t.Parallel()
	service := setupTestService(&mockFileSystem{}, &mockExecutor{}, &mockAuthenticator{})
	m := setupTestModel(service)
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}

	updatedModel, _ := m.Update(msg)
	m = updatedModel.(*Model)

	if m.width != 100 || m.height != 50 {
		t.Errorf("expected width=100, height=50, got width=%d, height=%d", m.width, m.height)
	}
	if m.viewport.Width != 100 || m.viewport.Height != 49 {
		t.Error("viewport dimensions not updated correctly")
	}
}
