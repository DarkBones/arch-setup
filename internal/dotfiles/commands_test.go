package dotfiles

import (
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

type mockExecutor struct {
	shouldError bool
	isRoot      bool
	canSudo     bool
	combined    []byte
	combinedErr error
}

func (m *mockExecutor) Run(cmd *exec.Cmd) error {
	if m.shouldError {
		return errors.New("mock command failed")
	}
	return nil
}

func (l *mockExecutor) RunPiped(cmd1 *exec.Cmd, cmd2 *exec.Cmd) error {
	if l.shouldError {
		return errors.New("mock command failed")
	}

	return nil
}

func (m *mockExecutor) Output(cmd *exec.Cmd) ([]byte, error) {
	if m.shouldError {
		return nil, errors.New("mock command failed")
	}
	return []byte("mock output"), nil
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

type mockFileInfo struct {
	isDir bool
}

type mockFileSystem struct {
	statInfo os.FileInfo
	statErr  error

	isNotExist bool

	mkDirAllErr error

	createTempFile *os.File
	createTempErr  error

	removeErr error

	readDirEntries []os.DirEntry
	readDirErr     error

	readFileContents []byte
	readFileErr      error

	openFile *os.File
	openErr  error
}

func (mfs *mockFileSystem) Stat(path string) (os.FileInfo, error) {
	return mfs.statInfo, mfs.statErr
}
func (mfs *mockFileSystem) IsNotExist(err error) bool {
	// For the mock, we can just return a pre-configured boolean.
	return mfs.isNotExist
}
func (mfs *mockFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return mfs.mkDirAllErr
}
func (mfs *mockFileSystem) CreateTemp(dir, pattern string) (*os.File, error) {
	// We don't need a real file for the test, so we can return nil.
	return mfs.createTempFile, mfs.createTempErr
}
func (mfs *mockFileSystem) Remove(name string) error {
	return mfs.removeErr
}
func (mfs *mockFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	return mfs.readDirEntries, mfs.readDirErr
}

func (mfs *mockFileSystem) ReadFile(name string) ([]byte, error) {
	return mfs.readFileContents, mfs.readFileErr
}

func (mfs *mockFileSystem) Open(name string) (*os.File, error) {
	return mfs.openFile, mfs.openErr
}

func (mfs *mockFileSystem) UserHomeDir() (string, error) {
	return "/home/mock", nil
}

func (mfi mockFileInfo) Name() string       { return "mock" }
func (mfi mockFileInfo) Size() int64        { return 0 }
func (mfi mockFileInfo) Mode() os.FileMode  { return 0 }
func (mfi mockFileInfo) ModTime() time.Time { return time.Now() }
func (mfi mockFileInfo) IsDir() bool        { return mfi.isDir }
func (mfi mockFileInfo) Sys() any           { return nil }

type mockDirEntry struct{}

func (mde *mockDirEntry) Name() string               { return "mock.entry" }
func (mde *mockDirEntry) IsDir() bool                { return false }
func (mde *mockDirEntry) Type() os.FileMode          { return 0 }
func (mde *mockDirEntry) Info() (os.FileInfo, error) { return nil, nil }

func TestCheckRepoExists_Success(t *testing.T) {
	// Arrange
	mockExec := &mockExecutor{shouldError: false}
	service := NewService(mockExec, nil)

	// Act
	err := service.CheckRepoExists("good/repo")

	// Assert
	if err != nil {
		t.Errorf("expected no error, but got: %v", err)
	}
}

func TestCheckRepoExists_Failure(t *testing.T) {
	// Arrange
	mockExec := &mockExecutor{shouldError: true}
	service := NewService(mockExec, nil)

	// Act
	err := service.CheckRepoExists("bad/repo")

	// Assert
	if err == nil {
		t.Error("expected an error, but got nil")
	}
}

func TestCheckRepoExists_InvalidFormat(t *testing.T) {
	// Arrange
	service := NewService(nil, nil)

	// Act
	err := service.CheckRepoExists("invalid-repo-format")

	// Assert
	if err == nil {
		t.Error("expected an error for invalid format, but got nil")
	}
}

func TestCheckDestIsValid_PathDoesNotExist_Success(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "mock-temp-")
	if err != nil {
		t.Fatalf("failed to create temp file for test setup: %v", err)
	}

	// Arrange
	mockFS := &mockFileSystem{
		statInfo:       &mockFileInfo{isDir: false},
		statErr:        os.ErrNotExist,
		isNotExist:     true,
		mkDirAllErr:    nil,
		createTempFile: tmpFile,
		createTempErr:  nil,
	}

	service := NewService(nil, mockFS)

	// Act
	err = service.CheckDestIsValid("/a/path/that/does/not/exist")

	// Assert
	if err != nil {
		t.Errorf("expected no error, but got: %v", err)
	}
}

func TestValidateCmd_Success(t *testing.T) {
	// Arrange
	mockExec := &mockExecutor{shouldError: false}

	mockFS := &mockFileSystem{
		statInfo:       &mockFileInfo{isDir: true},
		statErr:        nil,
		readDirEntries: []os.DirEntry{},
		readDirErr:     nil,
	}

	service := NewService(mockExec, mockFS)

	// Act
	cmd := service.ValidateCmd("good/repo", "/path/to/dest")
	msg := cmd()

	// Assert
	result, ok := msg.(validationResultMsg)
	if !ok {
		t.Fatalf("expected msg of type validationResultMsg, but got %T", msg)
	}

	if result.err != nil {
		t.Errorf("expected no error in message, but got: %v", result.err)
	}
	if result.path != "/path/to/dest" {
		t.Errorf("expected path '%s', but got '%s'", "/path/to/dest", result.path)
	}
}

func TestCheckDestIsValid_PathExistsAndIsNotEmpty(t *testing.T) {
	// Arrange
	mockFS := &mockFileSystem{
		statInfo:       &mockFileInfo{isDir: true},
		statErr:        nil,
		readDirEntries: []os.DirEntry{&mockDirEntry{}},
		readDirErr:     nil,
	}
	service := NewService(nil, mockFS)

	// Act
	err := service.CheckDestIsValid("/path/that/exists/and/is/not/empty")

	// Assert
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
	if !errors.Is(err, errDestDirExists) {
		t.Errorf("expected error to be errDestDirExists, but got %v", err)
	}
}

func TestValidateCmd_DestDirExists(t *testing.T) {
	// Arrange
	mockExec := &mockExecutor{shouldError: false}
	mockFS := &mockFileSystem{
		statInfo:       &mockFileInfo{isDir: true},
		statErr:        nil,
		readDirEntries: []os.DirEntry{&mockDirEntry{}},
		readDirErr:     nil,
	}
	service := NewService(mockExec, mockFS)

	// Act
	cmd := service.ValidateCmd("good/repo", "/path/to/dest")
	msg := cmd()

	// Assert
	result, ok := msg.(validationResultMsg)
	if !ok {
		t.Fatalf("expected msg of type validationResultMsg, but got %T", msg)
	}

	if result.err != nil {
		t.Errorf("expected no error in message, but got: %v", result.err)
	}
	if !result.DirAlreadyExists {
		t.Error("expected DirAlreadyExists to be true, but it was false")
	}
}
