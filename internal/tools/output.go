package tools

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

// Output schemas for the read tools. They describe the app's ControlResult
// (VPN-Bypass Sources/VPNBypassCore/CommandRouter.swift and ClassicControl.swift)
// loosely on purpose: no field is required and extra fields are allowed, so a
// field a newer app adds, or one an older app leaves out, never makes a valid
// answer fail a client's check. Optional Swift fields are left out of the JSON
// when nil, never sent as null.

type schema = map[string]any

func obj(props schema) schema { return schema{"type": "object", "properties": props} }
func arr(items schema) schema { return schema{"type": "array", "items": items} }

var (
	str     = schema{"type": "string"}
	boolean = schema{"type": "boolean"}
	integer = schema{"type": "integer"}
)

var outputFields = schema{
	"mode": schema{"type": "string", "description": "Routing mode: bypass, vpnOnly or custom."},
	"routes": arr(obj(schema{
		"id": str, "name": str, "enabled": boolean, "proxyHost": str, "proxyPort": integer,
		"hasProxyUser": boolean, "hasPassword": boolean, "tailscaleExitNode": str, "listenerPort": integer,
	})),
	"defaultRouteId":   str,
	"schemaVersion":    integer,
	"supportedVersion": integer,
	"runtime": obj(schema{
		"helperReady": boolean, "helperState": str, "vpnConnected": boolean, "vpnInterface": str, "vpnType": str,
		"enforcedRouteCount": integer, "enforcing": boolean, "appVersion": str,
	}),
	"rules": arr(obj(schema{
		"id": str, "matchType": str, "pattern": str, "routeId": str, "enabled": boolean, "order": integer,
	})),
	"domains": arr(obj(schema{
		"id": str, "domain": str, "enabled": boolean, "list": str, "isCIDR": boolean, "isWildcard": boolean,
	})),
	"services": arr(obj(schema{
		"id": str, "name": str, "enabled": boolean, "isCustom": boolean, "domainCount": integer, "ipRangeCount": integer,
		"domains": arr(str), "ipRanges": arr(str),
	})),
	"activeRoutes": arr(obj(schema{"destination": str, "gateway": str, "source": str})),
	"logs":         arr(obj(schema{"time": str, "level": str, "message": str})),
}

// output declares an outputSchema with the named top-level result fields.
func output(fields ...string) mcp.ToolOption {
	props := schema{}
	for _, f := range fields {
		p, ok := outputFields[f]
		if !ok {
			panic("tools: no output field " + f)
		}
		props[f] = p
	}
	raw, err := json.Marshal(obj(props))
	if err != nil {
		panic(err)
	}
	return mcp.WithRawOutputSchema(raw)
}
