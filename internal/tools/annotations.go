package tools

import "github.com/mark3labs/mcp-go/mcp"

// read marks a tool that never changes anything.
func read() mcp.ToolOption {
	return hints(true, false, true)
}

// write marks a tool that changes the app's state. destructive: it removes or
// overwrites something the user set up. idempotent: calling it twice with the
// same arguments changes nothing more than calling it once.
func write(destructive, idempotent bool) mcp.ToolOption {
	return hints(false, destructive, idempotent)
}

func hints(readOnly, destructive, idempotent bool) mcp.ToolOption {
	return mcp.WithToolAnnotation(mcp.ToolAnnotation{
		ReadOnlyHint:    mcp.ToBoolPtr(readOnly),
		DestructiveHint: mcp.ToBoolPtr(destructive),
		IdempotentHint:  mcp.ToBoolPtr(idempotent),
		// The only thing this server talks to is the app on this Mac.
		OpenWorldHint: mcp.ToBoolPtr(false),
	})
}
