package types

import "github.com/charmbracelet/bubbles/key"

type KeyMap struct {
	HardQuit key.Binding
	Quit     key.Binding
	Up       key.Binding
	Down     key.Binding
	Enter    key.Binding
	Back     key.Binding
	Tab      key.Binding
	ShiftTab key.Binding
}

func DefaultKeys() KeyMap {
	return KeyMap{
		HardQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc/backspace", "back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j", "s"),
			key.WithHelp("↓/j/s", "down"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k", "w"),
			key.WithHelp("↑/k/w", "up"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next"),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev"),
		),
	}
	// TODO: Map G and gg to go to bottom / top
}

// InputNavKeys returns a new KeyMap suitable for text input navigation.
// It keeps Tab/ShiftTab but filters character-based keys ('j', 'k') and
// 'backspace' from the Up, Down, and Back bindings.
func InputNavKeys(keys KeyMap) KeyMap {
	// Start with a copy of the original
	navKeys := keys

	// Rebind Up, Down, and Back to be unambiguous for text inputs
	navKeys.Up = key.NewBinding(
		key.WithKeys("up"),
		key.WithHelp("↑", "up"),
	)
	navKeys.Down = key.NewBinding(
		key.WithKeys("down"),
		key.WithHelp("↓", "down"),
	)
	navKeys.Back = key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	)

	return navKeys
}
