package ibmiauth

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	rolePrefix = "role/"
)

// roleStorageEntry holds the configuration for a role
type roleStorageEntry struct {
	// TTL for generated credentials (in days for IBM i)
	TTL time.Duration `json:"ttl"`
	// MaxTTL for generated credentials
	MaxTTL time.Duration `json:"max_ttl"`
	// Default TTL in days for IBM i user profiles
	DefaultTTLDays int `json:"default_ttl_days"`
}

// role retrieves a role from storage
func (b *backend) role(ctx context.Context, s logical.Storage, name string) (*roleStorageEntry, error) {
	raw, err := s.Get(ctx, fmt.Sprintf("%s%s", rolePrefix, strings.ToLower(name)))
	if err != nil {
		return nil, err
	}

	if raw == nil {
		return nil, nil
	}

	role := &roleStorageEntry{}
	if err := json.Unmarshal(raw.Value, role); err != nil {
		return nil, err
	}

	return role, nil
}

func (b *backend) PathsRoleList() *framework.Path {
	return &framework.Path{
		Pattern: "role/?",
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.ListOperation: b.pathRoleList,
		},
		DisplayAttrs: &framework.DisplayAttributes{
			OperationPrefix: "ibmiauth",
		},
	}
}

func (b *backend) PathRoleCRUD() *framework.Path {
	return &framework.Path{
		Pattern: "role/" + framework.GenericNameRegex("name"),
		Fields: map[string]*framework.FieldSchema{
			"name": {
				Type:        framework.TypeString,
				Description: "Name of the role",
			},
			"ttl": {
				Type:        framework.TypeDurationSecond,
				Description: "Default TTL for credentials generated under this role.",
				Default:     "30d",
			},
			"max_ttl": {
				Type:        framework.TypeDurationSecond,
				Description: "Maximum TTL for credentials generated under this role.",
				Default:     "90d",
			},
			"default_ttl_days": {
				Type:        framework.TypeInt,
				Description: "Default TTL in days for IBM i user profiles (default: 30).",
				Default:     30,
			},
		},
		ExistenceCheck: b.pathRoleExistenceCheck,
		Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.CreateOperation: b.pathRoleCreateUpdate,
			logical.UpdateOperation: b.pathRoleCreateUpdate,
			logical.ReadOperation:   b.pathRoleRead,
			logical.DeleteOperation: b.pathRoleDelete,
		},
		HelpSynopsis:    "Manage roles for IBM i credential generation",
		HelpDescription: "This path allows you to create, read, update, and delete roles that define how IBM i credentials are generated.",
	}
}

func (b *backend) pathRoleList(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	b.Logger().Debug("reading role list")

	b.l.RLock()
	defer b.l.RUnlock()

	roles, err := req.Storage.List(ctx, "role/")
	if err != nil {
		b.Logger().Error("failure to list roles", "error", err)
		return nil, err
	}

	return logical.ListResponse(roles), nil
}

func (b *backend) pathRoleRead(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleName := d.Get("name")
	if roleName == "" {
		return logical.ErrorResponse("missing role name"), nil
	}

	b.l.RLock()
	defer b.l.RUnlock()

	role, err := b.role(ctx, req.Storage, roleName.(string))
	if err != nil {
		return nil, err
	}

	if role == nil {
		return nil, nil
	}

	resp := map[string]interface{}{
		"ttl":              int64(role.TTL.Seconds()),
		"max_ttl":          int64(role.MaxTTL.Seconds()),
		"default_ttl_days": role.DefaultTTLDays,
	}

	return &logical.Response{Data: resp}, nil
}

func (b *backend) pathRoleCreateUpdate(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	roleName := d.Get("name").(string)
	if roleName == "" {
		return logical.ErrorResponse("missing role name"), nil
	}

	b.l.Lock()
	defer b.l.Unlock()

	// Check if role exists
	role, err := b.role(ctx, req.Storage, roleName)
	if err != nil {
		return nil, err
	}

	// Create new role if it doesn't exist
	if role == nil && req.Operation == logical.CreateOperation {
		role = &roleStorageEntry{}
	} else if role == nil {
		return nil, fmt.Errorf("trying to update a role that does not exist")
	}

	// Get TTL
	if ttlRaw, ok := d.GetOk("ttl"); ok {
		role.TTL = time.Duration(ttlRaw.(int)) * time.Second
	} else if req.Operation == logical.CreateOperation {
		role.TTL = 30 * 24 * time.Hour // 30 days default
	}

	// Get MaxTTL
	if maxTTLRaw, ok := d.GetOk("max_ttl"); ok {
		role.MaxTTL = time.Duration(maxTTLRaw.(int)) * time.Second
	} else if req.Operation == logical.CreateOperation {
		role.MaxTTL = 90 * 24 * time.Hour // 90 days default
	}

	// Get default TTL days for IBM i
	if ttlDaysRaw, ok := d.GetOk("default_ttl_days"); ok {
		role.DefaultTTLDays = ttlDaysRaw.(int)
	} else if req.Operation == logical.CreateOperation {
		role.DefaultTTLDays = 30
	}

	// Validate TTL <= MaxTTL
	if role.TTL > role.MaxTTL {
		return logical.ErrorResponse("ttl cannot be greater than max_ttl"), logical.ErrInvalidRequest
	}

	// Store the role
	entry, err := logical.StorageEntryJSON(rolePrefix+strings.ToLower(roleName), role)
	if err != nil {
		return nil, err
	}

	if entry == nil {
		return nil, fmt.Errorf("failed to create storage entry for role %s", roleName)
	}

	if err = req.Storage.Put(ctx, entry); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathRoleDelete(ctx context.Context, req *logical.Request, data *framework.FieldData) (*logical.Response, error) {
	roleName := data.Get("name").(string)
	if roleName == "" {
		return logical.ErrorResponse("missing role name"), nil
	}

	b.l.Lock()
	defer b.l.Unlock()

	if err := req.Storage.Delete(ctx, rolePrefix+strings.ToLower(roleName)); err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *backend) pathRoleExistenceCheck(ctx context.Context, req *logical.Request, data *framework.FieldData) (bool, error) {
	b.l.RLock()
	defer b.l.RUnlock()

	role, err := b.role(ctx, req.Storage, data.Get("name").(string))
	if err != nil {
		return false, err
	}

	return role != nil, nil
}
