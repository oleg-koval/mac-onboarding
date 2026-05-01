# mac-onboarding

Fast, privacy-first macOS configuration bootstrapper for MDM-managed Macs. Export settings from your source Mac, install on new machines—no Time Machine, no iCloud sync.

## What It Does

`mac-onboarding` captures your macOS apps, shell configs, system settings, hotkeys, and app preferences from a source Mac, then replays them on a fresh target Mac. One script, one command.

**Typical flow:**
```bash
# On source Mac
mac-onboarding export ~/onboard.tar.gz

# Transfer onboard.tar.gz to target Mac, then:
mac-onboarding install ~/onboard.tar.gz
```

## Privacy & Security

**By design:**
- **No cloud sync.** Everything stays local or goes over SSH (Tailscale).
- **Secrets redacted.** Shell rc files, git credentials, API keys are filtered before archiving.
- **Auditable.** See exactly what gets captured—no hidden backups.
- **MDM-aware.** Won't overwrite enrollment settings or protected system defaults.
- **Offline.** Works without internet after initial config.

**What's NOT captured:**
- SSH private keys (you add those manually)
- Password manager data (1Password has a guide)
- Cloud credentials (prompted during install)
- Large cache/logs

**Before open-sourcing:**
- Review `onboard.yaml` for any API keys you missed
- Test credential redaction patterns
- Audit module skip flags
- Check exported archive contents

## Quick Start

### Install

**Via Homebrew:**
```bash
brew tap oleg-koval/homebrew-tap
brew install mac-onboarding
```

**From GitHub Releases:**
```bash
# Intel
curl -Lo mac-onboarding https://github.com/oleg-koval/mac-onboarding/releases/download/v0.1.0/mac-onboarding-darwin-amd64

# Apple Silicon
curl -Lo mac-onboarding https://github.com/oleg-koval/mac-onboarding/releases/download/v0.1.0/mac-onboarding-darwin-arm64

chmod +x mac-onboarding
sudo mv mac-onboarding /usr/local/bin/
```

**Or build from source:**
```bash
git clone https://github.com/oleg-koval/mac-onboarding.git
cd mac-onboarding
make build
./dist/mac-onboarding --help
```

Homebrew-managed installs self-check for updates on each run and apply them automatically before executing the command. Set `MAC_ONBOARDING_AUTOUPDATE=0` to disable that behavior for a shell session or environment.

### Export (Source Mac)

```bash
# Dry-run first
mac-onboarding export --dry-run ~/onboard.tar.gz

# Verify redaction worked
mac-onboarding export ~/onboard.tar.gz
# → Captures 21 modules: bootstrap, brew, shell, git, system, hotkeys, ...
```

Copy `~/onboard.tar.gz` to target Mac (USB, iCloud, scp, whatever).

### Install (Target Mac)

```bash
# Dry-run first  
mac-onboarding install --dry-run ~/onboard.tar.gz

# Apply
mac-onboarding install ~/onboard.tar.gz
# → Installs Xcode CLT, Homebrew, apps, dotfiles, settings, hotkeys
```

### Bridge Mode (Live Pull)

Skip the archive—pull directly from source Mac via Tailscale SSH:

```bash
# On target Mac (requires source Mac's hostname in onboard.yaml)
# First, verify source.host in onboard.yaml points to your source Mac's Tailscale hostname
cat onboard.yaml | grep -A2 "^source:"

# Dry-run
mac-onboarding bridge pull --dry-run

# Apply
mac-onboarding bridge pull --only brew,shell  # or run all modules
```

**Requirements:**
- `source.host` set to source Mac's Tailscale hostname in `onboard.yaml`
- Tailscale running on both Macs
- Same username on both Macs (uses `ssh user@hostname`)
- Source Mac reachable via Tailscale SSH

**How it works:**
1. Target Mac SSHes to source Mac via Tailscale
2. Runs `mac-onboarding export --to-stdout`
3. Pipes archive to local install (no intermediate file)
4. Much faster for single modules: `bridge pull --only kitty,shell`

## Supported Modules (21 Total)

| Module | Exports | Install Behavior |
|--------|---------|------------------|
| **bootstrap** | — | Installs Xcode CLT, Homebrew, detects MDM |
| **brew** | `~/.Brewfile` | `brew bundle install` |
| **shell** | `.zshrc`, `.bashrc`, `.zprofile`, `.p10k.zsh`, oh-my-zsh custom | Redacts secrets, restores rc files |
| **git** | `.gitconfig`, `.gitignore_global`, `.config/git/` | Redacts credential helpers |
| **system** | macOS defaults: Dock, Finder, keyboard repeat, trackpad, screenshots | Filtered by allowlist (MDM-safe) |
| **hotkeys** | `com.apple.symbolichotkeys.plist` | Restarts pbs daemon |
| **kitty** | `~/.config/kitty/` | Full restore |
| **cursor** | Settings, keybindings, snippets, extensions | Installs extensions via CLI |
| **claude** | `~/.claude/` config | Full restore |
| **codex** | `~/.codex/config.toml`, agents, rules, prompts | Excludes sqlite/logs (ephemeral) |
| **pi** | `~/.pi/` config | Full restore |
| **swiftbar** | Plugin dir | Full restore |
| **alfred** | Alfred sync folder | Full restore (⚠️ audit for credentials) |
| **klack** | Settings plist | Full restore |
| **flux** | Settings plist | Full restore |
| **betterdisplay** | Settings plist | Full restore |
| **orbstack** | Settings plist | Full restore |
| **tailscale** | Settings plist | Full restore |
| **shottr** | Settings plist | Full restore |
| **synology** | NAS hostname reference only | Prompts for credentials on install |
| **onepassword** | — | Prints setup guide only |

## Configuration

Copy `onboard.yaml.example` to `onboard.yaml` and edit:

```yaml
source:
  # Tailscale hostname for bridge mode (bridge pull only)
  host: your-mac-hostname

modules:
  bootstrap:
    skip: false

  brew:
    skip: false
    options:
      brewfile_path: ~/.Brewfile

  shell:
    skip: false
    options:
      # Regex to redact from rc files
      redact_pattern: "export .*(KEY|TOKEN|SECRET|PASSWORD|API)=.*"

  git:
    skip: false

  system:
    skip: false

  # ... (see onboard.yaml.example for all modules)
```

**Path expansion:** All paths support `~` and `$HOME`.

**Per-module control:** Set `skip: true` to exclude any module.

**Dry-run always:** Test with `--dry-run` before committing to changes.

## Usage

### Export

```bash
# Standard export to archive
mac-onboarding export [flags] ARCHIVE_PATH

Flags:
  --config string       Config file (default: ./onboard.yaml)
  --dry-run             Show what would happen
  --only strings        Run only these modules (comma-separated)
  --verbose             Verbose output
```

Example:
```bash
mac-onboarding export --only brew,shell ~/quick.tar.gz
```

### Install

```bash
# Restore from archive
mac-onboarding install [flags] ARCHIVE_PATH

Flags:
  --config string       Config file
  --dry-run             Show what would happen
  --only strings        Run only these modules
  --verbose             Verbose output
```

### Bridge (Live Pull)

```bash
# Pull from source Mac via Tailscale SSH (no archive)
mac-onboarding bridge pull [flags]

Flags:
  --config string       Config file (required, must have source.host)
  --dry-run             Show what would happen
  --only strings        Run only these modules
  --verbose             Verbose output
```

**Requirements:**
- Both Macs on same Tailscale network
- Source Mac has `mac-onboarding` installed
- Target Mac has `onboard.yaml` with source Mac's Tailscale hostname

## Dry-Run First

Always test with `--dry-run`:

```bash
mac-onboarding export --dry-run ~/test.tar.gz
# Review output, then:
mac-onboarding export ~/test.tar.gz
```

## Security Considerations

1. **Archive contents:** `tar tzf onboard.tar.gz` to audit before transfer.
2. **Secrets redaction:** Verify your rc files, git config, AI tool configs don't leak API keys.
3. **Alfred workflows:** May contain credentials—review sync folder.
4. **MDM:** Won't override enrollment or restricted defaults.
5. **Tailscale SSH:** Requires Tailscale running; uses your account's SSH keys.

## Troubleshooting

| Issue | Solution |
|-------|----------|
| `config: no onboard.yaml found` | Copy `onboard.yaml.example` to `onboard.yaml` and edit source.host |
| Module skip not working | Check YAML indentation (spaces, not tabs) |
| Secrets leaked in archive | Review `shell.redact_pattern` in config |
| MDM won't let system defaults install | Check `system` module allowlist—some domains are protected |
| Bridge pull fails | Verify source Mac's Tailscale hostname in config; run `tailscale status` on source |
| Hotkeys not applied | May need System Preferences restart; pbs daemon auto-restarts |

## Building

```bash
make build           # Build mac-onboarding binary
make test            # Run tests
make lint            # Run go vet
make release         # Build darwin/amd64 and darwin/arm64 binaries
make clean           # Clean build artifacts
```

## CI/CD

- `test.yml` runs on every pull request and push to `main`: `go test`, `go vet`, `staticcheck`, `gofmt` check, and a build smoke test.
- `release.yml` runs on every push to `main`: it computes and pushes the next patch tag from the latest `v*` tag, publishes the macOS binaries to GitHub Releases, and updates `oleg-koval/homebrew-tap`.

Required secret for release automation:

- `HOMEBREW_TAP_GITHUB_TOKEN`: fine-grained GitHub token with `contents: write` access to `oleg-koval/homebrew-tap`.

## Contributing

Issues and PRs welcome. Before open-sourcing, audit for any hardcoded paths, credentials, or MDM-specific logic.

## License

MIT (coming soon)
