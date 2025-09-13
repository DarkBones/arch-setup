package github

import (
	"bytes"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	sshCommand      = "ssh"
	sshAuthTestHost = "git@github.com"
)

var sshAuthTestArgs = []string{
	"-T",
	"-o", "BatchMode=yes",
	"-o", "StrictHostKeyChecking=no",
}

type AuthStatusMsg struct {
	IsAuthenticated bool
	Username        string
}

type executor interface {
	Run(cmd *exec.Cmd) error
}

type liveExecutor struct{}

// Run executes the given command using the os/exec package.
func (e liveExecutor) Run(cmd *exec.Cmd) error {
	return cmd.Run()
}

// Authenticator handles the logic for checking SSH authentication with GitHub
// by executing and parsing the output of an ssh command.
type Authenticator struct {
	exec executor
}

func newAuthenticator(exec executor) *Authenticator {
	return &Authenticator{exec: exec}
}

var defaultAuthenticator = newAuthenticator(liveExecutor{})

// CheckAuthCmd is a command that checks for a successful SSH connection
// to GitHub and returns an AuthStatusMsg.
func CheckAuthCmd() tea.Cmd {
	return func() tea.Msg {
		isAuthenticated, username, _ := IsSshConnectionSuccessful()
		return AuthStatusMsg{
			IsAuthenticated: isAuthenticated,
			Username:        username,
		}
	}
}

// IsSshConnectionSuccessful attempts an SSH connection to GitHub using the
// default authenticator.
func IsSshConnectionSuccessful() (isAuthenticated bool, username string, output string) {
	return defaultAuthenticator.checkConnection()
}

func (a *Authenticator) checkConnection() (isAuthenticated bool, username string, output string) {
	args := append(sshAuthTestArgs, sshAuthTestHost)
	cmd := exec.Command(sshCommand, args...)

	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf

	_ = a.exec.Run(cmd)

	fullOutput := buf.String()
	isAuth, user := parseSshOutput(fullOutput)
	return isAuth, user, fullOutput
}

func parseSshOutput(output string) (isAuthenticated bool, username string) {
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(line, "successfully authenticated") {
			isAuthenticated = true

			// Expected line format: "Hi <username>! You've successfully authenticated..."
			fields := strings.Fields(line)
			if len(fields) > 1 {
				// The username is the second field, ending with "!".
				username = strings.TrimSuffix(fields[1], "!")
			}

			return
		}
	}

	return false, ""
}
