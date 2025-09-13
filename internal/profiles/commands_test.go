package profiles

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// --- Mocks ---

type mockExecutor struct {
	isRoot            bool
	canSudo           bool
	combinedOutput    []byte
	combinedOutputErr error
}

func (m *mockExecutor) Run(cmd *exec.Cmd) error {
	return nil
}
func (m *mockExecutor) RunPiped(cmd1 *exec.Cmd, cmd2 *exec.Cmd) error {
	return nil
}
func (m *mockExecutor) Output(cmd *exec.Cmd) ([]byte, error) {
	return nil, nil
}
func (m *mockExecutor) CombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	return m.combinedOutput, m.combinedOutputErr
}
func (m *mockExecutor) IsRoot() bool  { return m.isRoot }
func (m *mockExecutor) CanSudo() bool { return m.canSudo }

type mockFileSystem struct {
	StatFunc      func(path string) (os.FileInfo, error)
	readFileData  []byte
	readFileErr   error
	openReader    io.ReadCloser
	openErr       error
	homeDir       string
	homeDirErr    error
	createTempErr error
	removeErr     error
}

func (m *mockFileSystem) Stat(path string) (os.FileInfo, error) {
	if m.StatFunc != nil {
		return m.StatFunc(path)
	}
	return nil, errors.New("StatFunc not implemented for this test")
}

func (m *mockFileSystem) IsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist)
}
func (m *mockFileSystem) ReadFile(name string) ([]byte, error) {
	return m.readFileData, m.readFileErr
}
func (m *mockFileSystem) Open(name string) (*os.File, error) {
	if m.openErr != nil {
		return nil, m.openErr
	}
	r, w, _ := os.Pipe()
	go func() {
		defer w.Close()
		if m.openReader != nil {
			io.Copy(w, m.openReader)
		}
	}()
	return r, nil
}
func (m *mockFileSystem) UserHomeDir() (string, error) {
	return m.homeDir, m.homeDirErr
}
func (m *mockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return nil
}
func (m *mockFileSystem) CreateTemp(dir, pattern string) (*os.File, error) {
	if m.createTempErr != nil {
		return nil, m.createTempErr
	}
	// Return a real temp file for the test to succeed, which we'll clean up.
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		panic("test could not create a real temp file for mocking")
	}
	return f, nil
}

func (m *mockFileSystem) Remove(name string) error {
	return nil
}
func (m *mockFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	return nil, nil
}

type mockFileInfo struct {
	name  string
	isDir bool
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) IsDir() bool        { return m.isDir }
func (m mockFileInfo) Size() int64        { return 0 }
func (m mockFileInfo) Mode() os.FileMode  { return 0 }
func (m mockFileInfo) ModTime() time.Time { return time.Now() }
func (m mockFileInfo) Sys() any           { return nil }

// --- Test Helper ---

func setupService(exec *mockExecutor, fs *mockFileSystem) *Service {
	return NewService(exec, fs)
}

// --- Tests ---

func TestService_GetProfilesCmd(t *testing.T) {
	t.Run("it returns a loaded config on success", func(t *testing.T) {
		mockFS := &mockFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return &mockFileInfo{isDir: true}, nil
			},
			readFileData: []byte(`[[profiles]]
name = "Test Profile"`),
		}
		service := setupService(&mockExecutor{}, mockFS)

		msg := service.getProfilesCmd("/fake/dotfiles")()

		resultMsg, ok := msg.(profilesLoadedMsg)
		if !ok {
			t.Fatalf("Expected msg of type profilesLoadedMsg, but got %T", msg)
		}

		profilesLen := len(resultMsg.Config.Profiles)
		firstProfName := resultMsg.Config.Profiles[0].Name
		if profilesLen != 1 || firstProfName != "Test Profile" {
			t.Errorf("Unexpected profile data: %+v", resultMsg.Config.Profiles)
		}
	})

	t.Run("it returns not found when the config file is missing", func(t *testing.T) {
		mockFS := &mockFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				if strings.HasSuffix(path, profilesFileName) {
					return nil, os.ErrNotExist
				}
				return &mockFileInfo{isDir: true}, nil
			},
		}
		service := setupService(&mockExecutor{}, mockFS)

		msg := service.getProfilesCmd("/fake/dotfiles")()

		if _, ok := msg.(profilesNotFoundMsg); !ok {
			t.Fatalf(
				"Expected msg of type profilesNotFoundMsg, but got %T",
				msg,
			)
		}
	})

	t.Run("it returns an error if the dotfiles path does not exist", func(t *testing.T) {
		mockFS := &mockFileSystem{
			StatFunc: func(path string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
		}
		service := setupService(&mockExecutor{}, mockFS)

		msg := service.getProfilesCmd("/fake/dotfiles")()

		if _, ok := msg.(errMsg); !ok {
			t.Fatalf("Expected msg of type errMsg, but got %T", msg)
		}
	})
}

func TestService_LoadPackagesCmd(t *testing.T) {
	t.Run("it loads packages successfully", func(t *testing.T) {
		content := "package1\n# commented out\npackage2\n\npackage3"
		mockFS := &mockFileSystem{
			openReader: io.NopCloser(strings.NewReader(content)),
		}
		service := setupService(&mockExecutor{}, mockFS)

		msg := service.loadPackagesCmd("/fake", "packages.txt")()

		resultMsg, ok := msg.(packagesLoadedMsg)
		if !ok {
			t.Fatalf("Expected msg of type packagesLoadedMsg, but got %T", msg)
		}

		expected := []string{"package1", "package2", "package3"}
		if len(resultMsg.packages) != len(expected) {
			t.Fatalf(
				"Expected %d packages, got %d",
				len(expected),
				len(resultMsg.packages),
			)
		}
	})

	t.Run("it returns an error if the package file cannot be opened", func(t *testing.T) {
		mockFS := &mockFileSystem{
			openErr: errors.New("file not found"),
		}
		service := setupService(&mockExecutor{}, mockFS)

		msg := service.loadPackagesCmd("/fake", "packages.txt")()

		if _, ok := msg.(errMsg); !ok {
			t.Fatalf("Expected msg of type errMsg, but got %T", msg)
		}
	})
}

func TestService_StowCmd(t *testing.T) {
	t.Run("it runs stow successfully", func(t *testing.T) {
		mockFS := &mockFileSystem{homeDir: "/home/user"}
		mockExec := &mockExecutor{}
		service := setupService(mockExec, mockFS)

		msg := service.stowCmd("/dots", []string{"nvim", "git"})()

		resultMsg, ok := msg.(stowResultMsg)
		if !ok {
			t.Fatalf("Expected msg of type stowResultMsg, but got %T", msg)
		}
		if resultMsg.err != nil {
			t.Errorf("Expected nil error, but got %v", resultMsg.err)
		}
	})

	t.Run("it returns an error if stow fails", func(t *testing.T) {
		mockFS := &mockFileSystem{homeDir: "/home/user"}
		mockExec := &mockExecutor{
			combinedOutputErr: errors.New("stow failed"),
		}
		service := setupService(mockExec, mockFS)

		msg := service.stowCmd("/dots", []string{"nvim"})()

		resultMsg, ok := msg.(stowResultMsg)
		if !ok {
			t.Fatalf("Expected msg of type stowResultMsg, but got %T", msg)
		}
		if resultMsg.err == nil {
			t.Error("Expected an error, but got nil")
		}
	})
}

func TestService_InstallPackageCmd(t *testing.T) {
	t.Run("it creates a runner script and returns an exec command", func(t *testing.T) {
		mockExec := &mockExecutor{}
		mockFS := &mockFileSystem{}
		service := setupService(mockExec, mockFS)

		cmdFunc := service.installPackageCmd("pkg1")
		if cmdFunc == nil {
			t.Fatal("Expected a command function to be returned")
		}

		msg := cmdFunc()

		if msg == nil {
			t.Fatal("Expected a tea.Msg, but got nil")
		}

		msgType := fmt.Sprintf("%T", msg)
		if msgType != "tea.execMsg" {
			t.Errorf("Expected msg of type tea.execMsg, but got %T", msg)
		}
	})
}
