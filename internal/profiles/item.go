package profiles

type profileItem struct {
	Profile
}

func (p profileItem) Title() string       { return p.Profile.Name }
func (p profileItem) Description() string { return p.Profile.Description }
func (p profileItem) FilterValue() string { return p.Profile.Name }

func (p profileItem) IsEnabled() bool { return true }
