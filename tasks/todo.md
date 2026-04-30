# mac-onboarding — Task List

## Phase 1: Foundation
- [ ] Task 1: Go project scaffold + CLI skeleton (cobra, `export`/`install`/`bridge` cmds, `install.sh`)
- [ ] Task 2: Config loader + `onboard.yaml` schema (YAML parse, validation, path expansion)
- [ ] Task 3: Bootstrap module (Xcode CLT + Homebrew + MDM detection)

## ✅ Checkpoint: Foundation
- [ ] Binary builds, install.sh works, config loads, MDM probe works

## Phase 2: Core App Modules
- [ ] Task 4: Homebrew module (Brewfile export + `brew bundle install`)
- [ ] Task 5: Shell module (zsh rc files + oh-my-zsh + secret redaction)
- [ ] Task 6: Kitty terminal module (config dir export/restore)
- [ ] Task 7: Cursor + editor module (settings, keybindings, extensions)
- [ ] Task 8: Claude Code + Codex + pi.dev module (config dirs, no tokens)
- [ ] Task 9: Skillshare CLI module (plugin manifest export + re-install)
- [ ] Task 10: SwiftBar + Alfred + utility prefs (Klack, f.lux, BetterDisplay, Tailscale, Shottr, OrbStack)

## ✅ Checkpoint: Core App Modules
- [ ] Real export + install tested on actual hardware (work laptop)

## Phase 3: System + Dotfiles
- [ ] Task 11: Git module (~/.gitconfig, ~/.gitignore_global, redaction + SSH reminder)
- [ ] Task 12: macOS system settings module (defaults write manifest, MDM allowlist)
- [ ] Task 13: Hotkeys module (symbolic hotkeys plist export/restore)

## ✅ Checkpoint: System + Dotfiles
- [ ] Full end-to-end tested, privacy review done

## Phase 4: Bridge
- [ ] Task 14: Bridge mode (SSH serve + pull over Tailscale, any module on-demand)

## ✅ Checkpoint: Bridge
- [ ] Tested over real Tailscale network

## Phase 5: Open-source Polish
- [ ] Task 15: README + onboard.yaml.example (privacy section, quickstart, no real values)
- [ ] Task 16: GitHub Actions CI + release binaries (darwin/amd64 + darwin/arm64)

## ✅ Checkpoint: Complete
- [ ] End-to-end on MDM Mac, no secrets in history, repo public-ready
