package tools

import "github.com/mark3labs/mcp-go/mcp"

// read marks a tool that never changes anything.
func read() mcp.ToolOption {
	return hints(true, false, true)
}

// write marks a tool that changes the app's state. destructive: it changes
// where some of the Mac's traffic goes (around the VPN, through it, or through
// a proxy), or removes or overwrites something the user set up; clients should
// confirm these, because a prompt-injected agent could use them to reroute
// traffic. idempotent: calling it twice with the same arguments changes
// nothing more than calling it once.
func write(destructive, idempotent bool) mcp.ToolOption {
	return hints(false, destructive, idempotent)
}

// external marks a tool whose arguments name a host outside this Mac (a
// proxy) that the Mac will then send traffic to. Pass it after read or write.
func external() mcp.ToolOption {
	return mcp.WithOpenWorldHintAnnotation(true)
}

func hints(readOnly, destructive, idempotent bool) mcp.ToolOption {
	return mcp.WithToolAnnotation(mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(readOnly),
		DestructiveHint: mcp.ToBoolPtr(destructive),
		IdempotentHint:  mcp.ToBoolPtr(idempotent),
		// The tool itself talks only to the app on this Mac; external() says
		// otherwise for the tools that point traffic at a host they name.
		OpenWorldHint: mcp.ToBoolPtr(false),
	})
}
