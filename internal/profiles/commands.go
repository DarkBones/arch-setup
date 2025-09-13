package profiles

import (
	"archsetup/internal/assert"
	"archsetup/internal/system"
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	tea "github.com/charmbracelet/bubbletea"
)

const profilesFileName = "bas_settings.toml"

type profilesLoadedMsg struct {
	Config Config
}

type profilesNotFoundMsg struct{}

type packagesLoadedMsg struct {
	packages []string
}

type installLogMsg struct {
	line string
}

type installationFinishedMsg struct {
	err error
}

type installStartedMsg struct{}

type packageInstallResultMsg struct {
	pkg string
	err error
}

type stowResultMsg struct {
	err error
}

type postInstallLogMsg struct {
	line string
}

type postInstallCompleteMsg struct {
	err error
}

type errMsg struct{ err error }

type Service struct {
	exec system.Executor
	fs   system.FileSystem
}

type startStreamingCmdMsg struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
	stderr io.ReadCloser
}

type yayCheckResultMsg struct {
	isInstalled bool
}

type yayInstallResultMsg struct {
	err error
}

func NewService(
	exec system.Executor,
	fs system.FileSystem,
) *Service {
	return &Service{
		exec: exec,
		fs:   fs,
	}
}

func (s *Service) getProfilesCmd(dotfilesPath string) tea.Cmd {
	return func() tea.Msg {
		info, err := s.fs.Stat(dotfilesPath)
		if s.fs.IsNotExist(err) {
			return errMsg{
				fmt.Errorf("dotfiles path does not exist: %s", dotfilesPath),
			}
		}
		if err != nil {
			return errMsg{
				fmt.Errorf("error accessing dotfiles path: %w", err),
			}
		}
		if !info.IsDir() {
			return errMsg{
				fmt.Errorf(
					"dotfiles path is not a directory: %s",
					dotfilesPath,
				),
			}
		}

		configPath := filepath.Join(dotfilesPath, profilesFileName)
		if _, err := s.fs.Stat(configPath); s.fs.IsNotExist(err) {
			return profilesNotFoundMsg{}
		}

		data, err := s.fs.ReadFile(configPath)
		if err != nil {
			return errMsg{
				fmt.Errorf("could not read %s: %w", profilesFileName, err),
			}
		}

		var cfg Config
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return errMsg{
				fmt.Errorf("invalid %s format: %w", profilesFileName, err),
			}
		}

		return profilesLoadedMsg{Config: cfg}
	}
}

func (s *Service) getDefaultProfiles() []Profile {
	return nil
}

func (s *Service) loadPackagesCmd(
	dotfilesPath, profilePackagepath string,
) tea.Cmd {
	return func() tea.Msg {
		fullPath := filepath.Join(dotfilesPath, profilePackagepath)

		file, err := s.fs.Open(fullPath)
		if err != nil {
			return errMsg{
				fmt.Errorf(
					"could not open package list %s: %w",
					fullPath,
					err,
				),
			}
		}
		defer file.Close()

		var packages []string
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				packages = append(packages, line)
			}
		}

		if err := scanner.Err(); err != nil {
			return errMsg{
				fmt.Errorf(
					"error reading package list: %s: %w",
					fullPath,
					err,
				),
			}
		}

		return packagesLoadedMsg{packages: packages}
	}
}

// installPackagesCmd decides how to run yay:
// - if root: run directly (streaming)
// - else if sudo timestamp cached: sudo -n (streaming)
// - else: interactive takeover so user can type password (no stream)
func (s *Service) installPackageCmd(pkg string) tea.Cmd {
	info := system.CurrentOSInfo()
	var cmd *exec.Cmd

	switch info.Family {
	case "darwin":
		sh := fmt.Sprintf(`brew list --formula %[1]s >/dev/null 2>&1 || brew install %[1]s || brew list --cask %[1]s >/dev/null 2>&1 || brew install --cask %[1]s`, pkg)
		cmd = exec.Command("bash", "-lc", sh)

	case "linux":
		if isArchLike(info.Distro) {
			scriptPath, err := s.createInstallRunner(pkg)
			if err != nil {
				return func() tea.Msg { return packageInstallResultMsg{pkg: pkg, err: err} }
			}

			// NOTE: the temp file is cleaned up by the runner itself or on reboot; avoid removing here
			cmd = exec.Command(scriptPath)
		} else {
			return func() tea.Msg {
				return packageInstallResultMsg{
					pkg: pkg,
					err: fmt.Errorf("unsupported Linux distro for package install: %s", info.Distro),
				}
			}
		}

	default:
		return func() tea.Msg {
			return packageInstallResultMsg{
				pkg: pkg,
				err: fmt.Errorf("unsupported OS: %s", info.Family),
			}
		}
	}

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			return packageInstallResultMsg{pkg: pkg, err: err}
		}
		return packageInstallResultMsg{pkg: pkg, err: nil}
	})
}

func (s *Service) createInstallRunner(pkg string) (string, error) {
	scriptFile, err := os.CreateTemp("", "archsetup-runner-*.sh")
	if err != nil {
		return "", fmt.Errorf("failed to create temp runner script: %w", err)
	}
	defer scriptFile.Close()

	scriptContent := fmt.Sprintf(`#!/bin/sh
set -e
echo "--- Running installer for %s ---"
yay -S --noconfirm --needed %s
`, pkg, pkg)

	_, err = scriptFile.WriteString(scriptContent)
	if err != nil {
		return "", fmt.Errorf("failed to write to runner script: %w", err)
	}

	// Make the script executable.
	if err := os.Chmod(scriptFile.Name(), 0755); err != nil {
		return "", fmt.Errorf("failed to make runner script executable: %w", err)
	}

	log.Printf("profiles: Created runner script for package %s at %s", pkg, scriptFile.Name())
	return scriptFile.Name(), nil
}

func (s *Service) stowCmd(sourceDir string, stowDirs []string) tea.Cmd {
	return func() tea.Msg {
		if len(stowDirs) == 0 {
			log.Println("profiles: No directories specified to stow.")
			return stowResultMsg{err: nil}
		}

		home, err := s.fs.UserHomeDir()
		if err != nil {
			return errMsg{fmt.Errorf("could not get user home dir: %w", err)}
		}

		args := []string{"-t", home, "-R"}
		args = append(args, stowDirs...)

		cmd := exec.Command("stow", args...)
		cmd.Dir = sourceDir

		if output, err := s.exec.CombinedOutput(cmd); err != nil {
			return stowResultMsg{
				err: fmt.Errorf(
					"stow failed: %w\nOutput: %s",
					err,
					string(output),
				),
			}
		}

		return stowResultMsg{err: nil}
	}
}

func (s *Service) RunPostInstallCmd(
	dotfilesPath string,
	cmd PostInstallCommand,
	extraEnv map[string]string,
) tea.Cmd {
	workingDir := filepath.Join(dotfilesPath, cmd.WorkingDir)
	log.Printf(
		"Running post-install command '%s' in dir '%s'",
		cmd.Command,
		workingDir,
	)

	// We use "sh -c" to properly handle commands like "./bootstrap.sh"
	// that rely on the shell's path resolution and execution semantics.
	execCmd := exec.Command("sh", "-c", cmd.Command)
	execCmd.Dir = workingDir

	execCmd.Env = os.Environ()
	for k, v := range extraEnv {
		execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	return tea.ExecProcess(execCmd, func(err error) tea.Msg {
		return postInstallCompleteMsg{err: err}
	})
}

// On Arch Linux, we check/install yay. On MacOS we check/install Homebrew (+ ansible, stow).
func (s *Service) CheckYayCmd() tea.Cmd {
	return s.CheckPkgMgrCmd()
}

func (s *Service) CheckPkgMgrCmd() tea.Cmd {
	return func() tea.Msg {
		info := system.CurrentOSInfo()
		switch info.Family {
		case "darwin":
			_, err := exec.LookPath("brew")
			return yayCheckResultMsg{isInstalled: err == nil}
		case "linux":
			if isArchLike(info.Distro) {
				_, err := exec.LookPath("yay")
				return yayCheckResultMsg{isInstalled: err == nil}
			}

			assert.Fail("Unsupported Linux for package (for now)")
		}

		assert.Fail("Unsupported OS for package (for now)")

		return yayCheckResultMsg{isInstalled: true} // unreachable
	}
}

func isArchLike(distro string) bool {
	d := strings.ToLower(distro)

	switch d {
	case "arch", "archlinux", "manjaro", "endeavouros", "garuda", "archarm":
		return true
	}

	return false
}

// Add this new method to your Service
func (s *Service) InstallYayCmd() tea.Cmd {
	// Commands to install git and base-devel, clone yay, and run makepkg
	script := `
		set -e
		echo "--- Installing dependencies for yay (git, base-devel) ---"
		sudo pacman -S --noconfirm --needed git base-devel
		
		echo "--- Cloning yay from AUR ---"
		cd /tmp
		if [ -d "yay" ]; then rm -rf yay; fi
		git clone https://aur.archlinux.org/yay.git
		
		echo "--- Building and installing yay ---"
		cd yay
		makepkg -si --noconfirm
		
		echo "--- Cleaning up ---"
		cd /tmp
		rm -rf yay
		
		echo "--- yay installation complete! ---"
	`
	cmd := exec.Command("bash", "-c", script)

	// Use tea.ExecProcess to get a nice streaming output in the UI
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return yayInstallResultMsg{err: err}
	})
}

func (s *Service) InstallPkgMgrCmd() tea.Cmd {
	info := system.CurrentOSInfo()
	if info.Family == "darwin" {
		script := `
            set -e
            if ! command -v brew >/dev/null 2>&1; then
              echo '--- Installing Homebrew ---'
              /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
              test -x /opt/homebrew/bin/brew && eval "$(/opt/homebrew/bin/brew shellenv)"
              test -x /usr/local/bin/brew   && eval "$(/usr/local/bin/brew shellenv)"
            fi
            echo '--- Ensuring prerequisites on macOS ---'
            brew install ansible stow
        `
		return tea.ExecProcess(exec.Command("bash", "-c", script), func(err error) tea.Msg {
			return yayInstallResultMsg{err: err}
		})
	}
	if info.Family == "linux" && isArchLike(info.Distro) {
		return s.InstallYayCmd() // your existing yay bootstrap
	}
	// other Linux: nothing to install (we won't try packages)
	return tea.ExecProcess(exec.Command("bash", "-c", "true"), func(err error) tea.Msg {
		return yayInstallResultMsg{err: nil}
	})
}
