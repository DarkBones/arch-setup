package types

type Phase int

const (
	MenuPhase Phase = iota
	GithubAuthPhase
	DotfilesPhase
	NvidiaDriversPhase
	ProfilesPhase
	DonePhase
)

// A message sent when a menu item is selected.
type MenuItemSelected struct {
	Phase Phase
}

type PhaseFinished struct {
	Phase Phase
}

type PhaseCancelled struct {
	Phase Phase
}

type PhaseBack struct{}
