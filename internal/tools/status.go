package tools

import (
	"context"
	"encoding/json"

	"github.com/geiserx/vpn-bypass-mcp/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const modes = "Modes: bypass = everything uses the VPN except the domains on the bypass list and the enabled services, which go around it; " +
	"vpnOnly = everything goes around the VPN except the domains and CIDR ranges on the VPN Only list; " +
	"custom = rules send matching traffic to named egress routes (VPN, Direct, HTTP or SOCKS5 proxy, Tailscale exit node) and the rest to the default route."

func statusTools() []Tool {
	return []Tool{
		{
			ReadOnly: true,
			Def: mcp.NewTool("status",
				mcp.WithToolTitle("VPN Bypass status"),
				mcp.WithDescription(appLine+"Show its current state. Call this first. "+
					"result.mode is the routing mode. "+modes+" "+
					"result.routes lists the Custom-mode egress routes (credentials never included) and result.defaultRouteId the Custom-mode default. "+
					"result.runtime holds the live facts: helperReady and helperState (the privileged helper that installs routes), vpnConnected, vpnInterface, vpnType, "+
					"enforcedRouteCount (kernel routes installed now), enforcing (VPN up, helper ready and routes installed), and appVersion from VPN Bypass 4.9.0 on."),
				read(),
				output("mode", "routes", "defaultRouteId", "schemaVersion", "supportedVersion", "runtime"),
			),
			build: simple("status"),
		},
		{
			ReadOnly: true,
			Def: mcp.NewTool("get_logs",
				mcp.WithToolTitle("Read the VPN Bypass log"),
				mcp.WithDescription(appLine+"Read its recent log, newest first: what it resolved, routed, skipped or failed. "+
					"Each entry has time (ISO 8601, UTC), level (INFO, SUCCESS, WARNING, ERROR) and message. The app keeps the last 200 entries. "+
					"Read this after a change to see what the app did in the background."+needs49),
				mcp.WithInteger("limit", mcp.Description("How many entries, newest first: 1 to 200. Default 50."), mcp.Min(1), mcp.Max(200), mcp.DefaultNumber(50)),
				mcp.WithString("level", mcp.Description("Only entries of this level (lowercase here; the entries say it in capitals)."), mcp.Enum("info", "success", "warning", "error")),
				read(),
				output("logs"),
			),
			build: simple("logs", "limit", "limit", "level", "level"),
		},
		{
			Def: mcp.NewTool("set_mode",
				mcp.WithToolTitle("Switch routing mode"),
				mcp.WithDescription(appLine+"Switch its routing mode. This changes how all of the Mac's traffic is routed: ask the user first. "+modes+" "+
					"Each mode keeps its own lists, so switching back restores them. When Custom mode has no rules yet, switching to it turns the current lists and enabled services into rules first. "+
					"Switching re-detects the VPN and re-applies every kernel route. When the mode is already the one asked for, nothing is sent or re-applied and the answer is the same result.mode."),
				mcp.WithString("mode", mcp.Required(), mcp.Enum("bypass", "vpnOnly", "custom"), mcp.Description("The mode to switch to.")),
				write(true, true),
			),
			build: func(a args) (call, error) {
				if _, err := a.required("mode"); err != nil {
					return call{}, err
				}
				return simple("mode", "mode", "mode")(a)
			},
			skip: sameMode,
		},
	}
}

// sameMode answers set_mode without sending it when the app is already in the
// asked-for mode, with the result the app itself gives from 4.9.0 on. Older
// apps saved the config and re-applied every kernel route even then. When
// status cannot be read, the mode request goes out and reports its own error.
func sameMode(ctx context.Context, c *client.Client, call call) json.RawMessage {
	raw, err := c.Call(ctx, "status", nil, nil)
	if err != nil {
		return nil
	}
	var st struct {
		Mode string `json:"mode"`
	}
	want := call.args["mode"]
	if json.Unmarshal(raw, &st) != nil || st.Mode == "" || st.Mode != want {
		return nil
	}
	out, err := json.Marshal(map[string]string{"mode": want})
	if err != nil {
		return nil
	}
	return out
}
