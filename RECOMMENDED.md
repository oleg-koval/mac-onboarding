# Recommended Setup

A reference setup for new Macs. Everything here is captured automatically by `mac-onboarding export` once you've applied it locally.

## 1. Brewfile

Start from `Brewfile.example`:

```bash
cp Brewfile.example ~/.Brewfile
brew bundle install --file=~/.Brewfile
```

`mac-onboarding export` reads `~/.Brewfile` by default. Add or remove tools as you like — the next export captures them.

## 2. Shell — zsh + Starship

Install [Starship](https://starship.rs) prompt:

```bash
brew install starship
echo 'eval "$(starship init zsh)"' >> ~/.zshrc
```

For the [Catppuccin Powerline](https://starship.rs/presets/catppuccin-powerline) preset:

```bash
mkdir -p ~/.config && \
  starship preset catppuccin-powerline -o ~/.config/starship.toml
```

Or, stock zsh with git branch in prompt — drop into `~/.zshrc`:

```zsh
# Load version control information
autoload -Uz vcs_info
precmd() { vcs_info }

# Format the vcs_info_msg_0_ variable
zstyle ':vcs_info:git:*' formats '(%b)'

# Set up the prompt (with git branch name)
setopt PROMPT_SUBST
PROMPT='%F{magenta}==>%f %F{blue}${PWD/#$HOME/~}%f %F{cyan}${vcs_info_msg_0_}%f%F{green} ==> %f'
```

## 3. Git defaults

```bash
git config --global user.name "Your Name"
git config --global user.email you@example.com
git config --global push.autoSetupRemote true
git config --global init.defaultBranch main

# Verify
git config --list
```

The `git` module captures `~/.gitconfig`, `~/.gitignore_global`, and `~/.config/git/` — credentials are redacted.

## 4. Fonts

Programming fonts ship via Homebrew casks (already in `Brewfile.example`):

- **Fira Code** — popular, free, with ligatures
- **JetBrains Mono** — JetBrains' free programming font
- **Nerd Font variants** — include glyphs for Powerline / Starship icons

Set them in your terminal of choice (Kitty, iTerm2, Cursor, etc.).

## 5. Recommended order

1. Run `mac-onboarding install ~/onboard.tar.gz` — Xcode CLT, Homebrew, modules
2. Install the Brewfile bundle (auto-handled by `brew` module)
3. Configure Starship, dotfiles, fonts (auto-handled by `shell` + `system` modules)
4. Manually re-add SSH keys, 1Password, cloud creds (intentionally not exported)
