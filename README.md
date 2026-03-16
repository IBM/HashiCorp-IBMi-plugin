# HashiCorp-IBMi-plugin
IBM i plugin for Hashicorp Vault
-----
⚠️WORK IN PROGRESS⚠️

![Vault Logo](https://github.com/hashicorp/vault/raw/f22d202cde2018f9455dec755118a9b84586e082/Vault_PrimaryLogo_Black.png)

### What is it: 
This is a HashiCorp Vault secrets plugin that dynamically generates temporary IBM i user profiles with configurable TTLs. The plugin integrates with IBM i systems via Mapepire to create, manage, and revoke user credentials.

### Architecture:

This plugin follows the Vault secrets backend pattern, similar to the database secrets engine. It provides:
- **Dynamic credential generation** - Creates temporary IBM i user profiles on-demand
- **Role-based configuration** - Define roles with specific TTL and policies
- **Automatic revocation** - Removes user profiles when leases expire
- **Python integration** - Uses Mapepire to interact with IBM i systems

### Prerequisites:
- HashiCorp Vault server (CE or Enterprise)
- Python 3.x installed on the Vault server
- IBM i system with Mapepire or similar access
- Python script (`vault_crud.py`) for IBM i user management
- Golang installed locally (for development and building)

### Project Structure:
```
ibmi-plugin/
├── backend.go                    # Plugin initialization and backend setup
├── path_config.go                # Configuration path (connection details)
├── path_role.go                  # Role management (CRUD operations)
├── path_credentials.go           # Credential generation and revocation
├── cmd/
│   └── ibmiauth/
│       └── main.go              # Plugin entry point
├── internal/
│   ├── ibmihelper/
│   │   └── ibmihelper.go       # Helper functions (username generation, Python execution)
│   └── version/
│       └── version.go          # Version information
├── go.mod                       # Go module dependencies
└── README.md                    # This file
```

### Usage:

Please refer to the [Setup Guide](SETUP_GUIDE.md) for detailed installation and configuration instructions.

### Security Features:

1. **IBM i-compatible password generation** - Generates passwords using only valid IBM i characters (A-Z, a-z, 0-9, @, #, $, _)
2. **Seal wrap storage** - Configuration is encrypted using Vault's seal
3. **Automatic credential revocation** - User profiles are automatically deleted when leases expire
4. **Username constraints** - Usernames are limited to 10 characters and uppercase (IBM i requirement)
5. **Configurable TTLs** - Control credential lifetime at the role level

> **Important Note:** The `PWDEXPITV` (Password Expiration Interval) CL command parameter has a minimum value of 1 day. This means temporary user profiles created by this plugin will have passwords that remain valid for at least 24 hours.

### Workflow:

```
User Request → Read Role → Load Config → Generate Username & Password
    ↓
Execute Python Script (CreateTempUsrprf) → Create IBM i User
    ↓
Return Credentials with Lease → User Uses Credentials
    ↓
Lease Expires → Execute Python Script (RevokeUsrprf) → Delete IBM i User
```

### Development:

To run tests:
```bash
go test ./...
```

To build for different platforms:
```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o vault-plugin-secrets-ibmi-linux cmd/ibmiauth/main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o vault-plugin-secrets-ibmi-darwin cmd/ibmiauth/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o vault-plugin-secrets-ibmi.exe cmd/ibmiauth/main.go
```