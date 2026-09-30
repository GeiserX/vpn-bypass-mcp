package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("VPNB_SOCKET", "")
	t.Setenv("VPN_BYPASS_MCP_READ_ONLY", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	cfg := Load()
	want := filepath.Join(home, "Library", "Application Support", "VPNBypass", "control.sock")
	if cfg.SocketPath != want {
		t.Errorf("SocketPath = %q, want %q", cfg.SocketPath, want)
	}
	if cfg.ReadOnly {
		t.Error("ReadOnly = true, want false by default")
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("VPNB_SOCKET", "/tmp/vpnb-test.sock")
	t.Setenv("VPN_BYPASS_MCP_READ_ONLY", "1")
	cfg := Load()
	if cfg.SocketPath != "/tmp/vpnb-test.sock" {
		t.Errorf("SocketPath = %q, want the VPNB_SOCKET value", cfg.SocketPath)
	}
	if !cfg.ReadOnly {
		t.Error("ReadOnly = false, want true for VPN_BYPASS_MCP_READ_ONLY=1")
	}
}

func TestReadOnlyValues(t *testing.T) {
	for v, want := range map[string]bool{"1": true, "true": true, "YES": true, "0": false, "false": false, "": false, "on": false} {
		t.Setenv("VPN_BYPASS_MCP_READ_ONLY", v)
		if got := Load().ReadOnly; got != want {
			t.Errorf("VPN_BYPASS_MCP_READ_ONLY=%q: ReadOnly = %v, want %v", v, got, want)
		}
	}
}
