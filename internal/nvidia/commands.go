package nvidia

import (
	"archsetup/internal/system"
	"fmt"
	"log"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
)

type GpuCheckResultMsg struct {
	HasNvidiaGpu bool
}

type InstallResultMsg struct {
	Err error
}

type Service struct {
	exec system.Executor
}

func NewService(exec system.Executor) *Service {
	return &Service{
		exec: exec,
	}
}

var nvidiaPackages = []string{
	// Core drivers
	"nvidia-dkms",
	"nvidia-utils",
	"lib32-nvidia-utils",

	// Vulkan support for gaming (Proton)
	"vulkan-icd-loader",
	"lib32-vulkan-icd-loader",

	// GUI for driver settings
	"nvidia-settings",

	// Hardware video acceleration
	"libva-nvidia-driver",
}

// HasNvidiaGpu checks for an NVIDIA GPU using the provided executor.
// It returns a simple boolean and an error, making it easy to test.
func (s *Service) HasNvidiaGpu() (bool, error) {
	lspciCmd := exec.Command("lspci")
	grepCmd := exec.Command("grep", "-i", "nvidia")

	err := s.exec.RunPiped(lspciCmd, grepCmd)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil // No GPU found, is not an application error.
		}

		return false, fmt.Errorf(
			"gpu check command failed unexpectedly: %w",
			err,
		)
	}

	return true, nil
}

// CheckGpuCmd runs a command to detect an NVIDIA GPU and returns a message.
func (s *Service) CheckGpuCmd() tea.Cmd {
	return func() tea.Msg {
		log.Println("nvidia: checking for nvidia gpu...")
		hasGpu, err := s.HasNvidiaGpu()
		if err != nil {
			log.Printf("nvidia: error checking for nvidia gpu: %v", err)
			return GpuCheckResultMsg{HasNvidiaGpu: false}
		}

		if hasGpu {
			log.Println("nvidia: nvidia gpu found.")
		} else {
			log.Println("nvidia: no nvidia gpu found.")
		}
		return GpuCheckResultMsg{HasNvidiaGpu: hasGpu}
	}
}

// BuildInstallCommand creates the *exec.Cmd for installing drivers.
// It returns the command object, making it easy to inspect in tests.
func (s *Service) BuildInstallCommand() *exec.Cmd {
	args := append(
		[]string{"pacman", "-S", "--noconfirm", "--needed"},
		nvidiaPackages...,
	)
	return exec.Command("sudo", args...)
}

func (s *Service) InstallDriversCmd() tea.Cmd {
	cmd := s.BuildInstallCommand()

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			log.Printf("nvidia: sudo pacman install failed: %v", err)
			return InstallResultMsg{
				Err: fmt.Errorf(
					"installation failed or was cancelled by the user",
				),
			}
		}
		log.Println("nvidia: nvidia driver installation successful.")
		return InstallResultMsg{Err: nil}
	})
}
