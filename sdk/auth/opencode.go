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

// Login performs the login for OpenCode Zen free tier.
// The Zen free tier is keyless (Authorization: Bearer public) — no OAuth
// or API key is required. This login simply writes a marker credential file
// so the executor can route requests to the Zen endpoint.
func (a *OpencodeAuthenticator) Login(ctx context.Context, cfg *config.Config, opts *LoginOptions) (*auth.Auth, error) {
	return &auth.Auth{
		ID:       "opencode-default",
		Provider: "opencode",
		Metadata: map[string]any{
			"type":     "opencode",
			"provider": "opencode",
			"tier":     "zen-free",
		},
	}, nil
}

// RefreshLead returns nil as refresh is not supported for opencode.
func (a *OpencodeAuthenticator) RefreshLead() *time.Duration {
	return nil
}
