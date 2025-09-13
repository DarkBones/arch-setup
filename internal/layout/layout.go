package layout

import (
	"archsetup/internal/styles"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func View(content string, width, height int) string {
	availableWidth := width - styles.AppStyle.GetHorizontalPadding()
	availableHeight := height - styles.AppStyle.GetVerticalPadding()
	if availableHeight <= 0 {
		return ""
	}

	var finalContent string

	contentHeight := lipgloss.Height(content)

	if contentHeight > availableHeight {
		lines := strings.Split(content, "\n")
		lines = lines[:availableHeight]
		finalContent = strings.Join(lines, "\n")
	} else {
		finalContent = lipgloss.PlaceVertical(
			availableHeight,
			lipgloss.Center,
			content,
		)
	}

	centeredResult := lipgloss.PlaceHorizontal(
		availableWidth,
		lipgloss.Center,
		finalContent,
	)

	return styles.AppStyle.Render(centeredResult)
}
