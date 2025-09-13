package dotfiles

import (
	"archsetup/internal/system"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

type validationResultMsg struct {
	path             string
	err              error
	DirAlreadyExists bool
}

type cloneResultMsg struct {
	err error
}

type stowResultMsg struct {
	err error
}

type Service struct {
	exec system.Executor
	fs   system.FileSystem
}

var errDestDirExists = errors.New(
	"destination directory already exists and is not empty",
)

var directoriesToExcludeFromStow = []string{"system"}

func NewService(exec system.Executor, fs system.FileSystem) *Service {
	return &Service{
		exec: exec,
		fs:   fs,
	}
}

func (s *Service) CheckRepoExists(repoPath string) error {
	log.Printf("dotfiles: checking if repo exists: %s", repoPath)

	if len(strings.Split(repoPath, "/")) != 2 {
		return fmt.Errorf("invalid repository format, expected 'username/repo', got: %s", repoPath)
	}

	url := fmt.Sprintf("ssh://git@github.com/%s.git", repoPath)
	cmd := exec.Command("git", "ls-remote", url)

	if err := s.exec.Run(cmd); err != nil {
		log.Printf("dotfiles: error checking if repo exists: %v", err)
		return fmt.Errorf("repository not found or access denied: %v", err)
	}

	return nil
}

func (s *Service) CheckDestIsValid(destPath string) error {
	log.Printf("dotfiles: checking if destination path is valid: %s", destPath)

	info, err := s.fs.Stat(destPath)
	if err != nil {
		if s.fs.IsNotExist(err) {
			// Path doesn't exist. Try to create the parent directory.
			log.Printf("dotfiles: destination path does not exist, creating parent directory")
			parentDir := filepath.Dir(destPath)
			if err := s.fs.MkdirAll(parentDir, 0755); err != nil {
				log.Printf("dotfiles: could not create parent directory: %v", err)
				return fmt.Errorf("could not create parent directory: %w", err)
			}

			// Now check for write permissions.
			tmpFile, err := s.fs.CreateTemp(parentDir, ".perm-check-")
			if err != nil {
				log.Printf("dotfiles: could not create temp file: %v", err)
				return fmt.Errorf("no write permissions for %s", parentDir)
			}
			tmpFile.Close()
			s.fs.Remove(tmpFile.Name())
			return nil
		}

		log.Printf("dotfiles: error checking if destination path is valid: %v", err)
		return fmt.Errorf("could not stat destination path: %w", err)
	}

	if !info.IsDir() {
		log.Printf("dotfiles: destination path is not a directory")
		return fmt.Errorf("destination path exists but is not a directory")
	}

	entries, err := s.fs.ReadDir(destPath)
	if err != nil {
		log.Printf("dotfiles: error checking if destination path is valid: %v", err)
		return fmt.Errorf("could not read destination directory: %w", err)
	}

	if len(entries) > 0 {
		log.Printf("dotfiles: destination path is not empty")
		return errDestDirExists
	}

	return nil
}

// ValidateCmd runs all validation checks in parallel and returns a single message.
func (s *Service) ValidateCmd(repo, dest string) tea.Cmd {
	log.Printf("dotfiles: validating repo and destination")

	return func() tea.Msg {
		var wg sync.WaitGroup
		errs := make(chan error, 2)

		wg.Add(2)
		go func() {
			defer wg.Done()
			errs <- s.CheckRepoExists(repo)
		}()
		go func() {
			defer wg.Done()
			errs <- s.CheckDestIsValid(dest)
		}()
		wg.Wait()
		close(errs)

		var destExists bool
		for err := range errs {
			if errors.Is(err, errDestDirExists) {
				destExists = true
				continue
			}

			if err != nil {
				return validationResultMsg{err: err}
			}
		}
		return validationResultMsg{
			err:              nil,
			path:             dest,
			DirAlreadyExists: destExists,
		}
	}
}

func (s *Service) CloneRepoCmd(repo, dest string) tea.Cmd {
	log.Printf("dotfiles: cloning repo to destination")

	return func() tea.Msg {
		url := fmt.Sprintf("git@github.com:%s.git", repo)
		cmd := exec.Command("git", "clone", url, dest)

		if err := s.exec.Run(cmd); err != nil {
			return cloneResultMsg{
				err: fmt.Errorf("Failed to clone repo: %w", err),
			}
		}

		return cloneResultMsg{err: nil}
	}
}
