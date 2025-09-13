package github_auth

import (
	"archsetup/internal/github"
	"io/fs"
	"os"
	"os/exec"
)

// Executor defines an interface for running external commands.
type Executor interface {
	Run(cmd *exec.Cmd) error
	Output(cmd *exec.Cmd) ([]byte, error)
}

// FileSystem defines an interface for filesystem operations.
type FileSystem interface {
	UserHomeDir() (string, error)
	MkdirTemp(dir, pattern string) (string, error)
	MkdirAll(path string, perm os.FileMode) error
	ReadFile(name string) ([]byte, error)
	AppendFile(name string, data []byte, perm fs.FileMode) error
	IsNotExist(err error) bool
}

// Authenticator defines an interface for checking the GitHub SSH connection.
type Authenticator interface {
	CheckConnection() (isAuthenticated bool, username string, output string)
}

// NewDefaultService creates a service with live dependencies.
// This should be used in main.go.
func NewDefaultService() *Service {
	return NewService(
		LiveFileSystem{},
		LiveExecutor{},
		LiveAuthenticator{},
	)
}

// LiveExecutor is the live implementation of the Executor interface.
type LiveExecutor struct{}

func (e LiveExecutor) Run(cmd *exec.Cmd) error { return cmd.Run() }

func (e LiveExecutor) Output(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

// LiveFileSystem is the live implementation of the FileSystem interface.
type LiveFileSystem struct{}

func (fs LiveFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (fs LiveFileSystem) MkdirTemp(dir, pattern string) (string, error) {
	return os.MkdirTemp(dir, pattern)
}

func (fs LiveFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs LiveFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

func (fs LiveFileSystem) AppendFile(name string, data []byte, perm fs.FileMode) error {
	f, err := os.OpenFile(name, os.O_APPEND|os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func (fs LiveFileSystem) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

// LiveAuthenticator is the live implementation of the Authenticator interface.
type LiveAuthenticator struct{}

func (a LiveAuthenticator) CheckConnection() (bool, string, string) {
	return github.IsSshConnectionSuccessful()
}
