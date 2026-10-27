package main

import (
	"slices"
	"strings"
	"testing"
)

// modernMeta is the per-request _meta a 2026-07-28 client sends in place of
// the initialize handshake.
const modernMeta = `"_meta":{"io.modelcontextprotocol/protocolVersion":"2026-07-28","io.modelcontextprotocol/clientCapabilities":{},"io.modelcontextprotocol/clientInfo":{"name":"test","version":"0"}}`

func result(t *testing.T, resp map[string]any) map[string]any {
	t.Helper()
	r, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result in %v", resp)
	}
	return r
}

// A 2026-07-28 client never sends initialize: it asks server/discover, then
// sends each request with its version in _meta.
func TestProtocolModern(t *testing.T) {
	got := exchange(t, "",
		`{"jsonrpc":"2.0","id":1,"method":"server/discover","params":{`+modernMeta+`}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{`+modernMeta+`}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"status","arguments":{},`+modernMeta+`}}`,
		`{"jsonrpc":"2.0","id":4,"method":"ping","params":{`+modernMeta+`}}`,
	)

	discover := result(t, got[1])
	var versions []string
	for _, v := range discover["supportedVersions"].([]any) {
		versions = append(versions, v.(string))
	}
	for _, want := range []string{"2026-07-28", "2025-11-25", "2024-11-05"} {
		if !slices.Contains(versions, want) {
			t.Errorf("server/discover supportedVersions = %v, missing %s", versions, want)
		}
	}
	if !strings.Contains(discover["instructions"].(string), "kernel route") {
		t.Error("server/discover must carry the instructions")
	}
	meta, _ := discover["_meta"].(map[string]any)
	info, _ := meta["io.modelcontextprotocol/serverInfo"].(map[string]any)
	if info["name"] != "vpn-bypass-mcp" || info["title"] != "VPN Bypass" || info["websiteUrl"] != websiteURL {
		t.Errorf("serverInfo = %v", info)
	}

	for id, method := range map[float64]string{1: "server/discover", 2: "tools/list"} {
		r := result(t, got[id])
		if r["resultType"] != "complete" || r["ttlMs"] != float64(listTTL) || r["cacheScope"] != "private" {
			t.Errorf("%s: resultType=%v ttlMs=%v cacheScope=%v, want complete, %d, private", method, r["resultType"], r["ttlMs"], r["cacheScope"], listTTL)
		}
	}
	if n := len(toolNames(t, got[2])); n != 23 {
		t.Errorf("modern tools/list returned %d tools, want 23", n)
	}

	status := result(t, got[3])
	if status["resultType"] != "complete" || status["isError"] == true {
		t.Errorf("modern tools/call status = %v", status)
	}

	// ping was removed in 2026-07-28; a modern ping is an unknown method.
	if e, ok := got[4]["error"].(map[string]any); !ok || e["code"] != float64(-32601) {
		t.Errorf("modern ping = %v, want error -32601", got[4])
	}
}

// Claude Code, Claude Desktop, Cursor and VS Code still use the initialize
// handshake; the same binary must keep serving them.
func TestProtocolLegacy(t *testing.T) {
	got := exchange(t, "",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"ping"}`,
	)
	init := result(t, got[1])
	if init["protocolVersion"] != "2025-11-25" {
		t.Errorf("initialize protocolVersion = %v, want 2025-11-25", init["protocolVersion"])
	}
	if info := init["serverInfo"].(map[string]any); info["title"] != "VPN Bypass" {
		t.Errorf("serverInfo = %v", info)
	}
	list := result(t, got[2])
	if n := len(toolNames(t, got[2])); n != 23 {
		t.Errorf("legacy tools/list returned %d tools, want 23", n)
	}
	// Legacy results keep their old shape: no 2026-07-28 fields.
	for _, k := range []string{"resultType", "ttlMs", "cacheScope"} {
		if _, ok := list[k]; ok {
			t.Errorf("legacy tools/list carries %s", k)
		}
	}
	if _, ok := got[3]["result"]; !ok {
		t.Errorf("legacy ping = %v, want a result", got[3])
	}
}
