#!/bin/bash
# IBM i Plugin Registration Script
# This script builds and registers the IBM i plugin with a running Vault server

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory (where this script is located)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Detect if running from custom Vault build or standard installation
PARENT_DIR="$(dirname "$SCRIPT_DIR")"

# Check multiple locations for custom Vault build
VAULT_LOCATIONS=(
    "$PARENT_DIR/bin/vault"                                    # Parent directory
    "$HOME/Documents/vault/bin/vault"                          # Common location
    "$(dirname "$PARENT_DIR")/vault/bin/vault"                # Sibling directory
)

VAULT_BIN=""
for location in "${VAULT_LOCATIONS[@]}"; do
    if [ -f "$location" ]; then
        VAULT_BIN="$location"
        VAULT_DIR="$(dirname "$(dirname "$location")")"
        PLUGIN_DIR="${PLUGIN_DIR:-$VAULT_DIR/plugins}"
        IS_CUSTOM_BUILD=true
        echo -e "${BLUE}Detected custom Vault build at $VAULT_DIR${NC}"
        break
    fi
done

# If no custom build found, use system Vault
if [ -z "$VAULT_BIN" ]; then
    VAULT_BIN="vault"
    PLUGIN_DIR="${PLUGIN_DIR:-/opt/vault/plugins}"
    IS_CUSTOM_BUILD=false
    echo -e "${BLUE}Using system Vault installation${NC}"
fi

# Configuration
PLUGIN_NAME="vault-plugin-secrets-ibmi"
VAULT_ADDR="${VAULT_ADDR:-http://127.0.0.1:8200}"
VAULT_TOKEN="${VAULT_TOKEN:-root}"

echo -e "${GREEN}=== HashiCorp-IBMi-plugin Registration ===${NC}\n"
echo -e "${BLUE}Configuration:${NC}"
echo -e "  Vault Binary:    $VAULT_BIN"
echo -e "  Plugin Directory: $PLUGIN_DIR"
echo -e "  Vault Address:   $VAULT_ADDR"
echo -e "  Custom Build:    $IS_CUSTOM_BUILD"
echo ""

# Step 1: Check if Vault is accessible
echo -e "${YELLOW}Step 1: Checking Vault connectivity...${NC}"
export VAULT_ADDR
export VAULT_TOKEN

if ! $VAULT_BIN status > /dev/null 2>&1; then
    echo -e "${RED}Error: Cannot connect to Vault at $VAULT_ADDR${NC}"
    echo "Please ensure:"
    echo "  1. Vault server is running (try: ./restart-vault-ui.sh)"
    echo "  2. VAULT_ADDR is set correctly: export VAULT_ADDR='http://127.0.0.1:8200'"
    echo "  3. VAULT_TOKEN is set: export VAULT_TOKEN='root'"
    exit 1
fi
echo -e "${GREEN}✓ Vault is accessible${NC}\n"

# Step 2: Build the plugin
echo -e "${YELLOW}Step 2: Building plugin...${NC}"
cd "$SCRIPT_DIR"
make build
if [ ! -f "$PLUGIN_NAME" ]; then
    echo -e "${RED}Error: Plugin binary not found after build${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Plugin built successfully: $PLUGIN_NAME${NC}\n"

# Step 3: Copy plugin to plugin directory
echo -e "${YELLOW}Step 3: Copying plugin to $PLUGIN_DIR...${NC}"
if [ ! -d "$PLUGIN_DIR" ]; then
    echo -e "${YELLOW}Plugin directory doesn't exist. Creating it...${NC}"
    mkdir -p "$PLUGIN_DIR"
fi

cp "$PLUGIN_NAME" "$PLUGIN_DIR/"
chmod 755 "$PLUGIN_DIR/$PLUGIN_NAME"
echo -e "${GREEN}✓ Plugin copied to $PLUGIN_DIR${NC}\n"

# Step 4: Calculate SHA256
echo -e "${YELLOW}Step 4: Calculating SHA256 checksum...${NC}"
SHA256=$(shasum -a 256 "$PLUGIN_DIR/$PLUGIN_NAME" | cut -d' ' -f1)
echo -e "  SHA256: ${BLUE}$SHA256${NC}"
echo -e "${GREEN}✓ SHA256 calculated${NC}\n"

# Step 5: Register plugin
echo -e "${YELLOW}Step 5: Registering plugin with Vault...${NC}"
$VAULT_BIN plugin register \
    -sha256="$SHA256" \
    -command="$PLUGIN_NAME" \
    secret \
    "$PLUGIN_NAME"
echo -e "${GREEN}✓ Plugin registered${NC}\n"

# Step 6: Verify registration
echo -e "${YELLOW}Step 6: Verifying plugin registration...${NC}"
if $VAULT_BIN plugin list secret | grep -q "$PLUGIN_NAME"; then
    echo -e "${GREEN}✓ Plugin verified in catalog${NC}\n"
else
    echo -e "${RED}Error: Plugin not found in catalog${NC}"
    exit 1
fi

# Step 7: Enable plugin
echo -e "${YELLOW}Step 7: Enabling plugin at path 'ibmi'...${NC}"
if $VAULT_BIN secrets list | grep -q "ibmi/"; then
    echo -e "${YELLOW}Plugin already enabled at 'ibmi/', skipping...${NC}"
else
    $VAULT_BIN secrets enable \
        -path="ibmi" \
        -plugin-name="$PLUGIN_NAME" \
        plugin
    echo -e "${GREEN}✓ Plugin enabled at 'ibmi/'${NC}"
fi

# Step 8: Copy vault_crud.py if it exists
echo -e "${YELLOW}Step 8: Checking for vault_crud.py...${NC}"
if [ -f "$PARENT_DIR/vault_crud.py" ]; then
    cp "$PARENT_DIR/vault_crud.py" "$PLUGIN_DIR/"
    chmod 755 "$PLUGIN_DIR/vault_crud.py"
    echo -e "${GREEN}✓ vault_crud.py copied to plugin directory${NC}\n"
    PYTHON_SCRIPT_PATH="$PLUGIN_DIR/vault_crud.py"
elif [ -f "$SCRIPT_DIR/../vault_crud.py" ]; then
    cp "$SCRIPT_DIR/../vault_crud.py" "$PLUGIN_DIR/"
    chmod 755 "$PLUGIN_DIR/vault_crud.py"
    echo -e "${GREEN}✓ vault_crud.py copied to plugin directory${NC}\n"
    PYTHON_SCRIPT_PATH="$PLUGIN_DIR/vault_crud.py"
else
    echo -e "${YELLOW}vault_crud.py not found, you'll need to specify the path manually${NC}\n"
    PYTHON_SCRIPT_PATH="/path/to/vault_crud.py"
fi

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}  Registration Complete!${NC}"
echo -e "${GREEN}========================================${NC}\n"

echo -e "${BLUE}Plugin Information:${NC}"
echo -e "  Name:      $PLUGIN_NAME"
echo -e "  Location:  $PLUGIN_DIR/$PLUGIN_NAME"
echo -e "  SHA256:    $SHA256"
echo -e "  Path:      ibmi/"
echo ""

echo -e "${YELLOW}Next Steps:${NC}"
echo ""
echo -e "${BLUE}1. Configure the plugin:${NC}"
echo -e "   $VAULT_BIN write ibmi/config \\"
echo -e "     python_script_path=\"$PYTHON_SCRIPT_PATH\" \\"
echo -e "     python_executable=\"python3\" \\"
echo -e "     host=\"your-ibmi-host.com\" \\"
echo -e "     port=\"8076\" \\"
echo -e "     username=\"admin_user\" \\"
echo -e "     password=\"admin_password\""
echo ""
echo -e "${BLUE}2. Create a role:${NC}"
echo -e "   $VAULT_BIN write ibmi/role/developer \\"
echo -e "     ttl=720h \\"
echo -e "     max_ttl=2160h \\"
echo -e "     default_ttl_days=30"
echo ""
echo -e "${BLUE}3. Generate credentials:${NC}"
echo -e "   $VAULT_BIN read ibmi/creds/developer"
echo ""
echo -e "${YELLOW}Tip:${NC} Your Vault is running in dev mode with token: ${BLUE}root${NC}"
echo ""
