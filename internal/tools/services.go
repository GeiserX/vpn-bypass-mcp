package tools

import "github.com/mark3labs/mcp-go/mcp"

const serviceLine = "A service is a bundle of domains and IP ranges for one app or site, built in or added by the user. " +
	"In Bypass mode an enabled service goes around the VPN; VPN Only mode ignores services; Custom mode reaches them through rules with match=service. "

func serviceTools() []Tool {
	return []Tool{
		{
			ReadOnly: true,
			Def: mcp.NewTool("list_services",
				mcp.WithDescription(appLine+serviceLine+
					"List every service with id, name, enabled, isCustom, domainCount and ipRangeCount. Use get_service for the domains and IP ranges of one."+needs49),
				read(),
			),
			build: simple("service.list"),
		},
		{
			ReadOnly: true,
			Def: mcp.NewTool("get_service",
				mcp.WithDescription(appLine+serviceLine+
					"Show one service with its full domains and ipRanges lists. Fails with not_found for an unknown id."+needs49),
				mcp.WithString("id", mcp.Required(), mcp.Description("The service id from list_services.")),
				read(),
			),
			build: func(a args) (call, error) {
				if _, err := a.required("id"); err != nil {
					return call{}, err
				}
				return simple("service.list", "id", "id")(a)
			},
		},
		{
			Def: mcp.NewTool("set_service_enabled",
				mcp.WithDescription(appLine+serviceLine+
					"Turn one service on or off. The change is saved before the answer comes back; its kernel routes are added or removed in the background, and only while a VPN is connected. "+
					"Asking for the state it already has is not an error."+needs49),
				mcp.WithString("id", mcp.Required(), mcp.Description("The service id from list_services.")),
				mcp.WithBoolean("enabled", mcp.Required(), mcp.Description("true to enable, false to disable.")),
				write(false, true),
			),
			build: func(a args) (call, error) {
				on, err := a.requiredBool("enabled")
				if err != nil {
					return call{}, err
				}
				if _, err := a.required("id"); err != nil {
					return call{}, err
				}
				cmd := "service.disable"
				if on {
					cmd = "service.enable"
				}
				return simple(cmd, "id", "id")(a)
			},
		},
	}
}
