package auth

import (
	"context"
	"fmt"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v7/sdk/cliproxy/auth"
)

// OpencodeAuthenticator handles OpenCode authentication.
type OpencodeAuthenticator struct{}

// NewOpencodeAuthenticator creates a new OpenCode authenticator.
func NewOpencodeAuthenticator() *OpencodeAuthenticator {
	return &OpencodeAuthenticator{}
}

// Login performs the login for OpenCode.
func (a *OpencodeAuthenticator) Login(ctx context.Context, cfg *config.Config, opts *LoginOptions) (*auth.Record, error) {
	// For now, since OpenCode free tier is keyless or anonymous, we return a generic record.
	// This can be updated to include actual OAuth or device flow if needed.
	return &auth.Record{
		ID:       "opencode-default",
		Provider: "opencode",
	}, nil
}
