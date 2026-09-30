package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/geiserx/vpn-bypass-mcp/internal/fakeapp"
)

// session drives the real stdio server: it writes JSON-RPC lines and collects
// the response to each id.
func session(t *testing.T, readOnly string, calls ...string) map[float64]map[string]any {
	t.Helper()
	app := fakeapp.Start(t, func(r fakeapp.Request) fakeapp.Reply {
		if r.Cmd == "status" {
			return fakeapp.OK(map[string]any{"mode": "bypass", "runtime": map[string]any{"appVersion": "4.9.0", "enforcing": true}})
		}
		return fakeapp.Fail("unknown_command", "unknown command: "+r.Cmd)
	})
	t.Setenv("VPNB_SOCKET", app.Path)
	t.Setenv("VPN_BYPASS_MCP_READ_ONLY", readOnly)

	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan int, 1)
	var stderr bytes.Buffer
	go func() { done <- run(ctx, nil, inR, outW, &stderr); outW.Close() }()

	lines := append([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
	}, calls...)
	go func() {
		for _, l := range lines {
			io.WriteString(inW, l+"\n")
		}
	}()

	want := 0
	for _, l := range lines {
		if strings.Contains(l, `"id":`) {
			want++
		}
	}
	got := map[float64]map[string]any{}
	sc := bufio.NewScanner(outR)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	timer := time.AfterFunc(10*time.Second, func() { cancel(); inW.Close() })
	defer timer.Stop()
	for len(got) < want && sc.Scan() {
		var msg map[string]any
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
			t.Fatalf("stdout carried a non-JSON line: %q", sc.Text())
		}
		if id, ok := msg["id"].(float64); ok {
			got[id] = msg
		}
	}
	cancel()
	inW.Close()
	go io.Copy(io.Discard, outR)
	<-done
	if len(got) < want {
		t.Fatalf("got %d responses, want %d; stderr: %s", len(got), want, stderr.String())
	}
	return got
}

func toolNames(t *testing.T, resp map[string]any) []string {
	t.Helper()
	var names []string
	for _, tool := range resp["result"].(map[string]any)["tools"].([]any) {
		names = append(names, tool.(map[string]any)["name"].(string))
	}
	return names
}

func TestStdioInitializeListAndCall(t *testing.T) {
	got := session(t, "",
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"status","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_domains","arguments":{}}}`,
	)
	init := got[1]["result"].(map[string]any)
	if name := init["serverInfo"].(map[string]any)["name"]; name != "vpn-bypass-mcp" {
		t.Errorf("serverInfo.name = %v", name)
	}
	if !strings.Contains(init["instructions"].(string), "kernel route") {
		t.Error("initialize must carry the instructions")
	}
	if n := len(toolNames(t, got[2])); n != 23 {
		t.Errorf("tools/list returned %d tools, want 23", n)
	}
	status := got[3]["result"].(map[string]any)
	text := status["content"].([]any)[0].(map[string]any)["text"].(string)
	if text != `{"mode":"bypass","runtime":{"appVersion":"4.9.0","enforcing":true}}` || status["isError"] == true {
		t.Errorf("status result = %s (isError=%v)", text, status["isError"])
	}
	domains := got[4]["result"].(map[string]any)
	if domains["isError"] != true {
		t.Error("list_domains against an app without the verb must be an error")
	}
	if text := domains["content"].([]any)[0].(map[string]any)["text"].(string); !strings.Contains(text, "version 4.9.0") {
		t.Errorf("list_domains error = %q, want the running version named", text)
	}
}

func TestStdioReadOnly(t *testing.T) {
	got := session(t, "1", `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	names := toolNames(t, got[2])
	if len(names) != 8 {
		t.Fatalf("read-only tools/list returned %d tools (%v), want 8", len(names), names)
	}
	for _, n := range names {
		if n == "clear_routes" || n == "set_mode" || strings.HasPrefix(n, "add_") || strings.HasPrefix(n, "remove_") {
			t.Errorf("read-only mode registered %s", n)
		}
	}
}

func TestFlags(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run(context.Background(), []string{"--version"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), "dev") {
		t.Errorf("--version: code %d, out %q", code, out.String())
	}
	out.Reset()
	if code := run(context.Background(), []string{"--help"}, nil, &out, &errOut); code != 0 || !strings.Contains(out.String(), "VPNB_SOCKET") {
		t.Errorf("--help: code %d, out %q", code, out.String())
	}
	if code := run(context.Background(), []string{"--bogus"}, nil, &out, &errOut); code != 2 {
		t.Errorf("--bogus: code %d, want 2", code)
	}
}
