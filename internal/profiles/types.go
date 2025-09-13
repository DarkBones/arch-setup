package profiles

type PostInstallCommand struct {
	Description string `toml:"description"`
	Command     string `toml:"command"`
	WorkingDir  string `toml:"working_dir"`
}

type Profile struct {
	Name        string              `toml:"name"`
	Description string              `toml:"description"`
	Path        string              `toml:"path"`
	OsFamily    string              `toml:"os_family"`
	OsDistro    string              `toml:"os_distro"`
	StowDirs    []string            `toml:"stow_dirs"`
	Roles       []string            `toml:"roles"`
	PostInstall *PostInstallCommand `toml:"post_install"`
}

type Config struct {
	Profiles []Profile `toml:"profiles"`
}
