#!/bin/bash

# Configuration
GO_VERSION="1.24.3" # Full version required

install_or_update_go() {
    
    # Check current Go version
    local current_version
    if command -v go >/dev/null 2>&1; then
        current_version=$(go version | awk '{print $3}' | sed 's/go//')
    else
        current_version="not_installed"
    fi
    
    # Skip if already up to date
    if [[ "$current_version" == "$GO_VERSION" ]]; then
        echo "Go $GO_VERSION is already installed"
        return 0
    fi
    
    # Detect platform
    local os arch filename
    case "$(uname -s)" in
        Linux*)     os="linux" ;;
        Darwin*)    os="darwin" ;;
        *)          echo "Unsupported OS"; return 1 ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64)   arch="amd64" ;;
        arm64|aarch64)  arch="arm64" ;;
        armv6l)         arch="armv6l" ;;
        i386|i686)      arch="386" ;;
        *)              echo "Unsupported architecture"; return 1 ;;
    esac
    
    filename="go${GO_VERSION}.${os}-${arch}.tar.gz"
    
    echo "Installing Go $GO_VERSION for $os-$arch..."
    
    # Download and install
    local temp_dir="/tmp/go_install_$$"
    mkdir -p "$temp_dir"
    cd "$temp_dir" || return 1
    
    if command -v curl >/dev/null 2>&1; then
        curl -sLO "https://go.dev/dl/${filename}"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "https://go.dev/dl/${filename}"
    else
        echo "Need curl or wget to download Go"
        return 1
    fi
    
    [[ -f "$filename" ]] || { echo "Download failed"; return 1; }
    
    # Remove old installation and install new
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "$filename"
    
    # Cleanup
    cd / && rm -rf "$temp_dir"
    
    echo "Go $GO_VERSION installed successfully"
    
    # Add to PATH if not already there
    if ! echo "$PATH" | grep -q "/usr/local/go/bin"; then
        echo "Add /usr/local/go/bin to your PATH"
    fi
}




# Sudo check
if [ "$EUID" -ne 0 ]; then
    echo "Please run as root"
    exit 1
fi
