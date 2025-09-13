package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Main application frame
	AppStyle = lipgloss.NewStyle().Padding(1, 2)

	// Titles and Headers
	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")). // Magenta
			Bold(true).
			Padding(0, 1)

	// Normal text
	NormalTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

	// Dimmed/subtle text for help, descriptions etc.
	SubtleTextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Success and Error messages
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)  // Green
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true) // Red

	// Borders and Boxes
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")) // Purple

	// Styles for input boxes
	FocusedBorderStyle = BorderStyle.BorderForeground(lipgloss.Color("205")) // Magenta border
	BlurredBorderStyle = BorderStyle

	// Special styles for components
	SpinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")) // Magenta
	KeyStyle     = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1)
)
