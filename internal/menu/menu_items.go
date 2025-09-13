package menu

import (
	"archsetup/internal/types"
	"fmt"

	"github.com/charmbracelet/bubbles/list"
)

const GithubAuthDesc = "Set up SSH keys for GitHub. %s"
const DotfilesDesc = "Clone and set up your dotfiles. %s"

func GetMenuItems() []list.Item {
	return []list.Item{
		MenuItem{item{
			Phase:       types.GithubAuthPhase,
			Title:       "Github Authentication",
			Description: fmt.Sprintf(GithubAuthDesc, "Checking..."),
			Enabled:     true,
		}},
		MenuItem{item{
			Phase:       types.DotfilesPhase,
			Title:       "Dotfiles Setup",
			Description: fmt.Sprintf(DotfilesDesc, " (Not connected)"),
			Enabled:     false,
		}},
		// MenuItem{item{
		// 	Phase:       types.NvidiaDriversPhase,
		// 	Title:       "Nvidia Drivers",
		// 	Description: "Install drivers for your Nvidia GPU",
		// 	Enabled:     false,
		// }},
		MenuItem{item{
			Phase:       types.ProfilesPhase,
			Title:       "Machine Profile",
			Description: "Pick a machine profile to match your setup",
			Enabled:     false,
		}},
	}
}
