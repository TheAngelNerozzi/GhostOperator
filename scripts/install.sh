# <ghost-bash-installer>
# GhostOperator (GO) Installer for Linux / macOS
# This script downloads the latest release from GitHub.

REPO="TheAngelNerozzi/GhostOperator"
INSTALL_DIR="$HOME/.ghostoperator"
mkdir -p "$INSTALL_DIR"

echo "Checking for latest GhostOperator release..."
RELEASE_DATA=$(curl -s "https://api.github.com/repos/$REPO/releases/latest")
OS_TYPE=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH_TYPE=$(uname -m)

if [ "$ARCH_TYPE" = "x86_64" ]; then
    ARCH_TYPE="amd64"
elif [ "$ARCH_TYPE" = "aarch64" ] || [ "$ARCH_TYPE" = "arm64" ]; then
    ARCH_TYPE="arm64"
fi

ASSET_NAME="ghost-${OS_TYPE}-${ARCH_TYPE}"
DOWNLOAD_URL=$(echo "$RELEASE_DATA" | grep "browser_download_url" | grep "$ASSET_NAME" | cut -d '"' -f 4)

if [ -z "$DOWNLOAD_URL" ]; then
    echo "❌ Error: Could not find a binary for $OS_TYPE-$ARCH_TYPE."
    exit 1
fi

echo "Downloading GhostOperator from GitHub..."
curl -L "$DOWNLOAD_URL" -o "$INSTALL_DIR/ghost"
chmod +x "$INSTALL_DIR/ghost"

# Add to PATH
SHELL_RC="$HOME/.bashrc"
if [ "$SHELL" = "/bin/zsh" ]; then SHELL_RC="$HOME/.zshrc"; fi

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_RC"
    echo "✅ GhostOperator added to $SHELL_RC. Please restart your shell."
fi

echo "GhostOperator (GO) installed successfully! 👻"
echo "Try running: ghost --version"
