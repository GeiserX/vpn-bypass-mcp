package tools

import "github.com/mark3labs/mcp-go/mcp"

func kernelTools() []Tool {
	return []Tool{
		{
			ReadOnly: true,
			Def: mcp.NewTool("list_active_routes",
				mcp.WithDescription(appLine+"List the kernel routes it has installed in the macOS routing table right now. "+
					"Each has destination (an IP or CIDR), gateway, and source (the domain or service name that caused it)."+notEgress+needs49),
				mcp.WithString("source", mcp.Description("Only routes from this domain or service name (exact match, any case).")),
				read(),
			),
			build: simple("routes.active", "source", "source"),
		},
		{
			Def: mcp.NewTool("clear_routes",
				mcp.WithDescription(appLine+"Remove every kernel route it installed, like the Clear button in its menu. Ask the user first: the listed traffic stops being routed until the routes come back. "+
					"The configuration does not change, and the routes return on the next refresh_routes, VPN reconnect or DNS refresh. result.message says how many were removed."+notEgress+needs49),
				write(true, true),
			),
			build: simple("routes.clear"),
		},
		{
			Def: mcp.NewTool("refresh_routes",
				mcp.WithDescription(appLine+"Detect the VPN again and re-apply every kernel route, like Refresh Routes in its menu. "+
					"Answers \"refresh started\" at once and works in the background: read status or list_active_routes a few seconds later. "+
					"Fails with helper_not_ready when the privileged helper that installs routes is not ready."+needs49),
				write(false, true),
			),
			build: simple("refresh"),
		},
		{
			Def: mcp.NewTool("refresh_dns",
				mcp.WithDescription(appLine+"Resolve every listed domain again and change only the kernel routes whose addresses changed, like Refresh DNS now in its Settings. "+
					"Answers \"DNS refresh started\" at once and works in the background."+needs49),
				write(false, true),
			),
			build: simple("dns.refresh"),
		},
	}
}
