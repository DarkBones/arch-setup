package main

import (
	"archsetup/internal/app"
	"archsetup/internal/assert"
	"archsetup/internal/dotfiles"
	"archsetup/internal/github_auth"
	"archsetup/internal/menu"
	"archsetup/internal/nvidia"
	"archsetup/internal/profiles"
	"archsetup/internal/system"
	"archsetup/internal/types"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	tea "github.com/charmbracelet/bubbletea"
)

type Application interface {
	Run() error
}

type TUIApp struct {
	program *tea.Program
}

type PanicCatchingModel struct {
	Model tea.Model
}

func (app *TUIApp) Run() error {
	_, err := app.program.Run()
	return err
}

func main() {
	privilegeChecker := SudoChecker{}

	keys := types.DefaultKeys()

	dotfilesSvc := dotfiles.NewService(
		&system.LiveExecutor{},
		&system.LiveFileSystem{},
	)

	nvidiaSvc := nvidia.NewService(
		&system.LiveExecutor{},
	)

	profilesSvc := profiles.NewService(
		&system.LiveExecutor{},
		&system.LiveFileSystem{},
	)

	home, err := os.UserHomeDir()
	if err != nil {
		log.Printf(
			"Could not get user home dir, falling back to empty: %v",
			err,
		)
		assert.Fail("Could not get user home dir")
	}
	defaultDotfilesPath := filepath.Join(home, "Developer", "dotfiles")

	githubAuthSvc := github_auth.NewDefaultService()
	models := map[types.Phase]tea.Model{
		types.MenuPhase:       menu.New(keys),
		types.GithubAuthPhase: github_auth.New(keys, githubAuthSvc),
		types.DotfilesPhase: dotfiles.New(
			keys,
			dotfilesSvc,
			defaultDotfilesPath,
		),
		types.NvidiaDriversPhase: nvidia.New(keys, nvidiaSvc),
		types.ProfilesPhase:      profiles.New(keys, profilesSvc),
	}

	appModel := app.New(types.MenuPhase, models, keys)

	wrappedModel := &PanicCatchingModel{Model: appModel}
	program := tea.NewProgram(wrappedModel, tea.WithAltScreen())
	tuiApp := &TUIApp{program: program}

	if err := run(os.Args, privilegeChecker, tuiApp); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, privilegeChecker PrivilegeChecker, app Application) (err error) {
	_, debugEnabled := os.LookupEnv("DEBUG")
	f, err := setupLogging(debugEnabled, logfileCreator)
	if err != nil {
		return err
	}
	if f != nil {
		defer f.Close()
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in main: %v\nStack trace:\n%s", r, debug.Stack())
			err = fmt.Errorf("application panicked: %v", r)
		}
	}()

	log.Println("booting...")

	fmt.Println("Checking administrative privileges...")
	if err := privilegeChecker.Check(); err != nil {
		log.Printf("sudo check failed: %v", err)
		return fmt.Errorf("could not obtain administrative privileges: %w", err)
	}

	if err := app.Run(); err != nil {
		log.Printf("Application error: %v", err)
		return fmt.Errorf("application error: %w", err)
	}

	println("Bye! To run this app again, run `bas-tui`")

	return nil
}

func (m *PanicCatchingModel) Init() tea.Cmd {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in Init: %v\nStack trace:\n%s", r, debug.Stack())
		}
	}()
	return m.Model.Init()
}

func (m *PanicCatchingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in Update: %v\nStack trace:\n%s", r, debug.Stack())
		}
	}()

	updatedModel, cmd := m.Model.Update(msg)
	m.Model = updatedModel

	return m, cmd
}

func (m *PanicCatchingModel) View() string {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in View: %v\nStack trace:\n%s", r, debug.Stack())
		}
	}()
	return m.Model.View()
}

func setupLogging(
	debugEnabled bool,
	createFile func() (*os.File, error),
) (*os.File, error) {
	if !debugEnabled {
		log.SetOutput(io.Discard)
		return nil, nil
	}

	return createFile()
}

func logfileCreator() (*os.File, error) {
	return tea.LogToFile("debug.log", "debug")
}
