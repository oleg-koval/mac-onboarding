---
name: mac-onboarding
description: Export macOS app configs and settings from source Mac, install on target Mac without Time Machine or cloud sync
tags: [macos, dotfiles, configuration, bootstrap, mdm-safe, tailscale]
---

# mac-onboarding

Fast, privacy-first macOS configuration exporter and installer. One script, one command—migrate your entire Mac environment to a new machine without Time Machine, iCloud, or cloud backups.

## When to Use

- **Setting up a new Mac** — restore apps, dotfiles, settings, hotkeys from your old Mac
- **MDM-managed Macs** — works safely on corporate machines (respects enrollment and protected defaults)
- **Privacy-sensitive environments** — all secrets redacted, nothing to cloud
- **Bridging Macs** — pull configuration live from source Mac via Tailscale SSH (no archive needed)
- **Reproducible setups** — version your onboard.yaml in git, repeat fresh installs identically

**NOT for:** Migrating user data (photos, documents, mail), password managers (separate process per app), SSH keys (add manually), Time Machine replacement.

## What It Captures

| Category | What's Exported | Install Behavior |
|----------|-----------------|------------------|
| **Package managers** | Homebrew (via `brew bundle dump`) | `brew bundle install` |
| **Shell** | zsh/bash rc files, oh-my-zsh custom | Secrets redacted, configs restored |
| **Git** | `.gitconfig`, `.gitignore_global`, `.config/git/` | Credential helpers redacted |
| **Code editors** | Cursor, Claude Code, Codex, pi.dev | Settings, keybindings, extensions, AI tokens redacted |
| **Terminal** | Kitty full config | Full restore |
| **System** | Dock, Finder, keyboard repeat, trackpad, screenshot location | MDM-safe allowlist |
| **Hotkeys** | Global shortcuts via `com.apple.symbolichotkeys` | Full restore, pbs daemon restarted |
| **Apps** | SwiftBar, Alfred, Klack, f.lux, BetterDisplay, OrbStack, Tailscale, Shottr, Synology | Plist restore (⚠️ audit Alfred for credentials) |
| **Cloud tools** | 1Password | Setup guide only (security) |

## Quick Start

### 1. Export (Source Mac)

```bash
# Install (if not using Homebrew yet)
git clone https://github.com/oleg-koval/mac-onboarding.git
cd mac-onboarding
make build
sudo mv dist/mac-onboarding /usr/local/bin/

# Copy and edit config
cp onboard.yaml.example onboard.yaml
nano onboard.yaml  # set source.host for bridge mode (optional)

# Dry-run
mac-onboarding export --dry-run ~/onboard.tar.gz

# Export
mac-onboarding export ~/onboard.tar.gz
# → Captures 21 modules: bootstrap, brew, shell, git, system, hotkeys, ...
```

### 2. Install (Target Mac)

```bash
# Copy config if using bridge mode
scp onboard.yaml target-mac:~/onboard.yaml

# Get archive and install
scp source-mac:~/onboard.tar.gz ~/

# Dry-run
mac-onboarding install --dry-run ~/onboard.tar.gz

# Apply
mac-onboarding install ~/onboard.tar.gz
# → Installs Xcode CLT, Homebrew, apps, dotfiles, settings, hotkeys
```

### 3. Bridge Mode (Live Pull — No Archive)

```bash
# On target Mac, update source.host in onboard.yaml
echo "source:" > onboard.yaml
echo "  host: source-mac.tailscale.com" >> onboard.yaml

# Pull live from source Mac via Tailscale SSH
mac-onboarding bridge pull --dry-run
mac-onboarding bridge pull --only brew,shell
mac-onboarding bridge pull  # all modules
```

## Configuration

`onboard.yaml` (copied from `onboard.yaml.example`):

```yaml
source:
  host: your-mac-hostname  # Tailscale hostname (bridge mode only)

modules:
  bootstrap:
    skip: false            # Installs Xcode CLT, Homebrew

  brew:
    skip: false
    options:
      brewfile_path: ~/.Brewfile

  shell:
    skip: false
    options:
      redact_pattern: "export .*(KEY|TOKEN|SECRET|PASSWORD|API)=.*"

  git:
    skip: false

  system:
    skip: false            # Dock, Finder, keyboard defaults (MDM-safe)

  hotkeys:
    skip: false

  # ... 15 more modules (see onboard.yaml.example)
```

**All paths support `~` and `$HOME` expansion.**

## Usage Patterns

### Pattern 1: Archive Export/Install

Typical for USB transfer, email, cloud storage:

```bash
# Source
mac-onboarding export ~/onboard-$(date +%Y%m%d).tar.gz

# Transfer archive to target (USB, iCloud, S3, whatever)

# Target
mac-onboarding install ~/onboard.tar.gz
```

### Pattern 2: SSH Piping (No Archive)

Direct streaming via SSH:

```bash
# Source and target on same network
ssh user@source-mac "mac-onboarding export --to-stdout" | \
  mac-onboarding install --from-stdin
```

### Pattern 3: Bridge Mode (Live Pull)

Fastest for single modules:

```bash
# Target Mac
mac-onboarding bridge pull --only brew,shell

# Or everything
mac-onboarding bridge pull
```

**Requires:**
- `source.host` in `onboard.yaml`
- Tailscale running on both Macs
- Same username on both Macs

### Pattern 4: Selective Modules

Skip unwanted modules:

```yaml
modules:
  brew:
    skip: true   # Don't export/install Homebrew
  onepassword:
    skip: true   # 1Password needs manual setup
```

Or CLI:

```bash
mac-onboarding export --only shell,git ~/minimal.tar.gz
mac-onboarding install --only shell,git ~/minimal.tar.gz
mac-onboarding bridge pull --only kitty,cursor
```

## Dry-Run First

**Always test before applying:**

```bash
mac-onboarding export --dry-run ~/test.tar.gz
# Review output, then:
mac-onboarding export ~/test.tar.gz

mac-onboarding install --dry-run ~/test.tar.gz
# Review output, then:
mac-onboarding install ~/test.tar.gz
```

## Security

**By design:**
- Secrets redacted before archiving (shell env vars, git credentials, AI tokens)
- No cloud sync—everything stays local or over SSH
- MDM-aware—won't override enrollment or protected defaults
- Offline after initial config—no internet required

**Before open-sourcing:**
- Review `onboard.yaml` for any API keys you missed
- Audit exported archive: `tar tzf onboard.tar.gz`
- Test credential redaction: `grep -i token onboard.tar.gz` should show no real values
- Check Alfred sync folder for embedded credentials

**What's NOT captured:**
- SSH private keys (add to ~/.ssh manually)
- Password manager data (1Password, Bitwarden handled per app)
- Cloud credentials (prompted during install)
- Large caches, logs, sessions

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `config: no onboard.yaml found` | Copy `onboard.yaml.example` to `onboard.yaml` and edit |
| Module skip not working | Check YAML indentation (spaces not tabs) |
| Secrets leaked in archive | Review `shell.redact_pattern` and test: `tar xzOf onboard.tar.gz shell/.zshrc \| grep -i token` |
| MDM won't let system defaults install | Some domains (Dock, Finder) are protected—check logs for which ones failed |
| Bridge pull fails | Verify source Mac's Tailscale hostname in config; run `tailscale status` on source |
| Hotkeys not applied | Restart System Preferences or restart Mac (pbs daemon is auto-restarted) |
| Homebrew install fails | Ensure Xcode CLT is installed first (bootstrap module handles this) |

## Requirements

- **macOS 11+** (Big Sur or later)
- **Tailscale** (for bridge mode, optional otherwise)
- **Homebrew** (bootstrap module installs it)
- **Git** (to clone repo; `make build` requires Go)

## Building Locally

```bash
make build         # Build mac-onboarding binary in dist/
make test          # Run tests
make lint          # Run go vet
make release       # Build darwin/amd64 and darwin/arm64
make install       # Install to /usr/local/bin
make clean         # Clean dist/
```

## CI/CD

Automated tests and releases via GitHub Actions:

- **test.yml** — runs on every push/PR: go test, go vet, staticcheck, fmt check
- **release.yml** — runs on every push to `main`: increments the latest patch version, builds Intel + Apple Silicon binaries, creates a GitHub release with checksums, and updates the Homebrew tap formula

**Release prerequisites:**

```bash
# GitHub repo secret required:
# HOMEBREW_TAP_GITHUB_TOKEN
# -> Fine-grained token with contents:write on oleg-koval/homebrew-tap
```

## Examples

### Scenario: Fresh Mac Setup

```bash
# Old Mac (source)
mac-onboarding export ~/my-setup.tar.gz

# Move USB to new Mac
scp /Volumes/USB/my-setup.tar.gz ~

# New Mac (target)
mac-onboarding install ~/my-setup.tar.gz
# → 5 minutes later: Xcode CLT, Homebrew, apps, dotfiles, settings all restored
```

### Scenario: Team Standardized Setup

```bash
# Shared repo with onboard.yaml checked in
git clone https://github.com/company/mac-setup.git
cd mac-setup

# Each engineer
mac-onboarding export ~/standard.tar.gz

# Archive reviewed and committed to repo
tar tzf ~/standard.tar.gz | head -20

# New team member
git clone https://github.com/company/mac-setup.git
cd mac-setup
mac-onboarding install ~/standard.tar.gz
# → Identical environment as everyone else
```

### Scenario: Minimal Dotfiles Only

```bash
# Source
mac-onboarding export --only shell,git ~/dotfiles.tar.gz

# Target
mac-onboarding install --only shell,git ~/dotfiles.tar.gz
# → Just rc files and git config, no apps or settings
```

### Scenario: Bridge Pull from Remote Mac

```bash
# Target Mac's onboard.yaml
source:
  host: old-mac.tailscale.com

# Pull specific modules live
mac-onboarding bridge pull --only kitty,cursor
# → No archive, direct SSH stream from old-mac
```

## Privacy Checklist

Before sharing onboard.yaml or archive:

- [ ] Review `onboard.yaml` for any hardcoded secrets
- [ ] Test archive: `tar tzf onboard.tar.gz | grep -E '(token|key|secret|password)'` (should be empty or redacted)
- [ ] Audit `shell/` files for env vars
- [ ] Audit `git/` for credential helpers
- [ ] Check `ai_tools/` for API tokens (should be redacted)
- [ ] Test on fresh Mac to ensure all modules install cleanly
- [ ] If using bridge mode, verify source.host doesn't leak hostname

## License

MIT

## Contributing

Issues and PRs welcome. Before contributing, read CONTRIBUTING.md (coming soon).

## Contact

[GitHub Issues](https://github.com/oleg-koval/mac-onboarding/issues)
