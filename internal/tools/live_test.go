package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/geiserx/vpn-bypass-mcp/client"
	"github.com/geiserx/vpn-bypass-mcp/config"
	"github.com/mark3labs/mcp-go/mcp"
)

// TestLiveReadOnly runs against the real VPN Bypass when its control socket
// exists on this Mac, and skips everywhere else (CI included). It calls read
// tools only: a mutating verb would change the live routing of the machine.
// It never prints what the app returns, only whether each call worked.
func TestLiveReadOnly(t *testing.T) {
	path := config.Load().SocketPath
	if fi, err := os.Stat(path); err != nil || fi.Mode()&os.ModeSocket == 0 {
		t.Skipf("no VPN Bypass control socket at %s", path)
	}
	c := client.New(path)
	// A crashed or force-quit app leaves the socket file behind with nothing
	// listening: that is no app, not a failure of this code.
	if _, err := c.Call(context.Background(), "status", nil, nil); errors.Is(err, client.ErrNotRunning) {
		t.Skipf("no VPN Bypass answering on %s", path)
	}

	call := func(tool Tool, a map[string]any) (bool, string) {
		t.Helper()
		// Guard: this test must never reach a verb that changes state.
		built, err := tool.build(args(a))
		if err != nil {
			t.Fatalf("%s: %v", tool.Def.Name, err)
		}
		if !tool.ReadOnly || client.IsMutating(built.cmd) {
			t.Fatalf("%s would send %s, which is not a read verb", tool.Def.Name, built.cmd)
		}
		res, err := Handler(c, tool)(context.Background(), mcp.CallToolRequest{Params: mcp.CallToolParams{Name: tool.Def.Name, Arguments: a}})
		if err != nil {
			t.Fatalf("%s: %v", tool.Def.Name, err)
		}
		text, _ := mcp.AsTextContent(res.Content[0])
		return !res.IsError, text.Text
	}

	ok, text := call(find(t, "status"), nil)
	if !ok {
		t.Fatalf("status failed: %s", text)
	}
	var st struct {
		Mode    string `json:"mode"`
		Runtime struct {
			AppVersion string `json:"appVersion"`
		} `json:"runtime"`
	}
	if err := json.Unmarshal([]byte(text), &st); err != nil || st.Mode == "" {
		t.Fatalf("status result is not the expected shape (%v)", err)
	}
	has49 := st.Runtime.AppVersion != ""
	t.Logf("status ok; app reports version %q", st.Runtime.AppVersion)

	for _, tool := range All(true) {
		name := tool.Def.Name
		if name == "status" || name == "get_service" {
			continue
		}
		ok, text := call(tool, nil)
		switch name {
		case "custom_list_routes", "custom_list_rules":
			if !ok {
				t.Errorf("%s failed: %s", name, text)
			}
		default:
			if has49 && !ok {
				t.Errorf("%s failed on an app that reports its version: %s", name, text)
			}
			if !has49 && (ok || !strings.HasPrefix(text, "this needs VPN Bypass 4.9.0 or newer")) {
				t.Errorf("%s on an app before 4.9.0: ok=%v, want the 4.9.0 error, got %q", name, ok, text)
			}
		}
		t.Logf("%s: ok=%v, %d bytes", name, ok, len(text))
	}
}
