// Command vpn-bypass-mcp is an MCP server, on stdio, for VPN Bypass: it lets
// an AI agent read and change the app through its local control socket.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/geiserx/vpn-bypass-mcp/client"
	"github.com/geiserx/vpn-bypass-mcp/config"
	"github.com/geiserx/vpn-bypass-mcp/internal/tools"
	"github.com/geiserx/vpn-bypass-mcp/version"
	"github.com/mark3labs/mcp-go/server"
)

const usage = `vpn-bypass-mcp %s

MCP server for VPN Bypass (macOS). It speaks MCP on stdin and stdout; an MCP
client starts it. It reaches the app through the app's control socket.

Environment:
  VPNB_SOCKET                 control socket path (default:
                              ~/Library/Application Support/VPNBypass/control.sock)
  VPN_BYPASS_MCP_READ_ONLY=1  register only the tools that change nothing
`

// newServer builds the MCP server with its tools.
func newServer(cfg config.Config) (*server.MCPServer, int) {
	s := server.NewMCPServer(
		"vpn-bypass-mcp",
		version.Version,
		server.WithToolCapabilities(false),
		server.WithInstructions(tools.Instructions),
		server.WithRecovery(),
	)
	n := tools.Register(s, client.New(cfg.SocketPath), cfg.ReadOnly)
	return s, n
}

func run(ctx context.Context, argv []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(argv) > 0 {
		switch argv[0] {
		case "--help", "-h", "help":
			fmt.Fprintf(stdout, usage, version.String())
			return 0
		case "--version", "-v", "version":
			fmt.Fprintln(stdout, version.String())
			return 0
		default:
			fmt.Fprintf(stderr, "vpn-bypass-mcp: unknown argument %q (try --help)\n", argv[0])
			return 2
		}
	}

	// stdout carries the protocol, so every log line goes to stderr.
	logger := log.New(stderr, "vpn-bypass-mcp: ", log.LstdFlags)
	cfg := config.Load()
	s, n := newServer(cfg)
	mode := "read and write"
	if cfg.ReadOnly {
		mode = "read-only"
	}
	logger.Printf("%s starting on stdio, %d tools (%s), socket %s", version.String(), n, mode, cfg.SocketPath)

	stdio := server.NewStdioServer(s)
	stdio.SetErrorLogger(logger)
	if err := stdio.Listen(ctx, stdin, stdout); err != nil && ctx.Err() == nil {
		logger.Printf("stdio server stopped: %v", err)
		return 1
	}
	return 0
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
