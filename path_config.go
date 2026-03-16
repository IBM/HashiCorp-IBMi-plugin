package ibmiauth

import (
	"context"
	"reflect"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	configHelpSynopsis    = `Configure the IBM i plugin by specifying connection details and Python script path.`
	configHelpDescription = `
	The IBM i plugin requires configuration to connect to IBM i systems and execute user management operations.
	This configuration includes the Python script path, Python executable, and IBM i connection details.
	`
	configPath = "config"
)

// ibmiConfig holds the configuration data for the IBM i plugin
type ibmiConfig struct {
	PythonScriptPath string `json:"python_script_path"`
	PythonExecutable string `json:"python_executable"`
	Host             string `json:"host"`
	Port             string `json:"port"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	UsernameTemplate string `json:"username_template"`
}

func (b *backend) PathConfig() *framework.Path {
	return &framework.Path{
		Pattern: "config",
		Fields: map[string]*framework.FieldSchema{
			"python_script_path": {
				Type:        framework.TypeString,
				Description: "Path to the Python script (vault_crud.py) for IBM i operations.",
				Required:    true,
			},
			"python_executable": {
				Type:        framework.TypeString,
				Description: "Python executable to use (default: python3).",
				Default:     "python3",
			},
			"host": {
				Type:        framework.TypeString,
				Description: "IBM i host address.",
				Required:    true,
			},
			"port": {
				Type:        framework.TypeString,
				Description: "IBM i port (default: 8076 for Mapepire).",
				Default:     "8076",
			},
			"username": {
				Type:        framework.TypeString,
				Description: "IBM i admin username for managing user profiles.",
				Required:    true,
			},
			"password": {
				Type:        framework.TypeString,
				Description: "IBM i admin password.",
				Required:    true,
				DisplayAttrs: &framework.DisplayAttributes{
					Sensitive: true,
				},
			},
			"username_template": {
				Type:        framework.TypeString,
				Description: "Template for generating usernames (default: V{{.RoleName | truncate 4 | uppercase}}{{random 4}} | truncate 10 | uppercase).",
			},
		},
		Operations: map[logical.Operation]framework.OperationHandler{
			logical.UpdateOperation: &framework.PathOperation{
				Callback:    b.pathConfigWrite,
				Description: "Configure the IBM i plugin connection and settings.",
				DisplayAttrs: &framework.DisplayAttributes{
					Description: "Update IBM i plugin configuration.",
				},
			},
			logical.ReadOperation: &framework.PathOperation{
				Callback:    b.pathConfigRead,
				Description: "Read the IBM i plugin configuration.",
				DisplayAttrs: &framework.DisplayAttributes{
					Description: "Read IBM i plugin configuration.",
				},
			},
		},
		HelpSynopsis:    configHelpSynopsis,
		HelpDescription: configHelpDescription,
	}
}

func (b *backend) pathConfigWrite(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	pythonScriptPath := d.Get("python_script_path").(string)
	if pythonScriptPath == "" {
		return logical.ErrorResponse("python_script_path is required"), logical.ErrInvalidRequest
	}

	host := d.Get("host").(string)
	if host == "" {
		return logical.ErrorResponse("host is required"), logical.ErrInvalidRequest
	}

	username := d.Get("username").(string)
	if username == "" {
		return logical.ErrorResponse("username is required"), logical.ErrInvalidRequest
	}

	password := d.Get("password").(string)
	if password == "" {
		return logical.ErrorResponse("password is required"), logical.ErrInvalidRequest
	}

	pythonExecutable := d.Get("python_executable").(string)
	if pythonExecutable == "" {
		pythonExecutable = "python3"
	}

	port := d.Get("port").(string)
	if port == "" {
		port = "8076"
	}

	usernameTemplate := d.Get("username_template").(string)

	config := &ibmiConfig{
		PythonScriptPath: pythonScriptPath,
		PythonExecutable: pythonExecutable,
		Host:             host,
		Port:             port,
		Username:         username,
		Password:         password,
		UsernameTemplate: usernameTemplate,
	}

	entry, err := logical.StorageEntryJSON(configPath, config)
	if err != nil {
		return logical.ErrorResponse("failed to create storage entry: %v", err), err
	}

	err = req.Storage.Put(ctx, entry)
	if err != nil {
		return logical.ErrorResponse("failed to write configuration to storage: %v", err), err
	}

	return nil, nil
}

func (b *backend) pathConfigRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	entry, err := req.Storage.Get(ctx, configPath)
	if err != nil {
		return logical.ErrorResponse("failed to read configuration from storage: %v", err), err
	}

	if entry == nil {
		return logical.ErrorResponse("configuration not found"), logical.ErrInvalidRequest
	}

	var config ibmiConfig
	if err := entry.DecodeJSON(&config); err != nil {
		return logical.ErrorResponse("failed to decode configuration: %v", err), err
	}

	// Use reflection to get field names from struct tags
	cnf := reflect.TypeOf(config)
	
	resp := &logical.Response{
		Data: map[string]interface{}{},
	}

	// Add all fields except password (sensitive)
	for i := 0; i < cnf.NumField(); i++ {
		field := cnf.Field(i)
		jsonTag := field.Tag.Get("json")
		
		// Skip password field in response
		if jsonTag == "password" {
			continue
		}

		// Get the value using reflection
		val := reflect.ValueOf(config).Field(i).Interface()
		resp.Data[jsonTag] = val
	}

	return resp, nil
}
