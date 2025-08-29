#!/bin/bash
set -e

# PCP Installation Script
# Usage: curl -fsSL https://github.com/riazarbi/pcp/releases/latest/download/install.sh | sh

REPO="riazarbi/pcp"
BINARY_NAME="pcp"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print functions
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Detect OS and architecture
detect_platform() {
    local os arch
    
    case "$(uname -s)" in
        Linux*)
            os="linux"
            ;;
        Darwin*)
            os="darwin"
            ;;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            print_error "This script supports Linux and macOS only."
            print_error "For Windows, please use the PowerShell install script."
            exit 1
            ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        *)
            print_error "Unsupported architecture: $(uname -m)"
            print_error "Supported architectures: x86_64/amd64, arm64/aarch64"
            exit 1
            ;;
    esac
    
    echo "${os}-${arch}"
}

# Get latest release version
get_latest_version() {
    local version
    version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
              grep '"tag_name":' | \
              sed -E 's/.*"tag_name": "([^"]+)".*/\1/')
    
    if [ -z "$version" ]; then
        print_error "Failed to get latest version from GitHub API"
        exit 1
    fi
    
    echo "$version"
}

# Download and verify binary
download_binary() {
    local version="$1"
    local platform="$2"
    local binary_name="${BINARY_NAME}-${platform}"
    local download_url="https://github.com/${REPO}/releases/download/${version}/${binary_name}"
    local checksums_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"
    
    print_info "Downloading ${binary_name} ${version}..."
    
    # Create temporary directory
    local tmp_dir
    tmp_dir=$(mktemp -d)
    local binary_path="${tmp_dir}/${binary_name}"
    local checksums_path="${tmp_dir}/checksums.txt"
    
    # Download binary
    if ! curl -fsSL "$download_url" -o "$binary_path"; then
        print_error "Failed to download binary from: $download_url"
        rm -rf "$tmp_dir"
        exit 1
    fi
    
    # Download checksums for verification
    print_info "Downloading checksums for verification..."
    if ! curl -fsSL "$checksums_url" -o "$checksums_path"; then
        print_warning "Failed to download checksums, skipping verification"
    else
        # Verify checksum
        print_info "Verifying checksum..."
        if command -v sha256sum >/dev/null 2>&1; then
            local expected_checksum actual_checksum
            expected_checksum=$(grep "$binary_name" "$checksums_path" | awk '{print $1}')
            actual_checksum=$(sha256sum "$binary_path" | awk '{print $1}')
            
            if [ "$expected_checksum" = "$actual_checksum" ]; then
                print_success "Checksum verification passed"
            else
                print_error "Checksum verification failed!"
                print_error "Expected: $expected_checksum"
                print_error "Actual:   $actual_checksum"
                rm -rf "$tmp_dir"
                exit 1
            fi
        elif command -v shasum >/dev/null 2>&1; then
            local expected_checksum actual_checksum
            expected_checksum=$(grep "$binary_name" "$checksums_path" | awk '{print $1}')
            actual_checksum=$(shasum -a 256 "$binary_path" | awk '{print $1}')
            
            if [ "$expected_checksum" = "$actual_checksum" ]; then
                print_success "Checksum verification passed"
            else
                print_error "Checksum verification failed!"
                print_error "Expected: $expected_checksum"
                print_error "Actual:   $actual_checksum"
                rm -rf "$tmp_dir"
                exit 1
            fi
        else
            print_warning "sha256sum/shasum not available, skipping checksum verification"
        fi
    fi
    
    echo "$binary_path"
}

# Install binary
install_binary() {
    local binary_path="$1"
    local install_dir
    
    # Determine install directory
    if [ -w "/usr/local/bin" ]; then
        install_dir="/usr/local/bin"
    elif [ -d "$HOME/.local/bin" ]; then
        install_dir="$HOME/.local/bin"
    else
        # Create ~/.local/bin if it doesn't exist
        install_dir="$HOME/.local/bin"
        mkdir -p "$install_dir"
        print_info "Created directory: $install_dir"
    fi
    
    local install_path="${install_dir}/${BINARY_NAME}"
    
    # Copy and make executable
    print_info "Installing to: $install_path"
    cp "$binary_path" "$install_path"
    chmod +x "$install_path"
    
    # Check if install directory is in PATH
    if ! echo "$PATH" | grep -q "$install_dir"; then
        print_warning "Install directory $install_dir is not in your PATH"
        print_warning "Add it to your PATH by adding this line to your shell profile:"
        print_warning "  export PATH=\"$install_dir:\$PATH\""
        
        # Try to add to common shell profiles
        for shell_profile in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
            if [ -f "$shell_profile" ] && ! grep -q "$install_dir" "$shell_profile"; then
                print_info "Would you like to add $install_dir to PATH in $shell_profile? [y/N]"
                if [ "${AUTO_CONFIRM:-}" = "true" ]; then
                    echo "y (auto-confirmed)"
                    response="y"
                else
                    read -r response
                fi
                
                if [ "$response" = "y" ] || [ "$response" = "Y" ]; then
                    echo "export PATH=\"$install_dir:\$PATH\"" >> "$shell_profile"
                    print_success "Added $install_dir to PATH in $shell_profile"
                    print_info "Please restart your shell or run: source $shell_profile"
                    break
                fi
            fi
        done
    fi
    
    echo "$install_path"
}

# Main installation function
main() {
    print_info "Installing PCP (Prompt Composition Processor)..."
    
    # Check for required tools
    for tool in curl grep sed awk; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            print_error "Required tool '$tool' is not installed"
            exit 1
        fi
    done
    
    # Detect platform
    local platform
    platform=$(detect_platform)
    print_info "Detected platform: $platform"
    
    # Get latest version
    local version
    version=$(get_latest_version)
    print_info "Latest version: $version"
    
    # Download binary
    local binary_path
    binary_path=$(download_binary "$version" "$platform")
    
    # Install binary
    local install_path
    install_path=$(install_binary "$binary_path")
    
    # Clean up
    rm -rf "$(dirname "$binary_path")"
    
    # Test installation
    print_info "Testing installation..."
    if "$install_path" -h >/dev/null 2>&1; then
        print_success "PCP $version installed successfully!"
        print_info "Location: $install_path"
        print_info "Run 'pcp -h' to get started"
    else
        print_error "Installation test failed"
        exit 1
    fi
}

# Run main function
main "$@"