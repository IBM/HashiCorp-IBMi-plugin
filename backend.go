package ibmiauth

import (
	"context"
	"sync"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/IBM/HashiCorp-IBMi-plugin/internal/version"
)

func Factory(ctx context.Context, c *logical.BackendConfig) (logical.Backend, error) {
	b := Backend(c)
	if err := b.Setup(ctx, c); err != nil {
		return nil, err
	}
	return b, nil
}

type backend struct {
	*framework.Backend
	l sync.RWMutex
}

func Backend(c *logical.BackendConfig) *backend {
	var b backend
	
	b.Backend = &framework.Backend{
		BackendType: logical.TypeLogical, // This is a secrets backend
		PathsSpecial: &logical.Paths{
			// Unauthenticated paths - none for secrets backend
			Unauthenticated: []string{},
			// Seal wrap storage for sensitive configuration
			SealWrapStorage: []string{configPath},
		},
		Paths: []*framework.Path{
			b.PathConfig(),
			b.PathsRoleList(),
			b.PathRoleCRUD(),
			b.PathCredentials(),
		},
		Secrets: []*framework.Secret{
			b.ibmiCredential(),
		},
		RunningVersion: "v" + version.Version,
	}

	return &b
}
