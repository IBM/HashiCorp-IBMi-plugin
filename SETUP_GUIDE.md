# HashiCorp-IBMi-plugin - Complete Setup Guide

This guide walks you through setting up Vault and registering the IBM i plugin step by step.

---

## Prerequisites

Before you begin, ensure you have:
- ✅ Vault repository cloned at: `~//vault`
- ✅ Vault binary built at: `~//vault/bin/vault`
- ✅ Go 1.25+ installed
- ✅ Python 3.x installed
- ✅ pip3 install below packages : 
   - ✅ mapepire_python
   - ✅ python-dotenv
- ✅ IBM i system accessible
- ✅ IBM i system with Mapepire Server installed (To setup mapepire server refer this doc : https://mapepire-ibmi.github.io/guides/sysadmin/)
- ✅ IBM i User Profile with `*SECADM Special Authority`



---

## Part 1: Start Vault Server

### Step 1: Create Vault Configuration File

Create a file named `vault-config.hcl` in your project directory:

```bash
cat > ~//HashiCorp-IBMi-plugin/vault-config.hcl << 'EOF'
# Vault Configuration
storage "file" {
  path = "~//vault/data"
}

listener "tcp" {
  address     = "127.0.0.1:8200"
  tls_disable = 1
}

api_addr = "http://127.0.0.1:8200"
plugin_directory = "~//vault/plugins"
ui = true
log_level = "info"
disable_mlock = true
EOF

```

Or manually create the file with this content:

```hcl
# Vault Configuration
storage "file" {
  path = "~//vault/data"
}

listener "tcp" {
  address     = "127.0.0.1:8200"
  tls_disable = 1
}

api_addr = "http://127.0.0.1:8200"
plugin_directory = "~//vault/plugins"
ui = true
log_level = "info"
disable_mlock = true
```

**⚠️ IMPORTANT:** Use absolute paths (not `~`) to avoid Vault creating a literal `~` directory.

**What this does:**
- Sets up file-based storage for Vault data
- Configures Vault to listen on port 8200
- **Enables plugin directory** (required for external plugins)
- Enables the web UI

---

### Step 2: Create Required Directories

```bash
mkdir -p ~//vault/data
mkdir -p ~//vault/plugins
```

**What this does:**
- Creates directory for Vault's data storage
- Creates directory for plugin binaries

---

### Step 3: Stop Any Running Vault Instance

```bash
pkill -9 vault
```

**What this does:**
- Stops any existing Vault server to avoid conflicts

---

### Step 4: Start Vault Server

```bash
cd ~//vault 
./bin/vault server -config=~//HashiCorp-IBMi-plugin/vault-config.hcl > vault.log 2>&1 &
```

**What this does:**
- Starts Vault in the background
- Uses your configuration file
- Logs output to `vault.log`

**Verify it's running:**
```bash
ps aux | grep "vault server"
```

You should see a process running.

---

### Step 5: Set Environment Variables

```bash
export VAULT_ADDR='http://127.0.0.1:8200'
```

**What this does:**
- Tells Vault CLI where to connect

---

### Step 6: Initialize Vault (First Time Only)

```bash
~//vault/bin/vault operator init -key-shares=1 -key-threshold=1 | tee ~//vault/vault-keys.txt
```

**What this does:**
- Initializes Vault's encryption
- Generates an unseal key and root token
- Saves them to `vault-keys.txt`

**IMPORTANT:** You'll see output like this:

```
Unseal Key 1: B2IE/izRscl1TBzaCN+JgL2ez+NWtmo80JKdlR/jZD4=
Initial Root Token: hvs.7TeFygOSlvUUACuVnIEzfGK
```

**📝 SAVE THESE IMMEDIATELY!** They are saved to `~//vault/vault-keys.txt`

---

**⚠️ If You Get "Vault is already initialized" Error:**

This means Vault was initialized before. You have two options:

**Option A: Use Existing Keys (If You Have Them)**
```bash
# Check if keys were saved
cat ~//vault/vault-keys.txt
```

If the file contains your unseal key and root token, skip to Step 7 and use those keys.

**If the file is empty or doesn't exist, you MUST use Option B below.**

**Option B: Reset Vault (Deletes All Data)**

If you don't have the keys or want to start fresh:

1. **Stop Vault:**
   ```bash
   pkill -9 vault
   ```

2. **Delete ALL Vault data (including any literal `~` directory):**
   ```bash
   rm -rf ~//vault/data
   rm -rf ~//vault/~
   ```
   
   **Note:** The second command removes a literal `~` directory that Vault may have created if the config used `/Users/mohammedyaseenali/path` instead of absolute paths.

3. **Recreate data directory:**
   ```bash
   mkdir -p ~//vault/data
   ```

4. **Verify your config uses absolute paths (not `~`):**
   ```bash
   cat ~//HashiCorp-IBMi-plugin/vault-config.hcl
   ```
   
   Ensure paths look like `~//vault/data` NOT `~//vault/data`

5. **Start Vault again:**
   ```bash
   cd ~//vault
   ./bin/vault server -config=~//HashiCorp-IBMi-plugin/vault-config.hcl > vault.log 2>&1 &
   ```

6. **Set environment:**
   ```bash
   export VAULT_ADDR='http://127.0.0.1:8200'
   ```

7. **Initialize with new keys:**
   ```bash
   ~//vault/bin/vault operator init -key-shares=1 -key-threshold=1 | tee ~//vault/vault-keys.txt
   ```

8. **Continue with Step 7 using the NEW keys**

**⚠️ WARNING:** Option B deletes ALL Vault data including secrets, policies, and configurations!

---

### Step 7: Unseal Vault

Every time Vault starts, it's "sealed" and needs to be unsealed:

```bash
~//vault/bin/vault operator unseal <YOUR-UNSEAL-KEY>
```

**Example:**
```bash
~//vault/bin/vault operator unseal B2IE/izRscl1TDyaCN+JgL2ez+NWtmo80JKdlR/jZD4=
```

**What this does:**
- Unlocks Vault so it can be used

**Verify it's unsealed:**
```bash
~//vault/bin/vault status
```

You should see `Sealed: false`

---

### Step 8: Login to Vault

```bash
export VAULT_TOKEN='<YOUR-ROOT-TOKEN>'
~//vault/bin/vault login <YOUR-ROOT-TOKEN>
```

**Example:**
```bash
export VAULT_TOKEN='hvs.EXAMPLE_TOKEN_REPLACE_WITH_YOUR_ACTUAL_TOKEN'
~//vault/bin/vault login hvs.EXAMPLE_TOKEN_REPLACE_WITH_YOUR_ACTUAL_TOKEN
```

**What this does:**
- Authenticates you with Vault
- Allows you to perform administrative tasks

---

## Part 2: Register IBM i Plugin

### Step 9: Build the Plugin

```bash
cd ~//HashiCorp-IBMi-plugin
make build
```

**What this does:**
- Compiles the IBM i plugin for your Mac M1
- Creates binary: `vault-plugin-secrets-ibmi`

**Verify the build:**
```bash
ls -lh vault-plugin-secrets-ibmi
```

You should see the binary file.

---

### Step 10: Copy Plugin to Plugin Directory

```bash
cp vault-plugin-secrets-ibmi ~//vault/plugins/
chmod 755 ~//vault/plugins/vault-plugin-secrets-ibmi
```

**What this does:**
- Copies plugin to Vault's plugin directory
- Makes it executable

---

### Step 11: Copy Python Script

```bash
cp ~//HashiCorp-IBMi-plugin/vault_crud.py ~//vault/plugins/
chmod 755 ~//vault/plugins/vault_crud.py
```

**What this does:**
- Copies the IBM i management script to plugin directory
- Makes it executable

---

### Step 12: Calculate Plugin SHA256

```bash
shasum -a 256 ~//vault/plugins/vault-plugin-secrets-ibmi
```

**What this does:**
- Calculates a security checksum of the plugin
- Vault requires this for security

**Example output:**
```
63c17dd1dba87286c88bca9f52507c540d11d52f8dc890e5980b889cef1b45e8  ~//vault/plugins/vault-plugin-secrets-ibmi
```

**📝 SAVE THE SHA256 VALUE** (the long hex string before the path)

---

### Step 13: Register Plugin with Vault

```bash
~//vault/bin/vault plugin register \
    -sha256=<YOUR-SHA256> \
    -command="vault-plugin-secrets-ibmi" \
    secret \
    vault-plugin-secrets-ibmi
```

**Example:**
```bash
~//vault/bin/vault plugin register \
    -sha256=63c17dd1dba87286c88bca9f52507c540d11d52f8dc890e5980b889cef1b45e8 \
    -command="vault-plugin-secrets-ibmi" \
    secret \
    vault-plugin-secrets-ibmi
```

**What this does:**
- Registers the plugin in Vault's catalog
- Associates it with the SHA256 checksum for security

**Verify registration:**
```bash
~//vault/bin/vault plugin list secret
```

You should see `vault-plugin-secrets-ibmi` in the list.

---

### Step 14: Enable the Plugin

```bash
~//vault/bin/vault secrets enable \
    -path="ibmi" \
    -plugin-name="vault-plugin-secrets-ibmi" \
    plugin
```

**What this does:**
- Activates the plugin at path `ibmi/`
- Makes it available for use

**Verify it's enabled:**
```bash
~//vault/bin/vault secrets list
```

You should see `ibmi/` in the list.

---

## Part 3: Configure and Use the Plugin

### Step 15: Configure IBM i Connection

```bash
~//vault/bin/vault write ibmi/config \
    python_script_path="~//vault/plugins/vault_crud.py" \
    python_executable="python3" \
    host="your-ibmi-host.com" \
    port="8076" \
    username="admin_user" \
    password="admin_password"
```

**Replace these values:**
- `host`: Your IBM i hostname or IP address
- `port`: IBM i port (usually 8076 for Mapepire)
- `username`: IBM i admin username
- `password`: IBM i admin password

**What this does:**
- Configures how the plugin connects to IBM i
- Stores connection details securely in Vault

**Verify configuration:**
```bash
~//vault/bin/vault read ibmi/config
```

---

### Step 16: Create a Role

```bash
~//vault/bin/vault write ibmi/role/developer \
    ttl=720h \
    max_ttl=2160h \
    default_ttl_days=30
```

**What this does:**
- Creates a "developer" role
- Sets credential lifetime to 30 days (720 hours)
- Maximum lifetime is 90 days (2160 hours)

**You can create multiple roles:**
```bash
# Temporary access role (1 day)
~//vault/bin/vault write ibmi/role/temp \
    ttl=24h \
    max_ttl=72h \
    default_ttl_days=1
```

**Verify role:**
```bash
~//vault/bin/vault read ibmi/role/developer
```

---

### Step 17: Generate Credentials

```bash
~//vault/bin/vault read ibmi/creds/developer
```

**What this does:**
- Generates a temporary IBM i user
- Creates a random username and password
- Returns credentials you can use

**Example output:**
```
Key                Value
---                -----
lease_id           ibmi/creds/developer/abc123
lease_duration     720h
lease_renewable    true
password           Aa1@xYz9Bc3Def4Ghi5J
username           VDEV1A2B
```

**Use these credentials to login to IBM i:**
```bash
ssh VDEV1A2B@your-ibmi-host.com
# Password: Aa1@xYz9Bc3Def4Ghi5J
```

---

## Summary of Commands

### Starting Vault (Every Time)
```bash
# 1. Start Vault
cd ~//vault
./bin/vault server -config=~//HashiCorp-IBMi-plugin/vault-config.hcl > vault.log 2>&1 &

# 2. Set environment
export VAULT_ADDR='http://127.0.0.1:8200'

# 3. Unseal (use your saved key)
./bin/vault operator unseal <YOUR-UNSEAL-KEY>

# 4. Login (use your saved token)
export VAULT_TOKEN='<YOUR-ROOT-TOKEN>'
./bin/vault login <YOUR-ROOT-TOKEN>
```

### Using the Plugin (After Setup)
```bash
# Generate credentials
~//vault/bin/vault read ibmi/creds/developer

# List roles
~//vault/bin/vault list ibmi/role

# View role details
~//vault/bin/vault read ibmi/role/developer
```

### Stopping Vault
```bash
pkill -f "vault server"
```

---

## Troubleshooting

### Problem: "Vault is sealed"
**Solution:** Run the unseal command:
```bash
~//vault/bin/vault operator unseal <YOUR-UNSEAL-KEY>
```

### Problem: "Permission denied"
**Solution:** Check file permissions:
```bash
chmod 755 ~//vault/plugins/vault-plugin-secrets-ibmi
chmod 755 ~//vault/plugins/vault_crud.py
```

### Problem: "Plugin not found"
**Solution:** Verify plugin directory in config and that plugin exists:
```bash
ls -la ~//vault/plugins/vault-plugin-secrets-ibmi
```

### Problem: "SHA256 mismatch"
**Solution:** Recalculate and re-register:
```bash
# Calculate new SHA256
shasum -a 256 ~//vault/plugins/vault-plugin-secrets-ibmi

# Re-register with new SHA256
~//vault/bin/vault plugin register -sha256=<NEW-SHA256> -command="vault-plugin-secrets-ibmi" secret vault-plugin-secrets-ibmi
```

### Problem: "Cannot connect to IBM i"
**Solution:** Verify IBM i connection details in config:
```bash
~//vault/bin/vault read ibmi/config
```

---

## Important Files and Locations

| Item | Location |
|------|----------|
| Vault Binary | `~//vault/bin/vault` |
| Vault Config | `~//HashiCorp-IBMi-plugin/vault-config.hcl` |
| Vault Data | `~//vault/data` |
| Plugin Directory | `~//vault/plugins` |
| Plugin Binary | `~//vault/plugins/vault-plugin-secrets-ibmi` |
| Python Script | `~//vault/plugins/vault_crud.py` |
| Vault Keys | `~//vault/vault-keys.txt` |
| Vault Logs | `~//vault/vault.log` |

---

## Security Best Practices

1. **🔐 Protect Your Keys**
   - Store unseal key and root token securely
   - Never commit them to version control
   - Consider using a password manager

2. **🔒 Use TLS in Production**
   - The current config has `tls_disable = 1` for testing
   - Enable TLS for production use

3. **👥 Create Limited Tokens**
   - Don't use root token for daily operations
   - Create tokens with specific policies

4. **📝 Audit Logging**
   - Enable audit logging in production:
   ```bash
   ~//vault/bin/vault audit enable file file_path=/var/log/vault_audit.log
   ```

5. **🔄 Regular Backups**
   - Backup `~//vault/data` regularly
   - Test restore procedures

---

## Next Steps

After setup, you can:
- Create additional roles with different TTLs
- Integrate with your applications
- Set up automated credential rotation
- Configure policies for different user groups
- Enable audit logging
- Set up monitoring

For more information, see:
- [README.md](./README.md) - Plugin overview