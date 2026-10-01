package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/internal/config"
)

// DoOpencodeLogin sets up credentials for the OpenCode Zen free tier.
// The Zen free tier is keyless (Authorization: Bearer public) - no OAuth or
// API key is needed. This command simply writes a marker credential file so
// the server knows to route requests to the Zen anonymous lane.
func DoOpencodeLogin(cfg *config.Config, options *LoginOptions) {
	if options == nil {
		options = &LoginOptions{}
	}

	authDir := ""
	if cfg != nil {
		authDir = strings.TrimSpace(cfg.AuthDir)
	}
	if authDir == "" {
		fmt.Println("OpenCode: no auth directory configured; skipping credential file write.")
		fmt.Println("OpenCode Zen free tier is ready - no API key required.")
		return
	}

	if err := os.MkdirAll(authDir, 0700); err != nil {
		fmt.Printf("OpenCode: failed to create auth directory: %v\n", err)
		return
	}

	record := map[string]any{
		"type":     "opencode",
		"provider": "opencode",
		"tier":     "zen-free",
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		fmt.Printf("OpenCode: failed to encode credential: %v\n", err)
		return
	}

	filePath := filepath.Join(authDir, "opencode-default.json")
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		fmt.Printf("OpenCode: failed to write credential file: %v\n", err)
		return
	}

	fmt.Printf("OpenCode Zen free tier enabled. Marker file saved to %s\n", filePath)
	fmt.Println()
	fmt.Println("How this works:")
	fmt.Println("  The OpenCode Zen endpoint (https://opencode.ai/zen/v1) is a public,")
	fmt.Println("  anonymous free tier that requires NO account, NO API key, and NO login.")
	fmt.Println("  Access is granted to requests that match the official OpenCode client shape:")
	fmt.Println("    - Authorization: Bearer public")
	fmt.Println("    - Canonical session ID (ses_<12 hex><14 base62>, SHA-256 derived)")
	fmt.Println("    - x-opencode-{client,project,request,session} headers")
	fmt.Println("    - stream: true + tools array containing: bash, edit, glob, grep, read")
	fmt.Println("  CLIProxyAPI injects all of these automatically on every request.")
	fmt.Println()
	fmt.Println("Available free chat models are discovered from OpenCode Zen's live catalog.")
}
