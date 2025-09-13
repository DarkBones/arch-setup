package github_auth

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	sshKeygenCmd  = "ssh-keygen"
	sshKeyscanCmd = "ssh-keyscan"
	sshKeyType    = "ed25519"
	sshKeyFile    = "id_ed25519"
	githubHost    = "github.com"
)

type verificationSuccessMsg struct {
	username string
}

type verificationFailedMsg struct {
	err error
}

type Service struct {
	fs      FileSystem
	exec    Executor
	auth    Authenticator
	isDebug bool
}

func NewService(
	fs FileSystem,
	exec Executor,
	auth Authenticator,
	isDebug bool,
) *Service {
	return &Service{
		fs:      fs,
		exec:    exec,
		auth:    auth,
		isDebug: isDebug,
	}
}

func (s *Service) getSshPath() (string, error) {
	if s.isDebug {
		dir, err := s.fs.MkdirTemp("", "archsetup-ssh-")
		if err != nil {
			return "", fmt.Errorf("could not make a temporary directory: %w", err)
		}
		log.Printf("github_auth: [DEBUG] Using temporary SSH directory: %s", dir)

		return dir, nil
	}

	home, err := s.fs.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find user home directory: %w", err)
	}

	sshPath := filepath.Join(home, ".ssh")
	if err := s.fs.MkdirAll(sshPath, 0700); err != nil {
		return "", fmt.Errorf("could not create directory %s: %w", sshPath, err)
	}
	return sshPath, nil
}

func (s *Service) GenerateKeyCmd() tea.Cmd {
	return func() tea.Msg {
		sshDir, err := s.getSshPath()
		if err != nil {
			return errMsg{err: err}
		}

		keyPath := filepath.Join(sshDir, sshKeyFile)
		log.Printf("github_auth: Generating key at: %s", keyPath)

		cmd := exec.Command(sshKeygenCmd, "-t", sshKeyType, "-N", "", "-f", keyPath)
		if err := s.exec.Run(cmd); err != nil {
			return errMsg{err: fmt.Errorf("failed to run ssh-keygen: %w", err)}
		}

		publicKeyBytes, err := s.fs.ReadFile(keyPath + ".pub")
		if err != nil {
			return errMsg{err: fmt.Errorf("failed to read new public key: %w", err)}
		}

		if len(publicKeyBytes) == 0 {
			return errMsg{err: fmt.Errorf("ssh-keygen created an empty public key file")}
		}

		return keyGeneratedMsg{publicKey: string(publicKeyBytes)}
	}
}

func (s *Service) CheckKeyCmd() tea.Cmd {
	return func() tea.Msg {
		if err := s.ensureGitHubKnownHost(); err != nil {
			return errMsg{err: err}
		}

		sshDir, err := s.getSshPath()
		if err != nil {
			return errMsg{err: err}
		}

		keyPath := filepath.Join(sshDir, sshKeyFile+".pub")
		content, err := s.fs.ReadFile(keyPath)
		if s.fs.IsNotExist(err) {
			return keyCheckResultMsg{keyExists: false}
		}
		if err != nil {
			return errMsg{err: fmt.Errorf("could not read ssh key: %w", err)}
		}

		log.Printf("github_auth: Existing key found. Verifying connection...")
		isAuthenticated, username, _ := s.auth.CheckConnection()

		return keyCheckResultMsg{
			keyExists:       true,
			isAuthenticated: isAuthenticated,
			username:        username,
			publicKey:       string(content),
		}
	}
}

func (s *Service) VerifyConnectionCmd() tea.Cmd {
	return func() tea.Msg {
		log.Printf("github_auth: Verifying connection to GitHub...")

		if err := s.ensureGitHubKnownHost(); err != nil {
			return errMsg{err: err}
		}

		success, username, output := s.auth.CheckConnection()
		if success {
			return verificationSuccessMsg{username: username}
		}

		return verificationFailedMsg{
			err: fmt.Errorf("SSH connection failed: %s", output),
		}
	}
}

func (s *Service) ensureGitHubKnownHost() error {
	sshDir, err := s.getSshPath()
	if err != nil {
		return err
	}

	knownHostsPath := filepath.Join(sshDir, "known_hosts")

	_ = s.exec.Run(exec.Command(sshKeygenCmd, "-f", knownHostsPath, "-R", githubHost))

	scanCmd := exec.Command(sshKeyscanCmd, "-t", "ed25519,ecdsa,rsa", githubHost)
	out, err := s.exec.Output(scanCmd)
	if err != nil {
		return fmt.Errorf("ssh-keyscan failed: %w", err)
	}
	if err := s.fs.AppendFile(knownHostsPath, out, 0o644); err != nil {
		return fmt.Errorf("failed to append to known_hosts: %w", err)
	}
	return nil
}
