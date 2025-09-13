package utils

import "github.com/charmbracelet/bubbles/list"

type ListItem interface {
	list.Item
	Title() string
	Description() string
	IsEnabled() bool
}
