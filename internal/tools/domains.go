package tools

import "github.com/mark3labs/mcp-go/mcp"

func domainTools() []Tool {
	targetArgs := func(desc string, extra ...mcp.ToolOption) []mcp.ToolOption {
		return append([]mcp.ToolOption{
			mcp.WithDescription(desc),
			mcp.WithString("id", mcp.Description("The entry id (a UUID from list_domains). Give id or domain.")),
			mcp.WithString("domain", mcp.Description("The domain or CIDR as listed. Give id or domain.")),
			mcp.WithString("list", mcp.Description(listArg), mcp.Enum("bypass", "vpnOnly")),
		}, extra...)
	}
	targeted := func(cmd string) func(a args) (call, error) {
		return func(a args) (call, error) {
			if err := a.oneOf("id", "domain"); err != nil {
				return call{}, err
			}
			return simple(cmd, "id", "id", "domain", "domain", "list", "list")(a)
		}
	}

	return []Tool{
		{
			ReadOnly: true,
			Def: mcp.NewTool("list_domains",
				mcp.WithDescription(appLine+"List the entries of its two domain lists. "+
					"The bypass list is used in Bypass mode: those domains go around the VPN. "+
					"The vpnOnly list is used in VPN Only mode: only those domains and CIDR ranges go through the VPN. "+
					"Each entry has id, domain, enabled, list, isCIDR and isWildcard. Without list, both lists come back, bypass first."+needs49),
				mcp.WithString("list", mcp.Description(listArg+" Omit for both."), mcp.Enum("bypass", "vpnOnly")),
				read(),
			),
			build: simple("domain.list", "list", "list"),
		},
		{
			Def: mcp.NewTool("add_domain",
				mcp.WithDescription(appLine+"Add a domain to one of its lists. "+
					"The app cleans the value the way its Settings field does, so a pasted URL becomes its host name. "+
					"The bypass list takes host names only (no \"/\"); the vpnOnly list also takes an IPv4 CIDR such as 10.0.0.0/8 (never /0 or /1). "+
					"The list is saved before the answer comes back. The kernel routes for the new entry are added in the background, and only while a VPN is connected: "+
					"read list_active_routes or get_logs afterwards to see them. Fails with already_exists when the entry is on that list already."+needs49),
				mcp.WithString("domain", mcp.Required(), mcp.Description("The host name (example.com), or on the vpnOnly list a CIDR.")),
				mcp.WithString("list", mcp.Description(listArg+" Default bypass."), mcp.Enum("bypass", "vpnOnly")),
				write(false, true),
			),
			build: func(a args) (call, error) {
				if _, err := a.required("domain"); err != nil {
					return call{}, err
				}
				return simple("domain.add", "domain", "domain", "list", "list")(a)
			},
		},
		{
			Def: mcp.NewTool("remove_domain", targetArgs(
				appLine+"Remove an entry from one of its domain lists, by id or by domain. "+
					"Its kernel routes are removed in the background."+lookupNote+" Fails with not_found when nothing matches."+needs49,
				write(true, true),
			)...),
			build: targeted("domain.rm"),
		},
		{
			Def: mcp.NewTool("set_domain_enabled", targetArgs(
				appLine+"Turn one domain list entry on or off without removing it. "+
					"A disabled entry stays on its list and gets no kernel routes. Asking for the state it already has is not an error."+lookupNote+needs49,
				mcp.WithBoolean("enabled", mcp.Required(), mcp.Description("true to enable, false to disable.")),
				write(false, true),
			)...),
			build: func(a args) (call, error) {
				on, err := a.requiredBool("enabled")
				if err != nil {
					return call{}, err
				}
				cmd := "domain.disable"
				if on {
					cmd = "domain.enable"
				}
				return targeted(cmd)(a)
			},
		},
	}
}
