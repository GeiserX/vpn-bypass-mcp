// Package fakeapp is an in-process stand-in for the VPN Bypass control socket,
// for tests. It frames requests the way the app does (one JSON line in, one
// JSON line out, 64 KiB per request line, v=1 only), records every request
// line it receives, and answers with whatever the test's handler returns.
package fakeapp

import (
	"bufio"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const maxRequestLine = 64 * 1024

// Request is the decoded request line.
type Request struct {
	V       int               `json:"v"`
	Cmd     string            `json:"cmd"`
	Args    map[string]string `json:"args"`
	Secrets map[string]string `json:"secrets"`
}

// Error is the error object of a response.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Reply is what a handler answers. Raw, when set, is written verbatim
// (a newline is added); Hang keeps the connection open without answering;
// Close hangs up without answering.
type Reply struct {
	OK     bool
	Result any
	Error  *Error
	Raw    string
	Hang   bool
	Close  bool
}

// Handler answers one request.
type Handler func(Request) Reply

// OK answers ok=true with result.
func OK(result any) Reply { return Reply{OK: true, Result: result} }

// Fail answers ok=false with an error code and message.
func Fail(code, message string) Reply {
	return Reply{Error: &Error{Code: code, Message: message}}
}

// Server is a running fake socket.
type Server struct {
	Path    string
	handler Handler
	ln      net.Listener
	mu      sync.Mutex
	lines   []string
	done    chan struct{}
}

// Start listens on a fresh socket path (short, because sun_path holds only
// 104 bytes on macOS) and stops when the test ends.
func Start(t testing.TB, h Handler) *Server {
	t.Helper()
	dir, err := os.MkdirTemp("", "vpnb")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "c.sock")
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	s := &Server{Path: path, handler: h, ln: ln, done: make(chan struct{})}
	go s.serve()
	t.Cleanup(func() {
		close(s.done)
		ln.Close()
		os.RemoveAll(dir)
	})
	return s
}

// Lines returns every request line received so far, newline stripped.
func (s *Server) Lines() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.lines...)
}

func (s *Server) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReaderSize(conn, 4096)
	for {
		line, tooLarge, err := readLine(r)
		if tooLarge {
			// Recorded too, so a test can tell that an oversized request was sent.
			s.mu.Lock()
			s.lines = append(s.lines, "<request over 65536 bytes>")
			s.mu.Unlock()
			write(conn, Fail("request_too_large", "request line exceeded 65536-byte limit"))
			return
		}
		if err != nil {
			return
		}
		s.mu.Lock()
		s.lines = append(s.lines, string(line))
		s.mu.Unlock()

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			write(conn, Fail("bad_request", "malformed JSON"))
			continue
		}
		if req.V == 0 {
			req.V = 1
		}
		if req.V != 1 {
			write(conn, Fail("unsupported_version", "only v=1 is supported"))
			continue
		}
		reply := s.handler(req)
		switch {
		case reply.Close:
			return
		case reply.Hang:
			select {
			case <-s.done:
			case <-time.After(30 * time.Second):
			}
			return
		}
		if !write(conn, reply) {
			return
		}
	}
}

// readLine reads one newline-terminated line, refusing one longer than the
// app's limit.
func readLine(r *bufio.Reader) (line []byte, tooLarge bool, err error) {
	for {
		chunk, isPrefix, err := r.ReadLine()
		if err != nil {
			return nil, false, err
		}
		line = append(line, chunk...)
		if len(line) > maxRequestLine {
			return nil, true, nil
		}
		if !isPrefix {
			return line, false, nil
		}
	}
}

func write(conn net.Conn, reply Reply) bool {
	var out []byte
	if reply.Raw != "" {
		out = []byte(reply.Raw)
	} else {
		body := map[string]any{"v": 1, "ok": reply.OK}
		if reply.Result != nil {
			body["result"] = reply.Result
		}
		if reply.Error != nil {
			body["error"] = reply.Error
		}
		var err error
		if out, err = json.Marshal(body); err != nil {
			return false
		}
	}
	_, err := conn.Write(append(out, '\n'))
	return err == nil
}

// Legacy answers like the VPN Bypass 4.8.x apps: the verbs that shipped then
// work, every 4.9.0 verb is unknown_command, and status has no appVersion.
func Legacy(req Request) Reply {
	switch req.Cmd {
	case "status":
		return OK(map[string]any{
			"mode": "bypass", "schemaVersion": 2, "supportedVersion": 1, "routes": []any{},
			"runtime": map[string]any{"helperReady": true, "helperState": "Ready", "vpnConnected": false, "enforcedRouteCount": 0, "enforcing": false},
		})
	case "route.list":
		return OK(map[string]any{"routes": []any{}})
	case "rule.list":
		return OK(map[string]any{"rules": []any{}})
	case "route.add", "route.set", "route.enable", "route.disable", "route.rm", "rule.add", "rule.rm", "mode", "default":
		return OK(map[string]any{"message": "ok"})
	}
	return Fail("unknown_command", "unknown command: "+req.Cmd)
}
