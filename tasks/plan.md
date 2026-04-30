# Implementation Plan: mac-onboarding

## Overview

A Go CLI tool that exports app configs + settings from a source Mac, then installs + restores
them on a new MDM-managed Mac — without Time Machine. One command on each end.
Ships as a single static binary. Open-source safe: no secrets committed, all sensitive paths
configurable via a gitignored `onboard.yaml`.

A companion **bridge** mode lets the target machine pull any app's config live from the source
machine over Tailscale (which is already in the install list), so you can add apps ad-hoc
after initial setup.

## Architecture Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | **Go** | Single static binary, no runtime deps on fresh Mac, fast startup, solid SSH/net libs, simpler than Rust for this use case |
| Config format | YAML (`onboard.yaml`) | Human-readable, gitignored for private sections; example ships with repo |
| Transport (bridge) | SSH over Tailscale | Tailscale already required; no extra infra; E2E encrypted |
| Secrets policy | Never stored | SSH keys, vault passwords, tokens → excluded by module contract; user moves them manually |
| MDM awareness | Read-only probes | Detect MDM with `profiles status`; skip or warn on restricted paths |
| Dry-run | Always available | `--dry-run` on all destructive commands; default for first run |
| Privacy (open-source) | Configurable paths | No hardcoded usernames/emails; example config uses placeholders |

## Dependency Graph

```
onboard.yaml (config)
    │
    ├── bootstrap module (Homebrew + Xcode CLT)
    │       │
    │       ├── brew module (brew bundle)
    │       │       │
    │       │       └── all GUI apps via Brewfile
    │       │
    │       └── shell module (zsh + rc files)
    │               │
    │               └── all other modules (tools available in PATH)
    │
    ├── app modules (independent after bootstrap)
    │   ├── kitty
    │   ├── cursor
    │   ├── claude-code
    │   ├── codex / pi.dev
    │   ├── skillshare-cli + plugins
    │   ├── swiftbar + plugins
    │   ├── alfred
    │   ├── 1password (guide only — no automation)
    │   ├── orbstack
    │   ├── klack
    │   ├── f.lux
    │   ├── betterdisplay
    │   ├── tailscale
    │   └── shottr
    │
    ├── system module (macOS defaults write)
    │       └── hotkeys module
    │
    ├── git module (~/.gitconfig, ~/.gitignore_global)
    │
    └── bridge module (export server / import client)
            └── depends on: tailscale module complete
```

## Phases

### Phase 1: Scaffold + Bootstrap (Foundation)
Tasks 1–3. Gets Go project, CLI skeleton, and Homebrew working. Validates the
compile-to-binary approach and proves MDM detection works before touching anything sensitive.

### Phase 2: Core App Modules (Vertical slices)
Tasks 4–10. One module per app category. Each module: export (source Mac) + install (target Mac).
Every module works standalone — can run `mac-onboarding install --only kitty`.

### Phase 3: System + Dotfiles
Tasks 11–13. macOS defaults, hotkeys, git, shell. These touch system-wide state so isolated
from app modules.

### Phase 4: Bridge
Task 14. SSH server/client for live pull of any module's artifacts over Tailscale.

### Phase 5: Polish + Open-source prep
Tasks 15–16. README, example config, secret scrub, CI.

---

## Task List

### Phase 1: Foundation

#### Task 1: Go project scaffold + CLI skeleton

**Description:** Init Go module `github.com/oleg-koval/mac-onboarding`, wire cobra CLI with
`export`, `install`, `bridge` subcommands, `--dry-run` / `--config` / `--only` flags.
Compile to a single binary via `make build`. Add `install.sh` bootstrap (installs Go if absent,
clones repo, builds binary, moves to `/usr/local/bin/mac-onboarding`).

**Acceptance criteria:**
- [ ] `go build ./...` succeeds, binary < 20 MB
- [ ] `mac-onboarding --help` lists all subcommands
- [ ] `mac-onboarding export --dry-run` exits 0 with "dry-run: no changes made"
- [ ] `install.sh` installs binary from scratch on a fresh shell

**Verification:**
- [ ] `go test ./...` passes
- [ ] `make build` produces `dist/mac-onboarding`

**Dependencies:** None

**Files:** `main.go`, `cmd/root.go`, `cmd/export.go`, `cmd/install.go`, `cmd/bridge.go`,
`internal/config/config.go`, `Makefile`, `install.sh`, `go.mod`

**Scope:** M

---

#### Task 2: Config loader + `onboard.yaml` schema

**Description:** Define the YAML schema that controls which modules run, where dotfiles live
on the source machine, and target paths on the destination. Ship `onboard.yaml.example` with
placeholders. Gitignore `onboard.yaml`. Add validation with clear errors.

**Acceptance criteria:**
- [ ] `onboard.yaml.example` contains every configurable field with comments
- [ ] Missing required fields produce a clear error (not a panic)
- [ ] `~` and `$HOME` expand correctly in all path fields
- [ ] Unknown fields in yaml produce a warning, not a fatal

**Verification:**
- [ ] Unit test: parse example file → no errors
- [ ] Unit test: missing `source_host` → error message mentions field name

**Dependencies:** Task 1

**Files:** `internal/config/config.go`, `internal/config/config_test.go`, `onboard.yaml.example`

**Scope:** S

---

#### Task 3: Bootstrap module (Xcode CLT + Homebrew)

**Description:** The first thing `mac-onboarding install` runs. Detects if Xcode CLT and
Homebrew are installed; installs them if not. Detects MDM via `profiles status -type enrollment`
and logs a warning if managed (some restrictions may apply). All side effects guarded by `--dry-run`.

**Acceptance criteria:**
- [ ] Idempotent: running twice does nothing on a machine that already has both
- [ ] MDM detection prints a warning with list of known restricted operations
- [ ] `--dry-run` prints what would be installed without doing it
- [ ] CI (GitHub Actions macOS runner) passes this task

**Verification:**
- [ ] Integration test: run on macOS with Homebrew absent → installs it
- [ ] `profiles status` output parsed correctly for enrolled vs not-enrolled

**Dependencies:** Task 2

**Files:** `internal/modules/bootstrap/bootstrap.go`, `internal/mdm/mdm.go`

**Scope:** M

---

### Checkpoint: Phase 1

- [ ] `go test ./...` passes
- [ ] `make build` produces binary
- [ ] `install.sh` runs end-to-end on a fresh shell (no Go, no brew)
- [ ] Human reviews CLI UX and config schema before Phase 2

---

### Phase 2: Core App Modules

Each module implements the `Module` interface:

```go
type Module interface {
    Name() string
    Export(cfg Config, dst string) error   // source machine: pack artifacts
    Install(cfg Config, src string) error  // target machine: unpack + configure
}
```

---

#### Task 4: Homebrew module (Brewfile export + install)

**Description:** Export: `brew bundle dump --force --file=Brewfile` into the export archive.
Install: `brew bundle install --file=Brewfile`. Separate taps, brews, casks, mas (Mac App Store)
into sections. MDM machines may block some casks — detect and skip with a warning.

**Acceptance criteria:**
- [ ] Export produces a valid `Brewfile` in the archive
- [ ] Install runs `brew bundle install` with `--no-lock` to avoid lockfile conflicts
- [ ] Failed individual casks are logged and skipped (don't abort the whole run)
- [ ] `--only brew` runs only this module

**Verification:**
- [ ] Export → Brewfile has non-zero entries on a real Mac
- [ ] Dry-run install prints each brew/cask that would be installed

**Dependencies:** Task 3

**Files:** `internal/modules/brew/brew.go`, `internal/modules/brew/brew_test.go`

**Scope:** S

---

#### Task 5: Shell module (zsh + rc files + oh-my-zsh)

**Description:** Export: copy `~/.zshrc`, `~/.zprofile`, `~/.zshenv`, `~/.p10k.zsh` (if present),
oh-my-zsh custom plugins/themes. Redact lines matching secret patterns (export `API_KEY=`, tokens).
Install: place files, install oh-my-zsh if absent, install Powerlevel10k if detected.

**Acceptance criteria:**
- [ ] Redaction strips lines matching `export .*(KEY|TOKEN|SECRET|PASSWORD)=.*` (regex configurable)
- [ ] Redacted lines replaced with `# REDACTED — restore manually`
- [ ] oh-my-zsh custom dir restored (plugins, themes) but not the oh-my-zsh core (re-cloned fresh)
- [ ] Shell opens cleanly after install (no source errors)

**Verification:**
- [ ] Unit test: redaction regex strips secrets, preserves other exports
- [ ] Integration: zsh starts without errors after restore

**Dependencies:** Task 3

**Files:** `internal/modules/shell/shell.go`, `internal/modules/shell/redact.go`,
`internal/modules/shell/shell_test.go`

**Scope:** M

---

#### Task 6: Kitty terminal module

**Description:** Export: `~/.config/kitty/` (kitty.conf + all includes + themes). Install:
copy to same path, then run `kitty +runpy 'from kitty.fast_data_types import *; ...'` to
reload if kitty is running.

**Acceptance criteria:**
- [ ] All kitty config files (kitty.conf, color themes, session files) exported
- [ ] Install correctly handles symlinks inside kitty config dir
- [ ] If kitty is not installed on target, emit a note (brew module handles install)

**Verification:**
- [ ] Export produces a dir with kitty.conf at root
- [ ] After install, `kitty --config ~/.config/kitty/kitty.conf --debug-config` exits 0

**Dependencies:** Task 4 (kitty installed via brew)

**Files:** `internal/modules/kitty/kitty.go`

**Scope:** S

---

#### Task 7: Cursor + editor module

**Description:** Export: `~/Library/Application Support/Cursor/User/{settings.json,keybindings.json}`,
extension list via `cursor --list-extensions`. Install: place settings files, then
`cursor --install-extension <ext>` for each. Redact any tokens/snippets with inline credentials.

**Acceptance criteria:**
- [ ] `settings.json` and `keybindings.json` restored exactly
- [ ] All extensions installed (failures logged, not fatal)
- [ ] Snippets dir (`snippets/`) included in export

**Verification:**
- [ ] Extension count on target matches source (or diff logged)

**Dependencies:** Task 4 (cursor installed via brew cask)

**Files:** `internal/modules/cursor/cursor.go`

**Scope:** S

---

#### Task 8: Claude Code + Codex + pi.dev module

**Description:** Export Claude Code: `~/.claude/` (CLAUDE.md, settings.json, plugins cache metadata,
`keybindings.json`). Export Codex: `~/.codex/` config. Export pi.dev: `~/.pi/` or
`~/.config/pi/` (detect path from running process). Install: copy configs, re-run any
`npm install -g` or CLI auth flows with prompts.

**Acceptance criteria:**
- [ ] `~/.claude/CLAUDE.md` and `settings.json` restored
- [ ] Plugin metadata (not plugin code) exported so `skillshare sync` can re-pull
- [ ] No auth tokens in exported files (detect + redact pattern `"token":`, `"key":`)

**Verification:**
- [ ] `claude --version` works after install
- [ ] CLAUDE.md present and non-empty on target

**Dependencies:** Task 5 (shell + PATH needed)

**Files:** `internal/modules/ai_tools/claude.go`, `internal/modules/ai_tools/codex.go`,
`internal/modules/ai_tools/pi.go`

**Scope:** M

---

#### Task 9: Skillshare CLI module

**Description:** Export: installed plugin list from `skillshare list` (or equivalent manifest
file). Install: install `skillshare` npm package, then `skillshare install <plugin>` for each.
Plugins pulled from their registry — no plugin code stored in the archive (privacy).

**Acceptance criteria:**
- [ ] Plugin manifest (name + version) exported as JSON
- [ ] Install re-pulls each plugin from registry
- [ ] Plugins that fail to install are logged with reason, not fatal

**Verification:**
- [ ] `skillshare list` on target matches manifest

**Dependencies:** Task 8

**Files:** `internal/modules/skillshare/skillshare.go`

**Scope:** S

---

#### Task 10: SwiftBar + Alfred + utility apps module

**Description:** SwiftBar: export `~/Library/Application Support/SwiftBar/` plugin scripts +
prefs. Alfred: export Alfred preferences dir (user chooses sync dir — detect from prefs).
Klack: export `~/Library/Preferences/com.trphotography.Klack.plist`. f.lux, BetterDisplay,
Tailscale, Shottr, OrbStack: export relevant `~/Library/Preferences/` plist files.

Privacy note: Alfred workflows may contain credentials — emit a warning, let user opt-out
per-app in `onboard.yaml`.

**Acceptance criteria:**
- [ ] Each app's plist/prefs exported to archive under `prefs/<app>/`
- [ ] Install uses `defaults import` or file copy as appropriate per app
- [ ] Per-app opt-out in config (`modules.alfred.skip: true`)

**Verification:**
- [ ] Exported archive contains expected plist for each enabled app
- [ ] After install, each app launches without reconfiguration

**Dependencies:** Task 4 (apps installed via brew)

**Files:** `internal/modules/prefs/prefs.go` (generic plist handler),
`internal/modules/swiftbar/swiftbar.go`, `internal/modules/alfred/alfred.go`

**Scope:** M

---

### Checkpoint: Phase 2

- [ ] `mac-onboarding export` on source Mac produces a valid archive
- [ ] `mac-onboarding install` on target restores at least kitty, cursor, claude, brew
- [ ] `--only <module>` works for all implemented modules
- [ ] Human does a real install test on the work laptop before Phase 3

---

### Phase 3: System + Dotfiles

#### Task 11: Git module

**Description:** Export `~/.gitconfig`, `~/.gitignore_global`, `~/.config/git/`. Redact
credential helpers that embed passwords. Install: copy files. Emit reminder to add SSH key
to GitHub (can't automate — key stays on source).

**Acceptance criteria:**
- [ ] `.gitconfig` restored with correct `user.name` and `user.email`
- [ ] Credential section with embedded passwords redacted
- [ ] SSH key reminder printed at install completion

**Verification:**
- [ ] `git config --global user.email` returns correct value on target

**Dependencies:** Task 5

**Files:** `internal/modules/git/git.go`

**Scope:** S

---

#### Task 12: macOS system settings module

**Description:** Export: capture relevant `defaults read` values across domains
(Dock, Finder, keyboard repeat rate, trackpad, screenshot location, etc.). Store as a YAML
manifest of `defaults write` commands. Install: replay the manifest. Include a curated
allowlist of safe-to-restore defaults (avoid MDM-managed domains).

**Acceptance criteria:**
- [ ] Manifest covers: Dock position/size/autohide, Finder show hidden files,
  key repeat rate, screenshot format/location, dark mode, trackpad speed
- [ ] MDM-managed domains are skipped with a warning (not fatal)
- [ ] `--dry-run` prints the `defaults write` commands without executing

**Verification:**
- [ ] After install, `defaults read com.apple.dock autohide` matches source

**Dependencies:** Task 3

**Files:** `internal/modules/system/system.go`, `internal/modules/system/defaults.go`,
`internal/modules/system/allowlist.go`

**Scope:** M

---

#### Task 13: Hotkeys module

**Description:** Export macOS global keyboard shortcuts via
`defaults read com.apple.symbolichotkeys`. Store as plist. Also capture app-specific
shortcuts for common apps (Raycast/Alfred, SwiftBar). Install: `defaults write` + restart
relevant daemons (`/System/Library/CoreServices/pbs -flush`).

**Acceptance criteria:**
- [ ] Global shortcuts exported and restored
- [ ] Install doesn't clobber Accessibility shortcuts on MDM machines (probe first)
- [ ] Restart instructions printed if daemon restart requires sudo

**Verification:**
- [ ] `defaults read com.apple.symbolichotkeys` on target matches source for key bindings

**Dependencies:** Task 12

**Files:** `internal/modules/hotkeys/hotkeys.go`

**Scope:** S

---

### Checkpoint: Phase 3

- [ ] Full `mac-onboarding export` + `mac-onboarding install` tested end-to-end on real hardware
- [ ] Privacy review: no secrets in archive
- [ ] Human signs off before bridge work starts

---

### Phase 4: Bridge

#### Task 14: Bridge mode (live pull over Tailscale SSH)

**Description:** Source machine runs `mac-onboarding bridge serve` — starts an SSH server on
a random high port, advertises via a local file or Tailscale MagicDNS hostname.
Target runs `mac-onboarding bridge pull --from <hostname> --module kitty` — SSHes in,
runs export for that module in a temp dir, streams the archive back, installs it.
Auth: SSH key only (no passwords). Key exchange: user pastes source's public key into
target's `~/.ssh/authorized_keys` once.

**Acceptance criteria:**
- [ ] `bridge serve` listens and accepts connections authenticated by SSH key
- [ ] `bridge pull --module <name>` installs any single module live from source
- [ ] Connection fails gracefully if Tailscale not running (clear error message)
- [ ] No plaintext data over the wire (SSH transport)

**Verification:**
- [ ] End-to-end test: pull kitty config from source machine, verify on target

**Dependencies:** Tasks 4–13 (modules must exist to be pulled), Task 10 (tailscale)

**Files:** `internal/bridge/server.go`, `internal/bridge/client.go`, `cmd/bridge.go`

**Scope:** L

---

### Checkpoint: Phase 4

- [ ] Bridge tested over real Tailscale network between two Macs
- [ ] No credentials leak in bridge protocol

---

### Phase 5: Open-source Polish

#### Task 15: README + `onboard.yaml.example` finalization

**Description:** README covers: what it does, prerequisites, quickstart (source Mac →
target Mac in 5 steps), module list with what each restores, bridge usage, privacy model
(what is and isn't stored), contributing. `onboard.yaml.example` has every field with
inline comments and placeholder values only.

**Acceptance criteria:**
- [ ] README has a "Privacy" section explaining redaction + what's never stored
- [ ] Quickstart can be followed by someone who has never seen the tool
- [ ] No real usernames, emails, tokens, or hostnames in any committed file

**Verification:**
- [ ] Fresh clone → follow README → binary works

**Dependencies:** All previous tasks

**Files:** `README.md`, `onboard.yaml.example`

**Scope:** S

---

#### Task 16: GitHub Actions CI + release binary

**Description:** CI: `go test ./...` + `go vet` + `staticcheck` on push to main.
Release workflow: on tag `v*.*.*`, build `darwin/amd64` and `darwin/arm64` binaries,
attach to GitHub Release. Update `install.sh` to download from releases URL.

**Acceptance criteria:**
- [ ] CI passes on every PR
- [ ] Tagged release produces two binaries (Intel + Apple Silicon)
- [ ] `install.sh` downloads correct arch binary automatically

**Verification:**
- [ ] Push a `v0.1.0` tag → release appears with both binaries

**Dependencies:** Task 15

**Files:** `.github/workflows/ci.yml`, `.github/workflows/release.yml`

**Scope:** S

---

### Checkpoint: Complete

- [ ] All 16 tasks done
- [ ] End-to-end test on a real MDM work Mac
- [ ] No secrets in git history (`git log -p | grep -i token` clean)
- [ ] Repo set to public, license added (MIT)

---

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| MDM blocks cask installs or plist writes | High | Probe before each write; skip + warn; `--force` flag for user to override |
| Alfred workflows contain embedded credentials | High | Warn + require explicit opt-in; emit redaction report |
| macOS version delta (source ≠ target) | Med | Capture OS version in manifest; warn on mismatch before applying defaults |
| Tailscale not running at bridge time | Med | Clear pre-flight check with fix instructions |
| oh-my-zsh custom plugins have private Git URLs | Med | Log and skip private URLs; user re-adds manually |
| Binary quarantine (Gatekeeper) on fresh Mac | Med | `install.sh` runs `xattr -d com.apple.quarantine` after download |
| Skillshare CLI API changes between versions | Low | Pin version in `onboard.yaml`; emit version mismatch warning |

## Open Questions

1. **1Password vault** — no automation possible (security by design). Include a guide section
   in README or a `--guide 1password` subcommand that prints setup steps?
2. **Synology** — DriveClient prefs are mostly server hostname + credentials. Export hostname only;
   prompt for password on install. Confirm this is the right scope.
3. **Private repo first?** — Yes, recommended. Make private while bridge and redaction logic
   are being tested. Flip to public at Task 15.
4. **pi.dev config path** — need to confirm actual config location (`~/.config/pi` vs `~/.pi`).
   Probe both during Task 8.
5. **Codex config location** — confirm `~/.codex/` is the right path on macOS.
