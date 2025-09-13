# Bootstrap Any System (BAS)

**BAS** is a terminal UI (TUI) that helps you **speedrun** a fresh OS into your familiar, working environment.

It automates the boring bits:

1. Authenticate to GitHub via SSH (keygen + QR + one-press verify)
2. Clone your dotfiles repository
3. Stow configs and pick a machine **profile**
4. Install packages (Arch: `pacman` + `yay`; macOS: Homebrew)
5. Optionally run a post-install (e.g., Ansible bootstrap)

You can bundle BAS into a custom Arch ISO (auto-runs on first login), or install it normally and launch it any time.

---

## 🚀 Quickstart

### Arch Linux (AUR)

```bash
# with yay
yay -S bas-tui

# or manually
git clone https://aur.archlinux.org/bas-tui.git
cd bas-tui && makepkg -si
````

Run it any time:

```bash
bas-tui
```

### macOS (build from source)

BAS supports macOS provisioning but isn’t on Homebrew yet. Build locally:

```bash
# Requires Go 1.20+ and Xcode Command Line Tools
git clone https://github.com/DarkBones/arch-setup
cd arch-setup
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bas-tui ./cmd/archsetup
sudo mv bas-tui /usr/local/bin/   # or anywhere on your PATH
bas-tui
```

---

## ✅ Support Matrix

| OS     | Package Manager | Status                       |
| ------ | --------------- | ---------------------------- |
| Arch   | pacman + yay    | Production-ready             |
| macOS  | Homebrew        | Supported, not battle-tested |
| Others | —               | Not supported (Yet)          |

**Requirements (Arch):** network access, `base-devel`, `git`, `stow` (BAS will install `yay` if needed).
**Requirements (macOS):** Go 1.20+, Xcode Command Line Tools, network access.

---

## 🧭 How BAS works

1. **GitHub SSH**
   BAS generates an **ed25519** key if needed, shows it and a QR code, and guides you to add it at [https://github.com/settings/keys](https://github.com/settings/keys).

2. **Dotfiles**
   Enter (or accept) your dotfiles repo (`username/repo`). BAS clones to your chosen destination.

3. **Profiles**
   BAS reads `bas_settings.toml` from your dotfiles repo and shows only the profiles matching your OS/distro.

4. **Install**

   * **Arch**: Ensures `yay` exists, then installs packages from your profile list(s).
   * **macOS**: Ensures Homebrew exists, then installs your packages.

5. **Post-install (optional)**
   If your profile includes a `post_install` command, BAS will offer to run it (e.g., your Ansible bootstrap).

---

## ⚙️ `bas_settings.toml` (in your dotfiles repo)

BAS looks for this file at the **root of your dotfiles**. It defines selectable profiles and their behavior.

### Minimal starter

```toml
# bas_settings.toml
[[profiles]]
name = "Arch Desktop"
description = "My daily Arch setup."
path = "system/package_lists/arch_desktop.txt"
os_family = "linux"
os_distro = "arch"
stow_dirs = ["git", "zsh", "nvim", "tmux"]

[profiles.post_install]
description = "Bootstrap with Ansible"
command = "./bootstrap.sh"
working_dir = "ansible"
```

### Rich example

```toml
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

### Field reference

| Key              | Type        | Required | Description                                                                        |
| ---------------- | ----------- | -------- | ---------------------------------------------------------------------------------- |
| `name`           | string      | ✅        | Display name in the TUI.                                                           |
| `description`    | string      | ✅        | Shown below the name.                                                              |
| `path`           | string      | ✅        | Relative path to a **package list** (one package per line, `#` comments allowed).  |
| `os_family`      | string      | ❕        | `"linux"` or `"darwin"`. If omitted, the profile shows on all OSes.                |
| `os_distro`      | string      | ❕        | For Linux, `"arch"` (others currently unsupported).                                |
| `stow_dirs`      | array\[str] | ❕        | Directories inside your dotfiles to `stow` into `$HOME`.                           |
| `roles`          | array\[str] | ❕        | Free-form tags. BAS exports `MACHINE_PROFILES="role1,role2"` to your post-install. |
| `post_install.*` | table       | ❕        | Optional scripted handoff (e.g., Ansible), executed in `working_dir`.              |

---

## 📦 Package lists

* Plain text, **one package per line**
* `#` for comments
* Arch lists can mix repo and AUR packages (BAS will install `yay`)

Example (`system/package_lists/main_arch_desktop.txt`):

```
# Core
git
stow
zsh

# Dev
go
python
neovim

# AUR
spotify
```

---

## 🔍 Troubleshooting

* **GitHub auth fails**
  Add the shown public key at [https://github.com/settings/keys](https://github.com/settings/keys). Re-run BAS and select re-validate.

* **Stow conflicts (files already exist)**
  Stow won’t overwrite. Resolve files in `$HOME` (backup/remove), then re-run.

* **AUR installs fail on Arch**
  Ensure `base-devel` is installed: `sudo pacman -S --needed base-devel`. BAS will install `yay` if missing.

* **Debug logs**
  Run with `DEBUG=1 bas-tui`. A `debug.log` is written in the working directory.

---

## 🧹 Uninstall

**Arch**

```bash
sudo pacman -Rns bas-tui bas-tui-debug
```

(Your dotfiles and stowed symlinks are not removed.)

**macOS**

```bash
sudo rm -f /usr/local/bin/bas-tui
```

---

## 🛳️ Releasing new versions (maintainers)

From the source repo:

```bash
./release.sh               # bump patch, tag, update AUR
./release.sh --bump minor  # or: major|minor|patch
./release.sh --version 1.2.3
./release.sh --stamp       # version = YYYY.MM.DD.HHMM
```

---

## ⚖️ License

MIT — see `LICENSE`.
