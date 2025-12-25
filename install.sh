#!/bin/bash
# SV (Switch Version) Installer
# https://github.com/voocel/sv
#
# Pretty Go Version Manager

set -eu

# Constants
SV_VERSION="${SV_VERSION:-latest}"
readonly SV_HOME="${SV_HOME:-$HOME/.sv}"
readonly INSTALL_DIR="${INSTALL_DIR:-$HOME/.sv/bin}"
readonly REPO_URL="https://github.com/voocel/sv"
readonly DOWNLOAD_URL="$REPO_URL/releases/download"

# Environment variables
GOPATH=${GOPATH:-$HOME/go}
GOROOT=${GOROOT:-$HOME/.sv/go}

if [ "$(echo "$GOROOT" | cut -c1)" != "/" ]; then
    echo "Error: GOROOT must be an absolute path but it is set to $GOROOT" >&2
    exit 1
fi

print_banner() {
    cat <<-'EOF'
=================================================
              ___
             /  /\          ___
            /  /::\        /  /\
           /__/:/\:\      /  /:/
          _\_ \:\ \:\    /  /:/
         /__/\ \:\ \:\  /__/:/  ___
         \  \:\ \:\_\/  |  |:| /  /\
          \  \:\_\:\    |  |:|/  /:/
           \  \:\/:/    |__|:|__/:/
            \  \::/      \__\::::/
             \__\/           ````
       ___           _        _ _
      |_ _|_ __  ___| |_ __ _| | | ___ _ __
       | || '_ \/ __| __/ _` | | |/ _ \ '__|
       | || | | \__ \ || (_| | | |  __/ |
      |___|_| |_|___/\__\__,_|_|_|\___|_|
==================================================
EOF
}

# Colors
setup_colors() {
    if [ -t 1 ]; then
        CYAN='\033[0;36m'
        GREEN='\033[0;32m'
        YELLOW='\033[1;33m'
        RED='\033[0;31m'
        BLUE='\033[0;34m'
        BOLD='\033[1m'
        RESET='\033[0m'
    else
        CYAN='' GREEN='' YELLOW='' RED='' BLUE='' BOLD='' RESET=''
    fi
}

# Logging functions
log() { echo -e "${GREEN}info:${RESET} $*"; }
warn() { echo -e "${YELLOW}warn:${RESET} $*"; }
error() { echo -e "${RED}error:${RESET} $*" >&2; }
info() { echo -e "${CYAN}$*${RESET}"; }
step() { echo -e "${YELLOW}$*${RESET}"; }

# Error handling
fatal() {
    error "$*"
    exit 1
}

# Platform detection for binary download
get_svbin() {
    local svbin=''
    local thisos arch

    thisos=$(uname -s)
    arch=$(uname -m)

    case $thisos in
       Linux*)
          case $arch in
            arm64|aarch64)
              svbin="sv-linux-arm64"
              ;;
            *)
              svbin="sv-linux-amd64"
              ;;
          esac
          ;;
       Darwin*)
          case $arch in
            arm64)
              svbin="sv-darwin-arm64"
              ;;
            *)
              svbin="sv-darwin-amd64"
              ;;
          esac
          ;;
       Windows*|CYGWIN*|MINGW*|MSYS*)
          svbin="sv-windows-amd64.exe"
          ;;
        *)
          fatal "Unsupported OS: $thisos"
          ;;
    esac

    echo "$svbin"
}

# Check dependencies
check_dependencies() {
    if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
        fatal "curl or wget is required but not installed"
    fi
}

# Get latest version from GitHub API
get_latest_version() {
    local version
    local api_url="https://api.github.com/repos/voocel/sv/releases/latest"
    
    if command -v curl >/dev/null 2>&1; then
        version=$(curl -s "$api_url" | grep '"tag_name":' | cut -d'"' -f4)
    else
        version=$(wget -qO- "$api_url" | grep '"tag_name":' | cut -d'"' -f4)
    fi
    
    if [ -z "$version" ]; then
        fatal "Failed to fetch latest version"
    fi
    
    echo "$version"
}

# Download binary (keeping original download logic)
download_sv() {
    local version="$1"
    local svbin="$2"
    local download_url="$DOWNLOAD_URL/$version/$svbin"
    
    log "Downloading sv $version"
    info "URL: $download_url"
    
    # Ensure install directory exists
    mkdir -p "$INSTALL_DIR"
    
    # Download using wget (original method)
    if command -v wget >/dev/null 2>&1; then
        if ! wget "$download_url" -O "$INSTALL_DIR/sv"; then
            fatal "Failed to download sv binary"
        fi
    else
        if ! curl -fsSL "$download_url" -o "$INSTALL_DIR/sv"; then
            fatal "Failed to download sv binary"
        fi
    fi
    
    chmod +x "$INSTALL_DIR/sv"
    log "Installed sv to $INSTALL_DIR/sv"
}

# Generate environment file (original functionality)
gen_env() {
    log "Generating environment configuration"

    # Ensure SV_HOME directory exists
    mkdir -p "$SV_HOME"

    # Use user's GOPROXY if set, otherwise use default
    local goproxy="${GOPROXY:-https://proxy.golang.org,direct}"

    # Use single quotes to prevent variable expansion, keeping $HOME dynamic
    cat > "$SV_HOME/env" << 'ENVEOF'
#!/bin/sh
# sv shell setup
case ":${PATH}:" in
    *:"$HOME/.sv/go/bin:$HOME/.sv/bin":*)
        ;;
    *)
        export GO111MODULE=auto
        export SVHOME="$HOME/.sv"
        export GOROOT="$HOME/.sv/go"
ENVEOF
    # Append GOPROXY with the resolved value
    echo "        export GOPROXY=$goproxy" >> "$SV_HOME/env"
    cat >> "$SV_HOME/env" << 'ENVEOF'
        export PATH="$HOME/.sv/go/bin:$HOME/.sv/bin:$PATH"
        ;;
esac
ENVEOF
}

# Shell detection
get_shell_profile() {
    if [[ "${SHELL}" == *"bash"* ]]; then
        if [[ -f "$HOME/.bashrc" ]]; then
            echo "$HOME/.bashrc"
        elif [[ -f "$HOME/.bash_profile" ]]; then
            echo "$HOME/.bash_profile"
        else
            echo "$HOME/.bashrc"
        fi
    elif [[ "${SHELL}" == *"zsh"* ]]; then
        echo "${ZDOTDIR:-$HOME}/.zshrc"
    else
        echo "$HOME/.profile"
    fi
}

# Set environment (original logic with improvements)
set_env() {
    local shell_profile
    shell_profile=$(get_shell_profile)
    
    local env_str='. "$HOME/.sv/env"'
    
    # Create profile if it doesn't exist
    mkdir -p "$(dirname "$shell_profile")"
    touch "$shell_profile"
    
    if grep -q "$env_str" "$shell_profile" 2>/dev/null; then
        info "SV environment already configured in $shell_profile"
        return 0
    fi
    
    echo "" >> "$shell_profile"
    echo "# Added by sv installer" >> "$shell_profile"
    echo "$env_str" >> "$shell_profile"
    
    log "Added sv environment to $shell_profile"
}

# Create GOPATH directory structure
setup_go_directories() {
    log "Setting up Go directory structure"
    mkdir -p "$GOPATH/src" "$GOPATH/pkg" "$GOPATH/bin" "$GOROOT"
}

# Verify installation
verify_installation() {
    if [ ! -x "$INSTALL_DIR/sv" ]; then
        fatal "sv binary not found at $INSTALL_DIR/sv"
    fi
    
    # Check if sv is working
    if ! "$INSTALL_DIR/sv" --version >/dev/null 2>&1; then
        fatal "sv binary is not working correctly"
    fi
    
    log "Installation verified successfully"
}

# Print success message
print_success() {
    info ""
    info "${GREEN}${BOLD}sv has been successfully installed!${RESET}"
    info ""
    info "To get started:"
    info "  1. Restart your shell or run: source $(get_shell_profile)"
    info "  2. Run: sv list"
    info "  3. Install Go: sv install 1.24"
    info ""
    info "For more information:"
    info "  Documentation: $REPO_URL#readme"
    info "  Report issues: $REPO_URL/issues"
    info ""
}

# Main installation flow
main() {
    local version svbin

    setup_colors
    print_banner

    # Pre-flight checks first
    check_dependencies
    svbin=$(get_svbin)

    step "[1/4] Fetching sv latest version"
    if [ "$SV_VERSION" = "latest" ]; then
        version=$(get_latest_version)
    else
        version="$SV_VERSION"
    fi
    info "Version to install: $version"

    step "[2/4] Downloading sv binary"
    download_sv "$version" "$svbin"
    verify_installation

    step "[3/4] Setting up environment"
    gen_env
    set_env
    setup_go_directories

    step "[4/4] Installation completed"
    print_success
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "SV (Switch Version) Installer"
        echo ""
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  --help           Show this help"
        echo "  --version VER    Install specific version (default: latest)"
        echo "  --force          Force reinstall even if already installed"
        echo ""
        echo "Environment variables:"
        echo "  SV_VERSION       Version to install"
        echo "  SV_HOME          sv home directory (default: ~/.sv)"
        echo "  INSTALL_DIR      Binary install directory (default: ~/.sv/bin)"
        echo "  GOPATH           Go workspace (default: ~/go)"
        echo "  GOROOT           Go installation (default: ~/.sv/go)"
        exit 0
        ;;
    --version)
        shift
        SV_VERSION="${1:-latest}"
        shift 2>/dev/null || true
        ;;
    --force)
        FORCE_INSTALL=1
        shift
        ;;
esac

FORCE_INSTALL="${FORCE_INSTALL:-0}"

# Check if already installed (original logic)
set +u
if [ -d "${SV_HOME}" ] && [ -f "${SV_HOME}/bin/sv" ] && [ "$FORCE_INSTALL" != "1" ]; then
    setup_colors
    warn "sv is already installed at ${SV_HOME}"
    warn "Use --force to reinstall, or delete the directory and reinstall"
    info "Or run: rm -rf ${SV_HOME} && curl -sL https://raw.githubusercontent.com/voocel/sv/main/install.sh | sh"
    exit 0
fi
set -u

# Run main installation
main "$@"