// Package client talks to the VPN Bypass control socket: a UNIX stream socket
// that takes one JSON request per line and answers with one JSON line.
//
// Every call opens its own connection, writes one request line, reads one
// response line and closes. The result object is kept as raw JSON, so a field
// a newer app adds still reaches the caller without a change here.
package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"time"
)

const (
	// MaxRequestLine is the largest request line the app accepts, newline
	// excluded. Longer requests are refused here and never sent.
	MaxRequestLine = 64 * 1024
	// ReadTimeout bounds a read verb. The app gives up on a read after 30 s
	// and answers "timeout"; the extra 5 s leaves room for that answer.
	ReadTimeout = 35 * time.Second
	// MutatingTimeout bounds a verb that changes state. The app waits for a
	// change to finish before it answers, so these get longer.
	MutatingTimeout = 120 * time.Second
	// maxResponseLine caps how much of one reply is read, so a broken peer
	// cannot grow memory without bound. The largest real reply (every domain,
	// or 200 log lines) is far below it.
	maxResponseLine = 16 << 20
	// maxSocketPath is the longest path macOS fits in sockaddr_un.sun_path
	// (104 bytes including the terminating NUL).
	maxSocketPath = 103
)

// ErrNotRunning means nothing is listening on the control socket.
var ErrNotRunning = errors.New("VPN Bypass is not running; start the app")

// readVerbs never change the app's state. Every other verb is treated as
// mutating, which is also what the app assumes for a verb it does not know.
var readVerbs = map[string]bool{
	"status":        true,
	"route.list":    true,
	"rule.list":     true,
	"domain.list":   true,
	"service.list":  true,
	"routes.active": true,
	"logs":          true,
}

// IsMutating reports whether a verb can change the app's state.
func IsMutating(cmd string) bool { return !readVerbs[cmd] }

// Request is one line sent to the socket. Every args value is a string, as
// the app requires. Secrets carries a proxy password and nothing else.
type Request struct {
	V       int               `json:"v"`
	Cmd     string            `json:"cmd"`
	Args    map[string]string `json:"args,omitempty"`
	Secrets map[string]string `json:"secrets,omitempty"`
}

type response struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *AppError       `json:"error,omitempty"`
}

// AppError is an error the app answered with, or a request refused before
// it was sent (request_too_large).
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string { return e.Code + ": " + e.Message }

// TimeoutError means no answer arrived before the client-side deadline.
type TimeoutError struct {
	Cmd      string
	After    time.Duration
	Mutating bool
}

func (e *TimeoutError) Error() string {
	if e.Mutating {
		return fmt.Sprintf("VPN Bypass did not answer %s within %s. The change may still be applied: read the current state before you retry", e.Cmd, e.After)
	}
	return fmt.Sprintf("VPN Bypass did not answer %s within %s", e.Cmd, e.After)
}

// Client sends requests to one socket path.
type Client struct {
	SocketPath      string
	ReadTimeout     time.Duration
	MutatingTimeout time.Duration
}

// New returns a client for the socket at path with the default timeouts.
func New(path string) *Client {
	return &Client{SocketPath: path, ReadTimeout: ReadTimeout, MutatingTimeout: MutatingTimeout}
}

// Encode returns the request line for a call, without the trailing newline.
func Encode(cmd string, args, secrets map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	req := Request{V: 1, Cmd: cmd}
	if len(args) > 0 {
		req.Args = args
	}
	if len(secrets) > 0 {
		req.Secrets = secrets
	}
	if err := enc.Encode(req); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// Call sends one request and returns the result object as raw JSON. An
// answer with ok=false comes back as *AppError.
func (c *Client) Call(ctx context.Context, cmd string, args, secrets map[string]string) (json.RawMessage, error) {
	line, err := Encode(cmd, args, secrets)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}
	if len(line) > MaxRequestLine {
		return nil, &AppError{
			Code:    "request_too_large",
			Message: fmt.Sprintf("the request is %d bytes and the control socket accepts at most %d; nothing was sent", len(line), MaxRequestLine),
		}
	}

	if len(c.SocketPath) > maxSocketPath {
		return nil, fmt.Errorf("the socket path %s is too long: %d bytes, and macOS allows at most %d (check VPNB_SOCKET)", c.SocketPath, len(c.SocketPath), maxSocketPath)
	}

	mutating := IsMutating(cmd)
	timeout := c.ReadTimeout
	if mutating {
		timeout = c.MutatingTimeout
	}
	deadline := time.Now().Add(timeout)

	var d net.Dialer
	dialCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	conn, err := d.DialContext(dialCtx, "unix", c.SocketPath)
	if err != nil {
		if errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ECONNREFUSED) {
			return nil, fmt.Errorf("%w (nothing is listening on %s)", ErrNotRunning, c.SocketPath)
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("connect to the VPN Bypass control socket %s: %w", c.SocketPath, err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("set deadline: %w", err)
	}
	// A cancelled context unblocks the read or write at once.
	stop := context.AfterFunc(ctx, func() { _ = conn.SetDeadline(time.Unix(1, 0)) })
	defer stop()

	fail := func(err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			return &TimeoutError{Cmd: cmd, After: timeout, Mutating: mutating}
		}
		return err
	}

	if _, err := conn.Write(append(line, '\n')); err != nil {
		return nil, fail(fmt.Errorf("send %s to VPN Bypass: %w", cmd, err))
	}

	reply, err := bufio.NewReader(io.LimitReader(conn, maxResponseLine+1)).ReadBytes('\n')
	if err != nil {
		switch {
		case errors.Is(err, io.EOF) && len(reply) > maxResponseLine:
			return nil, fmt.Errorf("VPN Bypass answered %s with more than %d bytes; refusing to read further", cmd, maxResponseLine)
		case errors.Is(err, io.EOF):
			return nil, fmt.Errorf("VPN Bypass closed the connection without a complete answer to %s", cmd)
		default:
			return nil, fail(fmt.Errorf("read the answer to %s: %w", cmd, err))
		}
	}

	var resp response
	if err := json.Unmarshal(reply, &resp); err != nil {
		return nil, fmt.Errorf("VPN Bypass answered %s with malformed JSON: %w", cmd, err)
	}
	if !resp.OK {
		if resp.Error == nil {
			return nil, &AppError{Code: "unknown_error", Message: "VPN Bypass answered ok=false without an error"}
		}
		return nil, resp.Error
	}
	return resp.Result, nil
}
