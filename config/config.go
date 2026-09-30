// Package config reads the server's settings from the environment.
package config

import (
	"os"
	"path/filepath"
	"strings"
)

// Config is everything the server needs to start.
type Config struct {
	// SocketPath is the VPN Bypass control socket. VPNB_SOCKET overrides it,
	// the same variable the vpnb command-line client reads.
	SocketPath string
	// ReadOnly registers only the tools that never change the app's state.
	// Set with VPN_BYPASS_MCP_READ_ONLY=1; every value except 0, false and no
	// counts as on.
	ReadOnly bool
}

// Load reads the environment.
func Load() Config {
	return Config{
		SocketPath: socketPath(),
		ReadOnly:   readOnly(os.Getenv("VPN_BYPASS_MCP_READ_ONLY")),
	}
}

// DefaultSocketPath is where VPN Bypass creates its control socket:
// ~/Library/Application Support/VPNBypass/control.sock.
func DefaultSocketPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "~"
	}
	return filepath.Join(home, "Library", "Application Support", "VPNBypass", "control.sock")
}

func socketPath() string {
	if v := os.Getenv("VPNB_SOCKET"); v != "" {
		return v
	}
	return DefaultSocketPath()
}

// readOnly fails closed: any value other than empty, 0, false or no turns
// read-only mode on, so a typo such as "on" never exposes the write tools.
func readOnly(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "0", "false", "no":
		return false
	}
	return true
}
