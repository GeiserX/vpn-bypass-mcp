package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Prompt is one MCP prompt: a starting message for a common job. Prompts only
// steer the model; every change still goes through a tool and its hints.
type Prompt struct {
	Def      mcp.Prompt
	ReadOnly bool
	text     func(args map[string]string) string
}

// Prompts returns every prompt, or only those that change nothing when
// readOnly is set.
func Prompts(readOnly bool) []Prompt {
	all := []Prompt{
		{
			ReadOnly: true,
			Def: mcp.NewPrompt("diagnose_site",
				mcp.WithPromptTitle("Why does a site use or skip the VPN?"),
				mcp.WithPromptDescription("Find out, with read tools only, why a site goes through the VPN or around it."),
				mcp.WithArgument("site", mcp.RequiredArgument(), mcp.ArgumentDescription("The host name, such as example.com.")),
			),
			text: func(a map[string]string) string {
				return fmt.Sprintf("Find out why %s goes through the VPN or around it, using VPN Bypass read tools only; change nothing.\n"+
					"1. status: the mode, whether a VPN is connected and whether the app is enforcing.\n"+
					"2. In bypass or vpnOnly mode: list_domains for an entry matching %[1]s, and list_services plus get_service for a service that covers it.\n"+
					"3. In custom mode: custom_list_rules (first match wins) and custom_list_routes, then the default route from status.\n"+
					"4. list_active_routes with source set to the matching entry or service, and get_logs with level warning, then error.\n"+
					"Then tell me in a few sentences which entry, service or rule decides it, whether its kernel routes are installed, and what would change it.", a["site"])
			},
		},
		{
			Def: mcp.NewPrompt("bypass_site",
				mcp.WithPromptTitle("Send a site around the VPN"),
				mcp.WithPromptDescription("Make a site skip the VPN in whatever mode VPN Bypass is in, asking before each change."),
				mcp.WithArgument("site", mcp.RequiredArgument(), mcp.ArgumentDescription("The host name, such as example.com.")),
			),
			text: func(a map[string]string) string {
				return fmt.Sprintf("Make %s skip the VPN with VPN Bypass. Read status first and work in the current mode; do not switch modes.\n"+
					"- bypass mode: add_domain with list bypass.\n"+
					"- vpnOnly mode: the site already skips the VPN unless it is on the VPN Only list; if list_domains shows it there, offer remove_domain.\n"+
					"- custom mode: find the Direct route in custom_list_routes and offer custom_add_rule with match suffix, pattern %[1]s and that route_id.\n"+
					"Tell me the exact change and wait for my yes before calling any tool that changes something. Afterwards, show what changed with list_active_routes or get_logs.", a["site"])
			},
		},
		{
			Def: mcp.NewPrompt("rotate_proxy_port",
				mcp.WithPromptTitle("Move a proxy route to another port"),
				mcp.WithPromptDescription("Point a Custom-mode proxy route at another port, for example to get a new exit IP from a residential proxy."),
				mcp.WithArgument("route", mcp.RequiredArgument(), mcp.ArgumentDescription("The route's name or id, as custom_list_routes shows it.")),
				mcp.WithArgument("port", mcp.RequiredArgument(), mcp.ArgumentDescription("The new port, 1 to 65535.")),
			),
			text: func(a map[string]string) string {
				return fmt.Sprintf("Point the VPN Bypass Custom-mode route %q at port %s.\n"+
					"1. custom_list_routes: find the route by name or id. If none or more than one matches, stop and ask me.\n"+
					"2. Tell me its current host and port and the change, and wait for my yes.\n"+
					"3. custom_update_route with its id and port %[2]s; change nothing else.\n"+
					"4. custom_list_routes again and show me the new proxyPort and listenerPort.", a["route"], a["port"])
			},
		},
	}
	if !readOnly {
		return all
	}
	var out []Prompt
	for _, p := range all {
		if p.ReadOnly {
			out = append(out, p)
		}
	}
	return out
}

// RegisterPrompts adds the prompts to s and returns how many it added.
func RegisterPrompts(s *server.MCPServer, readOnly bool) int {
	prompts := Prompts(readOnly)
	for _, p := range prompts {
		s.AddPrompt(p.Def, promptHandler(p))
	}
	return len(prompts)
}

func promptHandler(p Prompt) server.PromptHandlerFunc {
	return func(_ context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		args := map[string]string{}
		for _, a := range p.Def.Arguments {
			v := strings.TrimSpace(req.Params.Arguments[a.Name])
			if v == "" && a.Required {
				return nil, fmt.Errorf("%s needs the %s argument", p.Def.Name, a.Name)
			}
			args[a.Name] = v
		}
		return mcp.NewGetPromptResult(p.Def.Description, []mcp.PromptMessage{
			mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(p.text(args))),
		}), nil
	}
}
