package system

import (
	"os"
	"os/exec"
)

type LiveExecutor struct{}
type LiveFileSystem struct{}

func (l *LiveExecutor) Run(cmd *exec.Cmd) error {
	return cmd.Run()
}

func (l *LiveExecutor) RunPiped(cmd1 *exec.Cmd, cmd2 *exec.Cmd) error {
	pipe, err := cmd1.StdoutPipe()
	if err != nil {
		return err
	}
	cmd2.Stdin = pipe

	if err := cmd1.Start(); err != nil {
		return err
	}
	if err := cmd2.Start(); err != nil {
		return err
	}

	if err := cmd1.Wait(); err != nil {
		return err
	}

	return cmd2.Wait()
}

func (e LiveExecutor) Output(cmd *exec.Cmd) ([]byte, error) {
	return cmd.Output()
}

func (fs LiveFileSystem) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}
func (fs LiveFileSystem) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}
func (fs LiveFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}
func (fs LiveFileSystem) CreateTemp(dir, pattern string) (*os.File, error) {
	return os.CreateTemp(dir, pattern)
}
func (fs LiveFileSystem) Remove(name string) error {
	return os.Remove(name)
}
func (fs LiveFileSystem) ReadDir(name string) ([]os.DirEntry, error) {
	return os.ReadDir(name)
}
func (e *LiveExecutor) CombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	return cmd.CombinedOutput()
}
func (fs LiveFileSystem) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}
func (fs LiveFileSystem) Open(name string) (*os.File, error) {
	return os.Open(name)
}
func (fs LiveFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (l *LiveExecutor) IsRoot() bool {
	return os.Geteuid() == 0
}

func (l *LiveExecutor) CanSudo() bool {
	return exec.Command("sudo", "-n", "true").Run() == nil
}
