#!/bin/sh

# Honeywire Automated Installer & Provisioner
set -e

# Pass all arguments into the function
do_install() {
    # 1. Parse Arguments passed via 'bash -s --'
    LINK_URL=""
    API_KEY=""

    while [ $# -gt 0 ]; do
        case "$1" in
            --link)
                LINK_URL="$2"
                shift 2
                ;;
            --api-key)
                API_KEY="$2"
                shift 2
                ;;
            *)
                # Ignore unknown arguments silently or log them,
                # but don't break the install.
                shift 1
                ;;
        esac
    done

    # Define colors
    RESET='\033[0m'
    BOLD='\033[1m'
    DIM='\033[2m'
    CYAN='\033[36m'
    GREEN='\033[32m'
    YELLOW='\033[33m'
    RED='\033[31m'
    MAGENTA='\033[35m'

    echo -e "${CYAN}${BOLD}==> Starting Honeywire Installation...${RESET}"

    # 2. Check for existing installation (Upgrade flow)
    if command -v honeywire >/dev/null 2>&1; then
        EXISTING_PATH=$(command -v honeywire)
        echo -e "${YELLOW}${BOLD}==> Warning:${RESET}${YELLOW} Honeywire is already installed at ${EXISTING_PATH}${RESET}"
        echo -e "${DIM}Upgrading to the latest version in 3 seconds... (Press CTRL + C to cancel)${RESET}"
        sleep 3
    fi

    # 3. Check for required tools
    if ! command -v curl >/dev/null 2>&1; then
        echo -e "${RED}${BOLD}Error:${RESET}${RED} 'curl' is required. Aborting.${RESET}"
        exit 1
    fi

    if ! command -v sha256sum >/dev/null 2>&1; then
        echo -e "${RED}${BOLD}Error:${RESET}${RED} 'sha256sum' is required. Aborting.${RESET}"
        exit 1
    fi

    # 4. Detect Architecture
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64)  BIN_ARCH="amd64" ;;
        aarch64) BIN_ARCH="arm64" ;;
        arm64)   BIN_ARCH="arm64" ;;
        *)
            echo -e "${RED}${BOLD}Error:${RESET}${RED} Unsupported architecture ($ARCH)${RESET}"
            exit 1
            ;;
    esac

    # 5. Fetch Latest Release Version
    echo -e "${CYAN}==> Fetching latest release info from GitHub...${RESET}"
    LATEST_TAG=$(curl -s https://api.github.com/repos/AndReicscs/HoneyWire/releases | grep -o '"tag_name": "wizard/[^"]*' | head -n 1 | cut -d'"' -f4)

    if [ -z "$LATEST_TAG" ]; then
        echo -e "${RED}${BOLD}Error:${RESET}${RED} Could not determine the latest Wizard version from GitHub. Aborting.${RESET}"
        exit 1
    fi

    # 6. Variables
    BASE_URL="https://github.com/AndReicscs/HoneyWire/releases/download/$LATEST_TAG"
    BINARY_NAME="honeywire-linux-$BIN_ARCH"
    INSTALL_PATH="/usr/local/bin/honeywire"

    # 7. Secure Temp Dir
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT
    cd "$TMP_DIR"

    # 8. Download
    echo -e "${CYAN}==> Downloading ${BOLD}$LATEST_TAG${RESET}${CYAN}...${RESET}"
    curl -fsSL "$BASE_URL/$BINARY_NAME" -o "$BINARY_NAME"

    echo -e "${CYAN}==> Downloading checksums...${RESET}"
    curl -fsSL "$BASE_URL/checksums.txt" -o checksums.txt

    # 9. Strict Checksum Verification
    echo -e "${CYAN}==> Verifying cryptographic signature...${RESET}"
    EXPECTED_HASH=$(grep " $BINARY_NAME\$" checksums.txt | awk '{print $1}')

    if [ -z "$EXPECTED_HASH" ]; then
        echo -e "${RED}${BOLD}Error:${RESET}${RED} Binary not found in checksums.txt! Aborting.${RESET}"
        exit 1
    fi

    if ! echo "$EXPECTED_HASH  $BINARY_NAME" | sha256sum -c - >/dev/null 2>&1; then
        echo -e "${RED}${BOLD}Error:${RESET}${RED} Checksum verification failed!${RESET}"
        exit 1
    fi
    echo -e "${GREEN}✓ Checksum valid.${RESET}"

    # 10. Install
    echo -e "${CYAN}==> Installing to ${DIM}$INSTALL_PATH${RESET}${CYAN} (requires sudo)...${RESET}"
    SUDO=""
    if [ "$(id -u)" -ne 0 ]; then
        SUDO="sudo"
    fi

    $SUDO mv "$BINARY_NAME" "$INSTALL_PATH"
    $SUDO chmod +x "$INSTALL_PATH"

    # 11. Auto-Provisioning (The Magic Step)
    if [ -n "$LINK_URL" ]; then
        echo -e "\n${CYAN}${BOLD}==> Provisioning Node...${RESET}"
        # /dev/tty to force the binary to read from the user's keyboard
        if [ -n "$API_KEY" ]; then
            $SUDO $INSTALL_PATH --link "$LINK_URL" --api-key "$API_KEY" < /dev/tty
        else
            $SUDO $INSTALL_PATH --link "$LINK_URL" < /dev/tty
        fi
        echo -e "\n${GREEN}${BOLD}==> Node Successfully Deployed!${RESET}"
    else
        echo -e "\n${GREEN}${BOLD}==> Installation Complete!${RESET}"
        echo -e "You can now run the tool by typing: ${MAGENTA}${BOLD}honeywire${RESET}\n"
    fi
}

# Execute the function and pass all arguments to it
do_install "$@"