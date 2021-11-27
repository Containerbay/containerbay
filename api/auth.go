package api

import (
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/moby/moby/api/types"
)

type staticAuth struct {
	auth *types.AuthConfig
}

// Authorization returns an *authn.AuthConfig from a static configuration
func (s staticAuth) Authorization() (*authn.AuthConfig, error) {
	if s.auth == nil {
		return nil, nil
	}
	return &authn.AuthConfig{
		Username:      s.auth.Username,
		Password:      s.auth.Password,
		Auth:          s.auth.Auth,
		IdentityToken: s.auth.IdentityToken,
		RegistryToken: s.auth.RegistryToken,
	}, nil
}
