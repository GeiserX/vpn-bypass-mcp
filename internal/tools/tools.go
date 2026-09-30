// Package tools defines the MCP tools. Each tool turns its MCP arguments into
// one control-socket request and returns the app's result object unchanged.
package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/geiserx/vpn-bypass-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Instructions is sent to the client at initialize.
const Instructions = `VPN Bypass is a macOS menu bar app that decides which traffic uses the VPN and which goes around it. This server reads and changes it through the app's local control socket.

Modes (status shows the current one):
- bypass: everything uses the VPN except the domains on the bypass list and the enabled services, which go around it.
- vpnOnly: everything goes around the VPN except the domains and CIDR ranges on the VPN Only list, which go through it.
- custom: rules match traffic by domain, IP, CIDR or service and send it to a named egress route (the VPN, Direct, an HTTP or SOCKS5 proxy, a Tailscale exit node); unmatched traffic uses the default route. The custom_* tools manage these.

"Route" means two different things here. A kernel route is an entry VPN Bypass has installed in the macOS routing table right now (list_active_routes, clear_routes, refresh_routes). A Custom-mode route is a named egress that rules point at (custom_list_routes and the other custom_*_route tools).

The domain, service, active-route, refresh and log tools need VPN Bypass 4.9.0 or newer. Ask the user before set_mode, clear_routes or any remove: they change how all or part of the Mac's traffic is routed.`

const (
	appLine    = "VPN Bypass is a macOS menu bar app that decides which traffic uses the VPN and which goes around it. "
	needs49    = " Needs VPN Bypass 4.9.0 or newer."
	notKernel  = " These are Custom-mode egress routes, not the kernel routes of list_active_routes."
	notEgress  = " These are kernel routes, not the Custom-mode egress routes of custom_list_routes."
	listArg    = "Which list: bypass (domains that go around the VPN in Bypass mode) or vpnOnly (domains and CIDR ranges that go through the VPN in VPN Only mode)."
	lookupNote = " Without list, an id is looked up on both lists; a domain that is on both lists needs list to choose."
)

// Tool is one MCP tool and the request it sends.
type Tool struct {
	Def      mcp.Tool
	ReadOnly bool
	build    func(a args) (call, error)
}

type call struct {
	cmd     string
	args    map[string]string
	secrets map[string]string
}

// simple builds a request that maps MCP arguments straight to socket args.
func simple(cmd string, pairs ...string) func(a args) (call, error) {
	return func(a args) (call, error) {
		w, err := a.wire(pairs...)
		return call{cmd: cmd, args: w}, err
	}
}

// All returns every tool, or only the read tools when readOnly is set.
func All(readOnly bool) []Tool {
	all := append(append(append(statusTools(), domainTools()...), serviceTools()...), append(kernelTools(), customTools()...)...)
	if !readOnly {
		return all
	}
	var out []Tool
	for _, t := range all {
		if t.ReadOnly {
			out = append(out, t)
		}
	}
	return out
}

// Register adds the tools to s and returns how many it added.
func Register(s *server.MCPServer, c *client.Client, readOnly bool) int {
	tools := All(readOnly)
	for _, t := range tools {
		s.AddTool(t.Def, Handler(c, t))
	}
	return len(tools)
}

// Handler runs one tool: build the request, send it, return the result.
func Handler(c *client.Client, t Tool) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		call, err := t.build(args(req.GetArguments()))
		if err != nil {
			return mcp.NewToolResultError("invalid arguments: " + err.Error()), nil
		}
		result, err := c.Call(ctx, call.cmd, call.args, call.secrets)
		if err != nil {
			return mcp.NewToolResultError(explain(call.cmd, err)), nil
		}
		if len(result) == 0 || string(result) == "null" {
			return mcp.NewToolResultText("{}"), nil
		}
		return mcp.NewToolResultText(string(result)), nil
	}
}

// explain turns a call error into a message an agent can act on. Only an app
// older than 4.9.0 answers unknown_command to a verb this server sends, so that
// code always means "update the app".
func explain(cmd string, err error) string {
	var ae *client.AppError
	if !errors.As(err, &ae) {
		return err.Error()
	}
	if ae.Code != "unknown_command" {
		return ae.Code + ": " + ae.Message
	}
	return fmt.Sprintf("this needs VPN Bypass 4.9.0 or newer: the running app is older and does not know the %s command. Update VPN Bypass and try again.", cmd)
}
