#!/usr/bin/env bash
# Bootstrap installer for mac-onboarding.
# Installs Go (via Homebrew) if absent, clones the repo, builds the binary.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/oleg-koval/mac-onboarding/main/install.sh | bash
#   # or from a local clone:
#   ./install.sh

set -euo pipefail

REPO="https://github.com/oleg-koval/mac-onboarding.git"
INSTALL_DIR="/usr/local/bin"
BINARY="mac-onboarding"
CLONE_DIR="${HOME}/.local/share/mac-onboarding"

info()  { printf "\033[32m[mac-onboarding]\033[0m %s\n" "$*"; }
warn()  { printf "\033[33m[mac-onboarding]\033[0m %s\n" "$*"; }
fatal() { printf "\033[31m[mac-onboarding]\033[0m %s\n" "$*"; exit 1; }

# --- Xcode CLT ---
if ! xcode-select -p &>/dev/null; then
  info "Installing Xcode Command Line Tools..."
  xcode-select --install
  info "Re-run this script after Xcode CLT installation completes."
  exit 0
fi

# --- Homebrew ---
if ! command -v brew &>/dev/null; then
  info "Installing Homebrew..."
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
  # Add brew to PATH for this session
  eval "$(/opt/homebrew/bin/brew shellenv 2>/dev/null || /usr/local/bin/brew shellenv)"
fi

# --- Go ---
if ! command -v go &>/dev/null; then
  info "Installing Go via Homebrew..."
  brew install go
fi

GO_VERSION=$(go version | awk '{print $3}')
info "Go: ${GO_VERSION}"

# --- Clone or update repo ---
if [ -d "${CLONE_DIR}/.git" ]; then
  info "Updating existing clone at ${CLONE_DIR}..."
  git -C "${CLONE_DIR}" pull --ff-only
else
  info "Cloning to ${CLONE_DIR}..."
  git clone --depth 1 "${REPO}" "${CLONE_DIR}"
fi

# --- Build ---
info "Building ${BINARY}..."
cd "${CLONE_DIR}"
make build

# --- Install binary ---
BUILT="${CLONE_DIR}/dist/${BINARY}"
if [ ! -f "${BUILT}" ]; then
  fatal "Build failed — ${BUILT} not found"
fi

# Remove Gatekeeper quarantine flag (common on fresh Macs)
xattr -d com.apple.quarantine "${BUILT}" 2>/dev/null || true

if [ -w "${INSTALL_DIR}" ]; then
  cp "${BUILT}" "${INSTALL_DIR}/${BINARY}"
else
  info "Copying to ${INSTALL_DIR} (sudo required)..."
  sudo cp "${BUILT}" "${INSTALL_DIR}/${BINARY}"
fi
chmod 755 "${INSTALL_DIR}/${BINARY}"

info "Installed: $(${BINARY} --version 2>/dev/null || echo ok)"
info ""
info "Next steps:"
info "  1. Copy onboard.yaml.example → onboard.yaml and edit it"
info "  2. On source Mac: mac-onboarding export --output ~/onboard.tar.gz"
info "  3. On target Mac: mac-onboarding install --input ~/onboard.tar.gz"
