package utils

import (
	"archsetup/internal/constants"
	"archsetup/internal/styles"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ItemDelegate struct{}

func (d ItemDelegate) Height() int                               { return 2 }
func (d ItemDelegate) Spacing() int                              { return 1 }
func (d ItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d ItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(ListItem)
	if !ok {
		fmt.Fprint(w, "Invalid item type")
		return
	}

	var title, desc string
	isSelected := index == m.Index()
	enabledPrefix := constants.Menu.SelectedEnabledPrefix
	descSpacer := strings.Repeat(" ", len(enabledPrefix))

	if isSelected {
		if i.IsEnabled() {
			title = styles.TitleStyle.Render(enabledPrefix + i.Title())
			desc = styles.SubtleTextStyle.Render(descSpacer + i.Description())
		} else {
			title = styles.SubtleTextStyle.Render(enabledPrefix + i.Title())
			desc = styles.SubtleTextStyle.Render(descSpacer + i.Description())
		}
	} else {
		if i.IsEnabled() {
			title = styles.NormalTextStyle.Render(descSpacer + i.Title())
			desc = styles.SubtleTextStyle.Render(descSpacer + i.Description())
		} else {
			title = styles.SubtleTextStyle.Render(descSpacer + i.Title())
			desc = styles.SubtleTextStyle.Render(descSpacer + i.Description())
		}
	}

	fmt.Fprintf(w, "%s\n%s", title, desc)
}

func CalculateListHeight(list list.Model) int {
	const listVerticalFudgeFactor = 3

	titleHeight := lipgloss.Height(list.Styles.Title.Render(list.Title))
	helpHeight := lipgloss.Height(list.Help.View(list))

	d := ItemDelegate{} // Use our new shared delegate
	numItems := len(list.Items())
	itemsHeight := numItems*d.Height() + (numItems-1)*d.Spacing()

	return titleHeight + itemsHeight + helpHeight + listVerticalFudgeFactor
}
