package tools

import (
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
)

const customLine = "In its Custom mode, rules send matching traffic to named egress routes and unmatched traffic to a default route. "

const egressTypes = "Egress types: vpnDefault = the VPN (optionally one pinned tunnel), direct = around the VPN through the local gateway, " +
	"proxyHTTP and proxySOCKS5 = a local 127.0.0.1 listener that forwards to a proxy, tailscaleExit = a Tailscale exit node."

const passwordArg = "The proxy password. Sent to the app in a separate secrets field; never returned, echoed or logged."

func customTools() []Tool {
	idArg := mcp.WithString("id", mcp.Required(), mcp.Description("The route id (a UUID from custom_list_routes)."))
	requireID := func(cmd string, pairs ...string) func(a args) (call, error) {
		return func(a args) (call, error) {
			if _, err := a.required("id"); err != nil {
				return call{}, err
			}
			return simple(cmd, append([]string{"id", "id"}, pairs...)...)(a)
		}
	}

	return []Tool{
		{
			ReadOnly: true,
			Def: mcp.NewTool("custom_list_routes",
				mcp.WithDescription(appLine+customLine+"List the egress routes: id, name, egress, enabled, proxyHost, proxyPort, hasProxyUser, hasPassword, tailscaleExitNode, vpnSelector, and listenerPort (the live local port of a proxy route). "+
					"Credentials are never returned; hasProxyUser and hasPassword say whether one is set. "+egressTypes+notKernel),
				read(),
			),
			build: simple("route.list"),
		},
		{
			Def: mcp.NewTool("custom_add_route",
				mcp.WithDescription(appLine+customLine+"Add an egress route. "+
					"type http or socks5: a proxy at host:port, with optional user and password. "+
					"type vpn: a VPN route; interface (such as utun4) pins one tunnel and product is its label (such as WireGuard), which still matches after the utun number changes. "+
					"type tailscale: a Tailscale route; this tool has no argument for the exit node, so pick it in the app. "+
					"Direct and detected VPN routes exist already. Every call adds a new route, even with the same name. The new route carries no traffic until a rule or the default points at it."+notKernel),
				mcp.WithString("name", mcp.Required(), mcp.Description("A name for the route.")),
				mcp.WithString("type", mcp.Enum("http", "socks5", "tailscale", "vpn"), mcp.Description("The egress type. Default http.")),
				mcp.WithString("host", mcp.Description("Proxy host (http and socks5).")),
				mcp.WithNumber("port", mcp.Description("Proxy port, 1 to 65535 (http and socks5)."), mcp.Min(1), mcp.Max(65535)),
				mcp.WithString("user", mcp.Description("Proxy user name.")),
				mcp.WithString("password", mcp.Description(passwordArg)),
				mcp.WithString("interface", mcp.Description("type vpn only: the tunnel interface to pin, such as utun4.")),
				mcp.WithString("product", mcp.Description("type vpn only: the tunnel's label, such as WireGuard.")),
				write(false, false),
			),
			build: func(a args) (call, error) {
				if _, err := a.required("name"); err != nil {
					return call{}, err
				}
				w, err := a.wire("name", "name", "type", "type", "host", "host", "port", "port", "user", "user", "interface", "interface", "product", "product")
				if err != nil {
					return call{}, err
				}
				s, err := a.secrets()
				return call{cmd: "route.add", args: w, secrets: s}, err
			},
		},
		{
			Def: mcp.NewTool("custom_update_route",
				mcp.WithDescription(appLine+customLine+"Change fields of an egress route; omitted fields keep their value. "+
					"An empty user or password clears it; an empty name, host or port is ignored. "+
					"A common use: point a residential proxy route at another port to change its exit IP. The app re-points the route's live listener."+notKernel),
				idArg,
				mcp.WithString("name", mcp.Description("New name.")),
				mcp.WithString("host", mcp.Description("New proxy host.")),
				mcp.WithNumber("port", mcp.Description("New proxy port, 1 to 65535."), mcp.Min(1), mcp.Max(65535)),
				mcp.WithString("user", mcp.Description("New proxy user name; an empty string removes it.")),
				mcp.WithBoolean("enabled", mcp.Description("true to enable, false to disable.")),
				mcp.WithString("password", mcp.Description(passwordArg+" An empty string removes it.")),
				write(true, true),
			),
			build: func(a args) (call, error) {
				c, err := requireID("route.set", "name", "name", "host", "host", "port", "port", "user", "user", "enabled", "enabled")(a)
				if err != nil {
					return call{}, err
				}
				if _, ok := c.args["enabled"]; ok {
					on, err := a.requiredBool("enabled")
					if err != nil {
						return call{}, err
					}
					c.args["enabled"] = strconv.FormatBool(on)
				}
				// An explicit empty user or password clears it: the app stores
				// "" and then reports hasProxyUser or hasPassword as false.
				if v, ok := a["user"].(string); ok && v == "" {
					c.args["user"] = ""
				}
				c.secrets, err = a.secrets()
				if v, ok := a["password"].(string); ok && v == "" && err == nil {
					c.secrets = map[string]string{"pass": ""}
				}
				return c, err
			},
		},
		{
			Def: mcp.NewTool("custom_set_route_enabled",
				mcp.WithDescription(appLine+customLine+"Turn one egress route on or off."+notKernel),
				idArg,
				mcp.WithBoolean("enabled", mcp.Required(), mcp.Description("true to enable, false to disable.")),
				write(false, true),
			),
			build: func(a args) (call, error) {
				on, err := a.requiredBool("enabled")
				if err != nil {
					return call{}, err
				}
				cmd := "route.disable"
				if on {
					cmd = "route.enable"
				}
				return requireID(cmd)(a)
			},
		},
		{
			Def: mcp.NewTool("custom_remove_route",
				mcp.WithDescription(appLine+customLine+"Remove an egress route, every rule that points at it, and the default setting if it was the default. Ask the user first."+notKernel),
				idArg,
				write(true, true),
			),
			build: requireID("route.rm"),
		},
		{
			ReadOnly: true,
			Def: mcp.NewTool("custom_list_rules",
				mcp.WithDescription(appLine+customLine+"List the rules in the order they are checked (first match wins): id, matchType, pattern, routeId, enabled, order. "+
					"Traffic no rule matches uses the default route (status result.defaultRouteId)."),
				read(),
			),
			build: simple("rule.list"),
		},
		{
			Def: mcp.NewTool("custom_add_rule",
				mcp.WithDescription(appLine+customLine+"Add a rule at the end of the list, so it is checked after the existing ones. "+
					"match: domain = one exact host name; suffix = a domain and its subdomains; ip = one IPv4 address; cidr = an IPv4 range (never /0 or /1); "+
					"service = a service id from list_services; process = a process name. suffix and process rules install no kernel routes. "+
					"Every call adds a new rule."),
				mcp.WithString("match", mcp.Required(), mcp.Enum("domain", "suffix", "ip", "cidr", "service", "process"), mcp.Description("What the pattern is.")),
				mcp.WithString("pattern", mcp.Required(), mcp.Description("The value to match.")),
				mcp.WithString("route_id", mcp.Required(), mcp.Description("The egress route id to send matching traffic to (from custom_list_routes).")),
				write(false, false),
			),
			build: func(a args) (call, error) {
				for _, k := range []string{"match", "pattern", "route_id"} {
					if _, err := a.required(k); err != nil {
						return call{}, err
					}
				}
				return simple("rule.add", "match", "match", "pattern", "pattern", "route_id", "routeId")(a)
			},
		},
		{
			Def: mcp.NewTool("custom_remove_rule",
				mcp.WithDescription(appLine+customLine+"Remove one rule."),
				mcp.WithString("id", mcp.Required(), mcp.Description("The rule id (a UUID from custom_list_rules).")),
				write(true, true),
			),
			build: requireID("rule.rm"),
		},
		{
			Def: mcp.NewTool("custom_set_default_route",
				mcp.WithDescription(appLine+customLine+"Set the egress route that traffic no rule matches uses."),
				mcp.WithString("route_id", mcp.Required(), mcp.Description("The egress route id (from custom_list_routes).")),
				write(true, true),
			),
			build: func(a args) (call, error) {
				if _, err := a.required("route_id"); err != nil {
					return call{}, err
				}
				return simple("default", "route_id", "routeId")(a)
			},
		},
	}
}
