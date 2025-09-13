package styles

import "github.com/charmbracelet/lipgloss"

var (
	Pumpkin   = lipgloss.Color("208")
	Bone      = lipgloss.Color("230")
	BoneMuted = lipgloss.Color("250")
	Charcoal  = lipgloss.Color("234")
	Purple    = lipgloss.Color("98")
	Green     = lipgloss.Color("112")
)

// App frame & typographic system
var (
	AppStyle = lipgloss.NewStyle().
			Padding(1, 2)

	TitleStyle = lipgloss.NewStyle().
			Foreground(Pumpkin).
			Bold(true).
			Padding(0, 1)

	NormalTextStyle = lipgloss.NewStyle().
			Foreground(Bone)

	SubtleTextStyle = lipgloss.NewStyle().
			Foreground(BoneMuted)

	// Dividers, faint lines
	Divider = lipgloss.NewStyle().
		Foreground(BoneMuted)
)

// Borders & emphasis
var (
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Purple)

	FocusedBorderStyle = BorderStyle.
				BorderForeground(Pumpkin)

	BlurredBorderStyle = BorderStyle.
				BorderForeground(BoneMuted)

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(Pumpkin)

	KeyStyle = lipgloss.NewStyle().
			Foreground(Bone).
			Border(lipgloss.NormalBorder()).
			BorderForeground(Purple).
			Padding(0, 1)
)

// Semantic feedback
var (
	SuccessStyle = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Purple).
			Bold(true)
)

// CTA tokens
var (
	PrimaryButton = lipgloss.NewStyle().
			Background(Pumpkin).
			Foreground(Charcoal).
			Bold(true).
			Padding(0, 1)

	SecondaryButton = lipgloss.NewStyle().
			Background(Purple).
			Foreground(Bone).
			Bold(true).
			Padding(0, 1)

	BadgeSuccess = lipgloss.NewStyle().
			Background(Green).
			Foreground(Charcoal).
			Bold(true).
			Padding(0, 1)

	LinkStyle = lipgloss.NewStyle().
			Foreground(Pumpkin).
			Underline(true)
)
