# Bootstrap Any System (BAS)

**BAS** is a terminal user interface (TUI) written in Go to *speedrun* your way from a fresh OS install to a fully set-up and familiar environment.

The workflow looks like this:

1. Authenticate with GitHub over SSH.
2. Clone your dotfiles repo.
3. Stow configs and pick a machine profile.
4. Install all packages (pacman + yay on Arch, brew on macOS).
5. Optionally run a post-install script (e.g., an Ansible provisioner).

The TUI is bundled into a custom Arch ISO, so it auto-runs on first login after a fresh install. You can re-launch it any time with:

```bash
bas-tui
```
---

## 🚀 Quickstart
1. Boot into the Arch ISO (bundled with BAS) or install using `yay -S bas-tui`.
2. On first login, BAS runs automatically.
3. Pick your dotfiles repo, select a profile, and watch it build.
4. Re-launch any time with `bas-tui`.

---

## ✅ What works today

* **Arch Linux**:
  * Package installs via `pacman` + `yay`.
  * Dotfiles management with `git clone` + `stow`.
  * Profile-based installs using `bas_settings.toml`.
  * Profile filtering works via `os_family = "darwin"`.

---

## ⚠️ What’s technically supported but untested

* **macOS**:

  * Uses Homebrew for package installs.
  * Post-install hooks (like Ansible bootstrap) work the same way.
  * Not production-tested yet.

---

## ❌ Unsupported OS / Distros

If you run BAS on an unsupported platform, you’ll need to extend support yourself:

  * Add your distro detection in `system/os.go`.
  * Implement a package manager handler in `profiles.Service`.
  * Update your profiles TOML (`os_family` / `os_distro`) accordingly.

Typical levers:

  * **New distro** → Add logic to detect it and provide a package installer.
  * **New package manager** → Add a `CheckPkgMgrCmd` + `InstallPkgMgrCmd` implementation.
  * **Custom flow** → Use the `post_install` hook in a profile to hand off to your provisioner.

---

## ⚙️ Settings File

BAS looks for a configuration file named **`bas_settings.toml`** in the root of your dotfiles repository.

This file defines the machine **profiles** you can pick during setup.

### Example

```toml
# bas_settings.toml

[[profiles]]
name = "Hyprland Desktop"
description = "Personal Hyprland gaming and dev setup."
path = "system/package_lists/main_arch_desktop.txt"
os_family = "linux"
os_distro = "arch"
stow_dirs = ["fzf", "git", "hyprland", "tmux", "zsh", "waybar"]
roles = ["gaming", "streaming", "dev"]

[profiles.post_install]
description = "Run the main Ansible provisioner"
command = "./bootstrap.sh"
working_dir = "ansible"


[[profiles]]
name = "Headless Pi Server"
description = "Runs Home Assistant and Pi-hole."
path = "system/package_lists/pi_server.txt"
os_family = "linux"
os_distro = "arch"
stow_dirs = ["git", "nvim", "tmux", "zsh"]


[[profiles]]
name = "Mac Desktop"
description = "Personal mac setup."
path = "system/package_lists/mac_desktop.txt"
os_family = "darwin"
stow_dirs = ["wezterm", "nvim", "tmux", "zsh"]

[profiles.post_install]
description = "Run the main Ansible provisioner"
command = "./bootstrap.sh"
working_dir = "ansible"
```

---

## 🔧 Levers you can pull

* **Profiles**

  * Each `[[profiles]]` entry defines one selectable machine profile.
  * `os_family` and `os_distro` control whether the profile shows up on the current system.

* **Package lists**

  * Plain text files (one package per line, `#` for comments).
  * Path is relative to your dotfiles repo.

* **Dotfile stowing**

  * `stow_dirs` lists directories inside your dotfiles repo to stow into `$HOME` using GNU `stow`.

* **Roles**

  * Free-form tags that get passed into your post-install as `MACHINE_PROFILES` env var, which can be consumed by e.g. an Ansible playbook.

* **Post-install hook**

  * Optional. Lets you kick off Ansible or any other bootstrap script.
  * Runs in the given `working_dir` inside your dotfiles repo.
