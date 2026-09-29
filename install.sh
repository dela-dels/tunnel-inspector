#!/bin/sh
set -e

# Cloudflare Tunnel Inspector installer script
# Works on macOS and Linux across any shell (bash, zsh, fish, etc.)

REPO="delaakakpo/tunnel-inspector"
BINARY_NAME="tunnel-inspector"
ALIAS_NAME="ti"

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
SKY='\033[0;36m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m' # No Color

info() {
    printf "${SKY}info:${NC} %s\n" "$1"
}

success() {
    printf "${GREEN}success:${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}warning:${NC} %s\n" "$1"
}

error() {
    printf "${RED}error:${NC} %s\n" "$1" >&2
    exit 1
}

# 1. Detect OS
OS="$(uname -s)"
case "$OS" in
    Darwin) OS="darwin" ;;
    Linux)  OS="linux" ;;
    *) error "Unsupported operating system: $OS. Supported OS: macOS, Linux." ;;
esac

# 2. Detect CPU Architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)   ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) error "Unsupported architecture: $ARCH. Supported architectures: x86_64/amd64, arm64." ;;
esac

info "Detected platform: ${BOLD}${OS}_${ARCH}${NC}"

# 3. Determine Installation Directory
if [ -n "$BINDIR" ]; then
    INSTALL_DIR="$BINDIR"
elif [ "$(id -u)" -eq 0 ]; then
    INSTALL_DIR="/usr/local/bin"
else
    INSTALL_DIR="$HOME/.local/bin"
fi

mkdir -p "$INSTALL_DIR"

# 4. Resolve Version
VERSION="${VERSION:-latest}"
if [ "$VERSION" = "latest" ]; then
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
else
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY_NAME}_${OS}_${ARCH}.tar.gz"
fi

info "Downloading ${BINARY_NAME} from GitHub..."

# Temporary directory for download and extraction
TMP_DIR="$(mktemp -d 2>/dev/null || mktemp -d -t 'tunnel-inspector')"
cleanup() {
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

ARCHIVE_PATH="$TMP_DIR/${BINARY_NAME}.tar.gz"

# Download with curl or wget
if command -v curl >/dev/null 2>&1; then
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$ARCHIVE_PATH"; then
        error "Failed to download $DOWNLOAD_URL. Please check your internet connection or verify the release exists."
    fi
elif command -v wget >/dev/null 2>&1; then
    if ! wget -q "$DOWNLOAD_URL" -O "$ARCHIVE_PATH"; then
        error "Failed to download $DOWNLOAD_URL. Please check your internet connection or verify the release exists."
    fi
else
    error "Neither curl nor wget was found. Please install either curl or wget to continue."
fi

# 5. Extract Archive
info "Extracting archive..."
tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR"

if [ ! -f "$TMP_DIR/$BINARY_NAME" ]; then
    error "Binary '$BINARY_NAME' was not found in downloaded archive."
fi

# 6. Install Binary and 'ti' symlink
cp "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

# Create symlink 'ti' so users can run 'ti' as well as 'tunnel-inspector'
ln -sf "$INSTALL_DIR/$BINARY_NAME" "$INSTALL_DIR/$ALIAS_NAME"

success "Installed ${BOLD}${BINARY_NAME}${NC} and ${BOLD}${ALIAS_NAME}${NC} to ${INSTALL_DIR}"

# 7. Check PATH and update user's shell configuration if needed
case ":$PATH:" in
    *":$INSTALL_DIR:"*)
        PATH_CONFIGURED=1
        ;;
    *)
        PATH_CONFIGURED=0
        ;;
esac

if [ "$PATH_CONFIGURED" -eq 0 ]; then
    warn "$INSTALL_DIR is not currently in your PATH."

    SHELL_NAME="$(basename "$SHELL")"
    CONFIG_FILE=""
    EXPORT_LINE=""

    case "$SHELL_NAME" in
        zsh)
            CONFIG_FILE="${ZDOTDIR:-$HOME}/.zshrc"
            EXPORT_LINE="export PATH=\"$INSTALL_DIR:\$PATH\""
            ;;
        bash)
            if [ -f "$HOME/.bashrc" ]; then
                CONFIG_FILE="$HOME/.bashrc"
            elif [ -f "$HOME/.bash_profile" ]; then
                CONFIG_FILE="$HOME/.bash_profile"
            else
                CONFIG_FILE="$HOME/.profile"
            fi
            EXPORT_LINE="export PATH=\"$INSTALL_DIR:\$PATH\""
            ;;
        fish)
            CONFIG_FILE="$HOME/.config/fish/config.fish"
            EXPORT_LINE="fish_add_path $INSTALL_DIR"
            ;;
        ksh|sh|ash)
            CONFIG_FILE="$HOME/.profile"
            EXPORT_LINE="export PATH=\"$INSTALL_DIR:\$PATH\""
            ;;
        *)
            CONFIG_FILE="$HOME/.profile"
            EXPORT_LINE="export PATH=\"$INSTALL_DIR:\$PATH\""
            ;;
    esac

    if [ -n "$CONFIG_FILE" ]; then
        if [ ! -f "$CONFIG_FILE" ]; then
            touch "$CONFIG_FILE"
        fi

        # Check if already present in rc file
        if ! grep -qs "$INSTALL_DIR" "$CONFIG_FILE"; then
            printf "\n# Added by tunnel-inspector installer\n%s\n" "$EXPORT_LINE" >> "$CONFIG_FILE"
            info "Added $INSTALL_DIR to PATH in ${BOLD}$CONFIG_FILE${NC}"
        fi

        printf "\nTo start using tunnel-inspector immediately in this terminal, run:\n"
        if [ "$SHELL_NAME" = "fish" ]; then
            printf "  ${BOLD}source %s${NC}\n\n" "$CONFIG_FILE"
        else
            printf "  ${BOLD}export PATH=\"%s:\$PATH\"${NC}\n\n" "$INSTALL_DIR"
        fi
    fi
fi

printf "${GREEN}${BOLD}✓ Setup complete!${NC}\n"
printf "You can now run either:\n"
printf "  ${BOLD}tunnel-inspector start --port 8000${NC}\n"
printf "  ${BOLD}ti start --port 8000${NC}\n\n"
