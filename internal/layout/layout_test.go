package layout

import (
	"archsetup/internal/styles"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestView(t *testing.T) {
	originalStyle := styles.AppStyle
	styles.AppStyle = lipgloss.NewStyle().Padding(1, 2)
	defer func() { styles.AppStyle = originalStyle }()

	t.Run("Content fits and is centered", func(t *testing.T) {
		content := "Hello, World!"
		width := 20
		height := 5

		output := View(content, width, height)
		lines := strings.Split(output, "\n")

		// 1. Check total line count
		if len(lines) != height {
			t.Fatalf("Expected output to have %d lines, but got %d", height, len(lines))
		}

		// 2. Check that the content is on the middle line
		middleLineIndex := height / 2
		contentLine := lines[middleLineIndex]
		if !strings.Contains(contentLine, content) {
			t.Fatalf("Content '%s' not found on the middle line (%d).\nGot:\n%s", content, middleLineIndex+1, output)
		}

		// 3. Check that the content is horizontally centered by trimming whitespace
		if strings.TrimSpace(contentLine) != content {
			t.Errorf("Expected trimmed content line to be '%s', but got '%s'", content, strings.TrimSpace(contentLine))
		}
	})

	t.Run("Content is truncated when too tall", func(t *testing.T) {
		content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
		width := 20
		height := 5 // availableHeight = 5 - 2 = 3

		output := View(content, width, height)

		if !strings.Contains(output, "Line 1") {
			t.Error("Output is missing 'Line 1'")
		}
		if !strings.Contains(output, "Line 3") {
			t.Error("Output is missing 'Line 3'")
		}
		if strings.Contains(output, "Line 4") {
			t.Error("Output should have truncated 'Line 4', but it is present")
		}
	})

	t.Run("Returns empty string for zero or negative available height", func(t *testing.T) {
		content := "Hello"
		width := 10
		height := 2 // availableHeight = 2 - 2 = 0

		output := View(content, width, height)

		if output != "" {
			t.Errorf("Expected an empty string for zero available height, but got '%s'", output)
		}
	})
}
