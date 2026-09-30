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
// The Zen free tier is keyless (Authorization: Bearer public) — no OAuth or
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
		fmt.Println("OpenCode Zen free tier is ready — no API key required.")
		return
	}

	if err := os.MkdirAll(authDir, 0700); err != nil {
		fmt.Printf("OpenCode: failed to create auth directory: %v
", err)
		return
	}

	record := map[string]any{
		"type":     "opencode",
		"provider": "opencode",
		"tier":     "zen-free",
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		fmt.Printf("OpenCode: failed to encode credential: %v
", err)
		return
	}

	filePath := filepath.Join(authDir, "opencode-default.json")
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		fmt.Printf("OpenCode: failed to write credential file: %v
", err)
		return
	}

	fmt.Printf("OpenCode Zen free tier configured. Credential saved to %s
", filePath)
	fmt.Println("No API key required. Use models like: big-pickle, deepseek-v4-flash-free, etc.")
}
