package menu

import "archsetup/internal/types"

type item struct {
	Phase       types.Phase
	Title       string
	Description string
	Enabled     bool
	Done        bool
}

// MenuItem wraps the item data and satisfies the List.Item interface.
type MenuItem struct {
	item
}

func (i MenuItem) FilterValue() string { return i.item.Title }

func (i MenuItem) Title() string       { return i.item.Title }
func (i MenuItem) Description() string { return i.item.Description }
func (i MenuItem) IsEnabled() bool     { return i.item.Enabled }
