package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/geiserx/vpn-bypass-mcp/client"
	"github.com/geiserx/vpn-bypass-mcp/internal/fakeapp"
	"github.com/mark3labs/mcp-go/mcp"
)

func find(t *testing.T, name string) Tool {
	t.Helper()
	for _, tool := range All(false) {
		if tool.Def.Name == name {
			return tool
		}
	}
	t.Fatalf("no tool named %s", name)
	return Tool{}
}

func invoke(t *testing.T, c *client.Client, name string, a map[string]any) (*mcp.CallToolResult, string) {
	t.Helper()
	req := mcp.CallToolRequest{Params: mcp.CallToolParams{Name: name, Arguments: a}}
	res, err := Handler(c, find(t, name))(context.Background(), req)
	if err != nil {
		t.Fatalf("%s: handler returned a Go error: %v", name, err)
	}
	if len(res.Content) != 1 {
		t.Fatalf("%s: %d content items, want 1", name, len(res.Content))
	}
	text, ok := mcp.AsTextContent(res.Content[0])
	if !ok {
		t.Fatalf("%s: content is not text", name)
	}
	return res, text.Text
}

// Every tool: the exact request line it sends and the result it passes back.
var cases = []struct {
	tool   string
	args   map[string]any
	line   string
	result string
}{
	{"status", nil, `{"v":1,"cmd":"status"}`,
		`{"mode":"bypass","runtime":{"appVersion":"4.9.0","enforcing":true,"helperReady":true}}`},
	{"get_logs", nil, `{"v":1,"cmd":"logs"}`,
		`{"logs":[{"level":"INFO","message":"Routes applied","time":"2026-09-30T12:00:00Z"}]}`},
	{"get_logs", map[string]any{"limit": float64(20), "level": "error"}, `{"v":1,"cmd":"logs","args":{"level":"error","limit":"20"}}`,
		`{"logs":[]}`},
	{"set_mode", map[string]any{"mode": "vpnOnly"}, `{"v":1,"cmd":"mode","args":{"mode":"vpnOnly"}}`,
		`{"mode":"vpnOnly"}`},
	{"list_domains", nil, `{"v":1,"cmd":"domain.list"}`,
		`{"domains":[{"domain":"example.com","enabled":true,"id":"6F1C","isCIDR":false,"isWildcard":false,"list":"bypass"}]}`},
	{"list_domains", map[string]any{"list": "vpnOnly"}, `{"v":1,"cmd":"domain.list","args":{"list":"vpnOnly"}}`,
		`{"domains":[]}`},
	{"add_domain", map[string]any{"domain": "example.com"}, `{"v":1,"cmd":"domain.add","args":{"domain":"example.com"}}`,
		`{"domains":[{"domain":"example.com","enabled":true,"id":"6F1C","isCIDR":false,"isWildcard":false,"list":"bypass"}]}`},
	{"add_domain", map[string]any{"domain": "https://www.example.com/page"}, `{"v":1,"cmd":"domain.add","args":{"domain":"https://www.example.com/page"}}`,
		`{"domains":[{"domain":"www.example.com","enabled":true,"id":"6F1D","isCIDR":false,"isWildcard":false,"list":"bypass"}]}`},
	{"add_domain", map[string]any{"domain": "10.0.0.0/8", "list": "vpnOnly"}, `{"v":1,"cmd":"domain.add","args":{"domain":"10.0.0.0/8","list":"vpnOnly"}}`,
		`{"domains":[{"domain":"10.0.0.0/8","enabled":true,"id":"A1","isCIDR":true,"isWildcard":false,"list":"vpnOnly"}]}`},
	{"remove_domain", map[string]any{"id": "6F1C"}, `{"v":1,"cmd":"domain.rm","args":{"id":"6F1C"}}`,
		`{"message":"domain removed"}`},
	{"remove_domain", map[string]any{"domain": "example.com", "list": "bypass"}, `{"v":1,"cmd":"domain.rm","args":{"domain":"example.com","list":"bypass"}}`,
		`{"message":"domain removed"}`},
	{"set_domain_enabled", map[string]any{"domain": "example.com", "enabled": true}, `{"v":1,"cmd":"domain.enable","args":{"domain":"example.com"}}`,
		`{"domains":[{"domain":"example.com","enabled":true}]}`},
	{"set_domain_enabled", map[string]any{"id": "6F1C", "enabled": false, "list": "bypass"}, `{"v":1,"cmd":"domain.disable","args":{"id":"6F1C","list":"bypass"}}`,
		`{"domains":[{"domain":"example.com","enabled":false}]}`},
	{"list_services", nil, `{"v":1,"cmd":"service.list"}`,
		`{"services":[{"domainCount":12,"enabled":true,"id":"netflix","ipRangeCount":3,"isCustom":false,"name":"Netflix"}]}`},
	{"get_service", map[string]any{"id": "netflix"}, `{"v":1,"cmd":"service.list","args":{"id":"netflix"}}`,
		`{"services":[{"domains":["netflix.com"],"id":"netflix","ipRanges":["23.246.0.0/18"]}]}`},
	{"set_service_enabled", map[string]any{"id": "netflix", "enabled": true}, `{"v":1,"cmd":"service.enable","args":{"id":"netflix"}}`,
		`{"services":[{"enabled":true,"id":"netflix"}]}`},
	{"set_service_enabled", map[string]any{"id": "netflix", "enabled": false}, `{"v":1,"cmd":"service.disable","args":{"id":"netflix"}}`,
		`{"services":[{"enabled":false,"id":"netflix"}]}`},
	{"list_active_routes", nil, `{"v":1,"cmd":"routes.active"}`,
		`{"activeRoutes":[{"destination":"93.184.216.34","gateway":"192.168.1.1","source":"example.com"}]}`},
	{"list_active_routes", map[string]any{"source": "Netflix"}, `{"v":1,"cmd":"routes.active","args":{"source":"Netflix"}}`,
		`{"activeRoutes":[]}`},
	{"clear_routes", nil, `{"v":1,"cmd":"routes.clear"}`, `{"message":"removed 12 routes"}`},
	{"refresh_routes", nil, `{"v":1,"cmd":"refresh"}`, `{"message":"refresh started"}`},
	{"refresh_dns", nil, `{"v":1,"cmd":"dns.refresh"}`, `{"message":"DNS refresh started"}`},
	{"custom_list_routes", nil, `{"v":1,"cmd":"route.list"}`,
		`{"routes":[{"egress":"proxySOCKS5","enabled":true,"hasPassword":true,"hasProxyUser":true,"id":"R1","name":"Resi","proxyPort":1080}]}`},
	{"custom_add_route", map[string]any{"name": "Resi", "type": "socks5", "host": "proxy.example.net", "port": float64(1080), "user": "u", "password": "hunter2"},
		`{"v":1,"cmd":"route.add","args":{"host":"proxy.example.net","name":"Resi","port":"1080","type":"socks5","user":"u"},"secrets":{"pass":"hunter2"}}`,
		`{"listenerPort":18042,"routes":[{"hasPassword":true,"id":"R1","name":"Resi"}]}`},
	{"custom_add_route", map[string]any{"name": "Work VPN", "type": "vpn", "interface": "utun4", "product": "WireGuard"},
		`{"v":1,"cmd":"route.add","args":{"interface":"utun4","name":"Work VPN","product":"WireGuard","type":"vpn"}}`,
		`{"routes":[{"id":"R2","name":"Work VPN"}]}`},
	{"custom_update_route", map[string]any{"id": "R1", "port": float64(24001), "enabled": false, "password": "hunter3"},
		`{"v":1,"cmd":"route.set","args":{"enabled":"false","id":"R1","port":"24001"},"secrets":{"pass":"hunter3"}}`,
		`{"routes":[{"enabled":false,"hasPassword":true,"id":"R1","proxyPort":24001}]}`},
	{"custom_update_route", map[string]any{"id": "R1", "user": "", "password": "", "name": "", "host": ""},
		`{"v":1,"cmd":"route.set","args":{"id":"R1","user":""},"secrets":{"pass":""}}`,
		`{"routes":[{"hasPassword":false,"hasProxyUser":false,"id":"R1"}]}`},
	{"custom_update_route", map[string]any{"id": "R1", "enabled": "TRUE"},
		`{"v":1,"cmd":"route.set","args":{"enabled":"true","id":"R1"}}`,
		`{"routes":[{"enabled":true,"id":"R1"}]}`},
	{"custom_set_route_enabled", map[string]any{"id": "R1", "enabled": true}, `{"v":1,"cmd":"route.enable","args":{"id":"R1"}}`,
		`{"routes":[{"enabled":true,"id":"R1"}]}`},
	{"custom_set_route_enabled", map[string]any{"id": "R1", "enabled": "false"}, `{"v":1,"cmd":"route.disable","args":{"id":"R1"}}`,
		`{"routes":[{"enabled":false,"id":"R1"}]}`},
	{"custom_remove_route", map[string]any{"id": "R1"}, `{"v":1,"cmd":"route.rm","args":{"id":"R1"}}`, `{"message":"route removed"}`},
	{"custom_list_rules", nil, `{"v":1,"cmd":"rule.list"}`,
		`{"rules":[{"enabled":true,"id":"U1","matchType":"cidr","order":0,"pattern":"10.0.0.0/8","routeId":"R1"}]}`},
	{"custom_add_rule", map[string]any{"match": "cidr", "pattern": "10.0.0.0/8", "route_id": "R1"},
		`{"v":1,"cmd":"rule.add","args":{"match":"cidr","pattern":"10.0.0.0/8","routeId":"R1"}}`,
		`{"rules":[{"id":"U1","matchType":"cidr","pattern":"10.0.0.0/8","routeId":"R1"}]}`},
	{"custom_remove_rule", map[string]any{"id": "U1"}, `{"v":1,"cmd":"rule.rm","args":{"id":"U1"}}`, `{"message":"rule removed"}`},
	{"custom_set_default_route", map[string]any{"route_id": "R1"}, `{"v":1,"cmd":"default","args":{"routeId":"R1"}}`, `{"defaultRouteId":"R1"}`},
}

func TestEveryToolSendsTheContractRequest(t *testing.T) {
	covered := map[string]bool{}
	for _, tc := range cases {
		covered[tc.tool] = true
		t.Run(tc.tool, func(t *testing.T) {
			app := fakeapp.Start(t, func(fakeapp.Request) fakeapp.Reply { return fakeapp.OK(json.RawMessage(tc.result)) })
			res, text := invoke(t, client.New(app.Path), tc.tool, tc.args)
			if res.IsError {
				t.Fatalf("tool error: %s", text)
			}
			if lines := app.Lines(); len(lines) != 1 || lines[0] != tc.line {
				t.Errorf("request lines = %q\nwant            [%s]", lines, tc.line)
			}
			if text != tc.result {
				t.Errorf("result = %s\nwant     %s", text, tc.result)
			}
		})
	}
	for _, tool := range All(false) {
		if !covered[tool.Def.Name] {
			t.Errorf("tool %s has no request test", tool.Def.Name)
		}
	}
}

func TestEmptyResultIsAnEmptyObject(t *testing.T) {
	app := fakeapp.Start(t, func(fakeapp.Request) fakeapp.Reply { return fakeapp.Reply{Raw: `{"v":1,"ok":true}`} })
	_, text := invoke(t, client.New(app.Path), "refresh_dns", nil)
	if text != "{}" {
		t.Fatalf("result = %q, want {}", text)
	}
}

func TestNumbersAndBooleansAreSentAsStrings(t *testing.T) {
	for _, tc := range []struct {
		in   any
		want string
	}{
		{float64(8080), "8080"}, {float64(0), "0"}, {float64(-1), "-1"}, {8080, "8080"}, {int64(65535), "65535"},
		{true, "true"}, {false, "false"}, {"24001", "24001"},
	} {
		got, ok, err := args{"port": tc.in}.str("port")
		if err != nil || !ok || got != tc.want {
			t.Errorf("%#v -> %q, %v, %v; want %q", tc.in, got, ok, err, tc.want)
		}
	}
	for _, bad := range []any{12.5, []any{1}, map[string]any{}, float64(1 << 60)} {
		if _, _, err := (args{"port": bad}).str("port"); err == nil {
			t.Errorf("%#v: want an error", bad)
		}
	}
}

func TestInvalidArgumentsAreRefusedBeforeSending(t *testing.T) {
	for _, tc := range []struct {
		tool string
		args map[string]any
		want string
	}{
		{"remove_domain", map[string]any{}, "give id or domain"},
		{"remove_domain", map[string]any{"id": "A", "domain": "b.com"}, "not both"},
		{"set_domain_enabled", map[string]any{"id": "A"}, "enabled is required"},
		{"set_domain_enabled", map[string]any{"id": "A", "enabled": "maybe"}, "true or false"},
		{"add_domain", map[string]any{"domain": ""}, "domain is required"},
		{"get_service", map[string]any{}, "id is required"},
		{"set_service_enabled", map[string]any{"enabled": true}, "id is required"},
		{"set_mode", map[string]any{}, "mode is required"},
		{"custom_add_route", map[string]any{"name": "x", "port": 80.5, "password": "hunter2"}, "whole number"},
		{"custom_update_route", map[string]any{"port": float64(1)}, "id is required"},
		{"custom_update_route", map[string]any{"id": "R1", "enabled": "yes"}, "true or false"},
		{"custom_set_route_enabled", map[string]any{"enabled": true}, "id is required"},
		{"custom_remove_route", map[string]any{}, "id is required"},
		{"custom_add_rule", map[string]any{"match": "ip", "pattern": "1.2.3.4"}, "route_id is required"},
		{"custom_remove_rule", map[string]any{}, "id is required"},
		{"custom_set_default_route", map[string]any{}, "route_id is required"},
	} {
		app := fakeapp.Start(t, func(fakeapp.Request) fakeapp.Reply { return fakeapp.OK(nil) })
		res, text := invoke(t, client.New(app.Path), tc.tool, tc.args)
		if !res.IsError || !strings.Contains(text, tc.want) {
			t.Errorf("%s %v: got %q (error=%v), want an error containing %q", tc.tool, tc.args, text, res.IsError, tc.want)
		}
		if strings.Contains(text, "hunter2") {
			t.Errorf("%s: the password appears in the error", tc.tool)
		}
		if n := len(app.Lines()); n != 0 {
			t.Errorf("%s: sent %d requests for invalid arguments, want 0", tc.tool, n)
		}
	}
}

func TestNotRunning(t *testing.T) {
	dir, err := os.MkdirTemp("", "vpnb")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	res, text := invoke(t, client.New(filepath.Join(dir, "none.sock")), "status", nil)
	if !res.IsError || !strings.HasPrefix(text, "VPN Bypass is not running; start the app") {
		t.Fatalf("got %q, want the not-running message", text)
	}
}

func TestUnknownCommandOnAnOldApp(t *testing.T) {
	app := fakeapp.Start(t, fakeapp.Legacy)
	res, text := invoke(t, client.New(app.Path), "list_domains", nil)
	if !res.IsError {
		t.Fatal("want an error from a 4.8 app")
	}
	want := "this needs VPN Bypass 4.9.0 or newer: the running app is older and does not know the domain.list command. Update VPN Bypass and try again."
	if text != want {
		t.Errorf("message = %q\nwant      %q", text, want)
	}
	// Only the verb itself: no second call to learn the version.
	if got := app.Lines(); len(got) != 1 {
		t.Errorf("request lines = %q, want one", got)
	}
}

func TestOtherAppErrorsPassThrough(t *testing.T) {
	for _, code := range []string{"not_found", "invalid_args", "already_exists", "helper_not_ready", "timeout", "request_too_large", "invalid_port"} {
		app := fakeapp.Start(t, func(fakeapp.Request) fakeapp.Reply { return fakeapp.Fail(code, "details from the app") })
		res, text := invoke(t, client.New(app.Path), "status", nil)
		if !res.IsError || text != code+": details from the app" {
			t.Errorf("%s: got %q", code, text)
		}
	}
}

func TestRequestOver64KiBIsRefused(t *testing.T) {
	app := fakeapp.Start(t, func(fakeapp.Request) fakeapp.Reply { return fakeapp.OK(nil) })
	res, text := invoke(t, client.New(app.Path), "add_domain", map[string]any{"domain": strings.Repeat("a", 70*1024) + ".com"})
	if !res.IsError || !strings.HasPrefix(text, "request_too_large: ") {
		t.Fatalf("got %q, want request_too_large", text)
	}
	if n := len(app.Lines()); n != 0 {
		t.Errorf("sent %d requests, want 0", n)
	}
}

var readTools = []string{"custom_list_routes", "custom_list_rules", "get_logs", "get_service", "list_active_routes", "list_domains", "list_services", "status"}

func TestReadOnlyModeKeepsOnlyReadTools(t *testing.T) {
	var names []string
	for _, tool := range All(true) {
		names = append(names, tool.Def.Name)
	}
	sort.Strings(names)
	if strings.Join(names, ",") != strings.Join(readTools, ",") {
		t.Fatalf("read-only tools = %v\nwant %v", names, readTools)
	}
	if len(All(false)) != 23 {
		t.Errorf("all tools = %d, want 23", len(All(false)))
	}
}

// The tools a client should confirm before calling. set_mode is one: the app
// re-applies every kernel route even when the mode does not change.
func TestDestructiveTools(t *testing.T) {
	want := "clear_routes,custom_remove_route,custom_remove_rule,custom_set_default_route,custom_update_route,remove_domain,set_mode"
	var got []string
	for _, tool := range All(false) {
		if *tool.Def.Annotations.DestructiveHint {
			got = append(got, tool.Def.Name)
		}
	}
	sort.Strings(got)
	if strings.Join(got, ",") != want {
		t.Errorf("destructive tools = %v\nwant %s", got, want)
	}
	if mode := find(t, "set_mode").Def.Annotations; *mode.IdempotentHint {
		t.Error("set_mode is not idempotent: every call re-applies the kernel routes")
	}
}

// The ReadOnly flag, the MCP annotation and the socket verb must agree, so
// read-only mode can never register a tool that sends a mutating verb.
func TestAnnotationsMatchTheVerb(t *testing.T) {
	sample := map[string]any{"id": "X", "domain": "", "enabled": true, "mode": "bypass", "name": "n", "match": "ip", "pattern": "1.1.1.1", "route_id": "R"}
	for _, tool := range All(false) {
		a := tool.Def.Annotations
		if a.ReadOnlyHint == nil || a.DestructiveHint == nil || a.IdempotentHint == nil || a.OpenWorldHint == nil {
			t.Errorf("%s: every hint must be set", tool.Def.Name)
			continue
		}
		c, err := tool.build(args(sample))
		if err != nil {
			withDomain := map[string]any{"domain": "example.com"}
			for k, v := range sample {
				if k != "id" && k != "domain" {
					withDomain[k] = v
				}
			}
			c, err = tool.build(args(withDomain))
		}
		if err != nil {
			t.Errorf("%s: sample arguments rejected: %v", tool.Def.Name, err)
			continue
		}
		if *a.ReadOnlyHint != tool.ReadOnly || client.IsMutating(c.cmd) == tool.ReadOnly {
			t.Errorf("%s: ReadOnly=%v, readOnlyHint=%v, verb %s mutating=%v disagree", tool.Def.Name, tool.ReadOnly, *a.ReadOnlyHint, c.cmd, client.IsMutating(c.cmd))
		}
		if tool.ReadOnly && *a.DestructiveHint {
			t.Errorf("%s: a read tool cannot be destructive", tool.Def.Name)
		}
		if *a.OpenWorldHint {
			t.Errorf("%s: openWorldHint must be false", tool.Def.Name)
		}
		if !strings.Contains(tool.Def.Description, "VPN Bypass is a macOS menu bar app") {
			t.Errorf("%s: the description must say what the app is", tool.Def.Name)
		}
	}
}

// Since VPN Bypass 5.0 domain.add takes a pasted link on the bypass list and saves its
// host; only an IP range is refused there. The description must not send an agent
// back to "host names only".
func TestAddDomainDescribesLinksAndRanges(t *testing.T) {
	desc := find(t, "add_domain").Def.Description
	for _, want := range []string{
		"On the bypass list a pasted link works",
		"saves the host",
		"The bypass list refuses an IP range",
		"match=cidr",
		"a link fails as a malformed CIDR",
		"links work there from 5.0",
	} {
		if !strings.Contains(desc, want) {
			t.Errorf("add_domain description lacks %q", want)
		}
	}
	for _, stale := range []string{"host names only", "not a URL"} {
		if strings.Contains(desc, stale) {
			t.Errorf("add_domain description still says %q", stale)
		}
	}
}

// add_domain saves only the host of a link, and the app refuses a "/" without a scheme
// as a malformed CIDR on remove and enable, so the target arg must ask for the saved host.
func TestTargetDomainAsksForTheSavedHost(t *testing.T) {
	for _, name := range []string{"remove_domain", "set_domain_enabled"} {
		prop, ok := find(t, name).Def.InputSchema.Properties["domain"].(map[string]any)
		if !ok {
			t.Fatalf("%s: no domain property", name)
		}
		if desc, _ := prop["description"].(string); !strings.Contains(desc, "the saved host") || !strings.Contains(desc, "not the link") {
			t.Errorf("%s: domain description %q does not ask for the saved host", name, desc)
		}
	}
}
