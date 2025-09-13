package constants

type menuStrings struct {
	Title                 string
	SelectedEnabledPrefix string
	NvidiaMenuDesc        string
	NvidiaDisDesc         string
	GithubAuthed          string
	GithubUnAuthed        string
	GithubConnected       string
	GithubDisconnected    string
}

var Menu = &menuStrings{
	Title:                 App.Title,
	SelectedEnabledPrefix: "» ",
	NvidiaMenuDesc:        "Install proprietary drivers for your GPU",
	NvidiaDisDesc:         "No NVIDIA GPU detected",
	GithubAuthed:          "(Authenticated)",
	GithubUnAuthed:        "(Not Authenticated)",
	GithubConnected:       "(Connected)",
	GithubDisconnected:    "(Not connected)",
}
