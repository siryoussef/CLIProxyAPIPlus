package auth

import (
	"context"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
)

// OpencodeAuthenticator handles OpenCode authentication.
type OpencodeAuthenticator struct{}

// NewOpencodeAuthenticator creates a new OpenCode authenticator.
func NewOpencodeAuthenticator() *OpencodeAuthenticator {
	return &OpencodeAuthenticator{}
}

// Provider returns the provider name.
func (a *OpencodeAuthenticator) Provider() string {
	return "opencode"
}

// Login performs the login for OpenCode.
func (a *OpencodeAuthenticator) Login(ctx context.Context, cfg *config.Config, opts *LoginOptions) (*auth.Auth, error) {
	// For now, since OpenCode free tier is keyless or anonymous, we return a generic auth.
	return &auth.Auth{
		ID:       "opencode-default",
		Provider: "opencode",
	}, nil
}

// RefreshLead returns nil as refresh is not supported for opencode.
func (a *OpencodeAuthenticator) RefreshLead() *time.Duration {
	return nil
}
