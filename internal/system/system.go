package system

import (
	"os"
	"os/exec"
)

// Executor defines a common interface for running external commands.
type Executor interface {
	Run(cmd *exec.Cmd) error
	RunPiped(cmd1 *exec.Cmd, cmd2 *exec.Cmd) error
	Output(cmd *exec.Cmd) ([]byte, error)
	CombinedOutput(cmd *exec.Cmd) ([]byte, error)
	IsRoot() bool
	CanSudo() bool
}

// FileSystem defines a common interface for filesystem operations.
type FileSystem interface {
	Stat(path string) (os.FileInfo, error)
	IsNotExist(err error) bool
	MkdirAll(path string, perm os.FileMode) error
	CreateTemp(dir, pattern string) (*os.File, error)
	Remove(name string) error
	ReadDir(name string) ([]os.DirEntry, error)
	ReadFile(name string) ([]byte, error)
	Open(name string) (*os.File, error)
	UserHomeDir() (string, error)
}
