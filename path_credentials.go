package ibmiauth

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/IBM/HashiCorp-IBMi-plugin/internal/ibmihelper"
)

func (b *backend) PathCredentials() *framework.Path {
	return &framework.Path{
		Pattern: "creds/" + framework.GenericNameRegex("name"),
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeString,
				Description: "Name of the role to generate credentials for",
			},
		},
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ReadOperation: b.pathCredentialsRead,
		},
		HelpSynopsis:    "Generate IBM i credentials for a specific role",
		HelpDescription: "This path generates temporary IBM i user credentials based on the specified role configuration.",
	}
}

func (b *backend) pathCredentialsRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleName := d.Get("name").(string)
	if roleName == "" {
		return logical.ErrorResponse("missing role name"), logical.ErrInvalidRequest
	}

	b.Logger().Debug("generating credentials for role", "role", roleName)

	// Get the role
	b.l.RLock()
	role, err := b.role(ctx, req.Storage, roleName)
	b.l.RUnlock()

	if err != nil {
		return nil, fmt.Errorf("error retrieving role: %w", err)
	}

	if role == nil {
		return logical.ErrorResponse("role not found"), logical.ErrInvalidRequest
	}

	// Get the configuration
	entry, err := req.Storage.Get(ctx, configPath)
	if err != nil {
		b.Logger().Error("failed to read configuration from storage", "error", err)
		return nil, fmt.Errorf("failed to read configuration: %w", err)
	}

	if entry == nil {
		b.Logger().Error("configuration not found in storage")
		return logical.ErrorResponse("plugin not configured"), logical.ErrInvalidRequest
	}

	var config ibmiConfig
	if err := entry.DecodeJSON(&config); err != nil {
		b.Logger().Error("failed to decode configuration", "error", err)
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	// Generate username using template
	username, err := ibmihelper.GenerateUsername(roleName, config.UsernameTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate username: %w", err)
	}

	// Generate IBM i-compatible password
	password, err := ibmihelper.GenerateIBMiPassword(20)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	// Calculate TTL in days for IBM i
	ttlDays := role.DefaultTTLDays
	if role.TTL > 0 {
		ttlDays = int(role.TTL.Hours() / 24)
		if ttlDays < 1 {
			ttlDays = 1
		}
	}

	// Create user via Python script
	result, err := ibmihelper.ExecutePythonFunction(
		config.PythonExecutable,
		config.PythonScriptPath,
		config.Host,
		config.Port,
		config.Username,
		config.Password,
		"CreateTempUsrprf",
		username,
		password,
		ttlDays,
	)

	if err != nil {
		b.Logger().Error("failed to create IBM i user", "error", err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if !result.Success {
		b.Logger().Error("Python script failed to create user", "error", result.Error)
		return nil, fmt.Errorf("failed to create user: %s", result.Error)
	}

	b.Logger().Info("successfully created IBM i user", "username", username, "ttl_days", ttlDays)

	// Calculate lease duration
	leaseDuration := role.TTL
	if leaseDuration == 0 {
		leaseDuration = time.Duration(ttlDays) * 24 * time.Hour
	}

	// Return the credentials
	resp := b.Secret(ibmiCredentialType).Response(map[string]interface{}{
		"username": username,
		"password": password,
	}, map[string]interface{}{
		"username":           username,
		"role":               roleName,
		"python_script_path": config.PythonScriptPath,
		"python_executable":  config.PythonExecutable,
		"host":               config.Host,
		"port":               config.Port,
		"admin_username":     config.Username,
		"admin_password":     config.Password,
	})

	resp.Secret.TTL = leaseDuration
	resp.Secret.MaxTTL = role.MaxTTL

	return resp, nil
}

// ibmiCredential returns the secret definition for IBM i credentials
func (b *backend) ibmiCredential() *framework.Secret {
	return &framework.Secret{
		Type: ibmiCredentialType,
		Fields: map[string]*framework.FieldSchema{
			"username": {
				Type:        framework.TypeString,
				Description: "IBM i username",
			},
			"password": {
				Type:        framework.TypeString,
				Description: "IBM i password",
			},
		},
		Renew:  b.credentialRenew,
		Revoke: b.credentialRevoke,
	}
}

const ibmiCredentialType = "ibmi_credential"

// credentialRenew handles renewal of IBM i credentials
func (b *backend) credentialRenew(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("renewing IBM i credential")

	// Get the role name from internal data
	roleNameRaw, ok := req.Secret.InternalData["role"]
	if !ok {
		return nil, fmt.Errorf("role name not found in secret internal data")
	}
	roleName := roleNameRaw.(string)

	// Get the role to check TTL limits
	b.l.RLock()
	role, err := b.role(ctx, req.Storage, roleName)
	b.l.RUnlock()

	if err != nil {
		return nil, fmt.Errorf("error retrieving role: %w", err)
	}

	if role == nil {
		return nil, fmt.Errorf("role not found")
	}

	// Use framework's LeaseExtend to handle renewal
	resp := &logical.Response{Secret: req.Secret}
	resp.Secret.TTL = role.TTL
	resp.Secret.MaxTTL = role.MaxTTL

	return resp, nil
}

// credentialRevoke handles revocation of IBM i credentials
func (b *backend) credentialRevoke(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	usernameRaw, ok := req.Secret.InternalData["username"]
	if !ok {
		return nil, fmt.Errorf("username not found in secret internal data")
	}
	username := usernameRaw.(string)

	b.Logger().Debug("revoking IBM i credential", "username", username)

	// Get configuration from internal data
	pythonExecutable := req.Secret.InternalData["python_executable"].(string)
	pythonScriptPath := req.Secret.InternalData["python_script_path"].(string)
	host := req.Secret.InternalData["host"].(string)
	port := req.Secret.InternalData["port"].(string)
	adminUsername := req.Secret.InternalData["admin_username"].(string)
	adminPassword := req.Secret.InternalData["admin_password"].(string)

	// Revoke user via Python script
	result, err := ibmihelper.ExecutePythonFunction(
		pythonExecutable,
		pythonScriptPath,
		host,
		port,
		adminUsername,
		adminPassword,
		"RevokeUsrprf",
		username,
	)

	if err != nil {
		// Log error but don't fail revocation if user doesn't exist
		b.Logger().Warn("failed to revoke IBM i user", "username", username, "error", err)
		return nil, nil
	}

	if !result.Success {
		// Log error but don't fail revocation if user doesn't exist
		b.Logger().Warn("Python script failed to revoke user", "username", username, "error", result.Error)
		return nil, nil
	}

	b.Logger().Info("successfully revoked IBM i user", "username", username)
	return nil, nil
}
