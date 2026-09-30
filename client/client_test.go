package client

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/geiserx/vpn-bypass-mcp/internal/fakeapp"
)

func TestCallSendsOneLineAndReturnsRawResult(t *testing.T) {
	app := fakeapp.Start(t, func(r fakeapp.Request) fakeapp.Reply {
		return fakeapp.Reply{Raw: `{"v":1,"ok":true,"result":{"mode":"bypass","futureField":{"x":1}}}`}
	})
	res, err := New(app.Path).Call(context.Background(), "status", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(res), `{"mode":"bypass","futureField":{"x":1}}`; got != want {
		t.Errorf("result = %s, want %s (unknown fields must pass through)", got, want)
	}
	if got, want := app.Lines(), []string{`{"v":1,"cmd":"status"}`}; !equal(got, want) {
		t.Errorf("request lines = %q, want %q", got, want)
	}
}

func TestEncodeArgsAndSecrets(t *testing.T) {
	line, err := Encode("route.set", map[string]string{"port": "24001", "id": "A", "host": "a<b>&c"}, map[string]string{"pass": "s3cret"})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"v":1,"cmd":"route.set","args":{"host":"a<b>&c","id":"A","port":"24001"},"secrets":{"pass":"s3cret"}}`
	if string(line) != want {
		t.Errorf("line = %s\nwant   %s", line, want)
	}
}

func TestAppErrorPassesThrough(t *testing.T) {
	app := fakeapp.Start(t, func(r fakeapp.Request) fakeapp.Reply {
		return fakeapp.Fail("not_found", "no route with that id")
	})
	_, err := New(app.Path).Call(context.Background(), "route.rm", map[string]string{"id": "x"}, nil)
	var ae *AppError
	if !errors.As(err, &ae) || ae.Code != "not_found" || ae.Message != "no route with that id" {
		t.Fatalf("err = %v, want AppError not_found", err)
	}
}

func TestNotRunningWhenSocketMissing(t *testing.T) {
	dir, err := os.MkdirTemp("", "vpnb")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "missing.sock")
	_, err = New(path).Call(context.Background(), "status", nil, nil)
	if !errors.Is(err, ErrNotRunning) {
		t.Fatalf("err = %v, want ErrNotRunning", err)
	}
	if !strings.HasPrefix(err.Error(), "VPN Bypass is not running; start the app") {
		t.Errorf("message = %q", err.Error())
	}
}

func TestNotRunningWhenSocketRefuses(t *testing.T) {
	dir, err := os.MkdirTemp("", "vpnb")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "stale.sock")
	// A socket file with no listener behind it: what a crashed app leaves.
	ln, err := net.ListenUnix("unix", &net.UnixAddr{Name: path, Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	ln.SetUnlinkOnClose(false)
	ln.Close()
	_, err = New(path).Call(context.Background(), "status", nil, nil)
	if !errors.Is(err, ErrNotRunning) {
		t.Fatalf("err = %v, want ErrNotRunning", err)
	}
}

func TestSocketPathTooLong(t *testing.T) {
	path := "/tmp/" + strings.Repeat("d", 120) + "/control.sock"
	_, err := New(path).Call(context.Background(), "status", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "too long") {
		t.Fatalf("err = %v, want a path-too-long error", err)
	}
}

func TestRequestTooLargeIsNotSent(t *testing.T) {
	app := fakeapp.Start(t, func(r fakeapp.Request) fakeapp.Reply { return fakeapp.OK(nil) })
	big := strings.Repeat("a", MaxRequestLine)
	_, err := New(app.Path).Call(context.Background(), "domain.add", map[string]string{"domain": big}, nil)
	var ae *AppError
	if !errors.As(err, &ae) || ae.Code != "request_too_large" {
		t.Fatalf("err = %v, want request_too_large", err)
	}
	if n := len(app.Lines()); n != 0 {
		t.Errorf("the app received %d lines, want 0", n)
	}
}

func TestRequestAtTheLimitIsSent(t *testing.T) {
	app := fakeapp.Start(t, func(r fakeapp.Request) fakeapp.Reply { return fakeapp.OK(map[string]any{}) })
	overhead, _ := Encode("domain.add", map[string]string{"domain": ""}, nil)
	fill := strings.Repeat("a", MaxRequestLine-len(overhead))
	if _, err := New(app.Path).Call(context.Background(), "domain.add", map[string]string{"domain": fill}, nil); err != nil {
		t.Fatalf("a request of exactly %d bytes must be sent: %v", MaxRequestLine, err)
	}
	if lines := app.Lines(); len(lines) != 1 || len(lines[0]) != MaxRequestLine {
		t.Fatalf("got %d lines, want 1 line of %d bytes", len(lines), MaxRequestLine)
	}
}

func TestReadTimeout(t *testing.T) {
	app := fakeapp.Start(t, func(r fakeapp.Request) fakeapp.Reply { return fakeapp.Reply{Hang: true} })
	c := New(app.Path)
	c.ReadTimeout = 100 * time.Millisecond
	c.MutatingTimeout = time.Hour
	_, err := c.Call(context.Background(), "logs", nil, nil)
	var te *TimeoutError
	if !errors.As(err, &te) || te.Mutating || te.After != 100*time.Millisecond {
		t.Fatalf("err = %v, want a read TimeoutError after 100ms", err)
	}
}

func TestMutatingTimeoutWarnsTheChangeMayApply(t *testing.T) {
	app := fakeapp.Start(t, func(r fakeapp.Request) fakeapp.Reply { return fakeapp.Reply{Hang: true} })
	c := New(app.Path)
	c.ReadTimeout = time.Hour
	c.MutatingTimeout = 100 * time.Millisecond
	_, err := c.Call(context.Background(), "domain.add", map[string]string{"domain": "example.com"}, nil)
	var te *TimeoutError
	if !errors.As(err, &te) || !te.Mutating {
		t.Fatalf("err = %v, want a mutating TimeoutError", err)
	}
	if !strings.Contains(err.Error(), "may still be applied") {
		t.Errorf("message = %q, want the warning that the change may still apply", err.Error())
	}
}

func TestDefaultTimeouts(t *testing.T) {
	c := New("/x")
	if c.ReadTimeout != 35*time.Second || c.MutatingTimeout != 120*time.Second {
		t.Fatalf("timeouts = %s / %s, want 35s / 120s", c.ReadTimeout, c.MutatingTimeout)
	}
}

func TestContextCancelStopsTheWait(t *testing.T) {
	app := fakeapp.Start(t, func(r fakeapp.Request) fakeapp.Reply { return fakeapp.Reply{Hang: true} })
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)
	start := time.Now()
	_, err := New(app.Path).Call(ctx, "status", nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("cancel did not stop the wait")
	}
}

func TestClosedWithoutAnswer(t *testing.T) {
	app := fakeapp.Start(t, func(r fakeapp.Request) fakeapp.Reply { return fakeapp.Reply{Close: true} })
	_, err := New(app.Path).Call(context.Background(), "status", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "closed the connection") {
		t.Fatalf("err = %v, want a closed-connection error", err)
	}
}

func TestMalformedAnswer(t *testing.T) {
	app := fakeapp.Start(t, func(r fakeapp.Request) fakeapp.Reply { return fakeapp.Reply{Raw: "not json"} })
	_, err := New(app.Path).Call(context.Background(), "status", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "malformed JSON") {
		t.Fatalf("err = %v, want a malformed-JSON error", err)
	}
}

func TestOkFalseWithoutError(t *testing.T) {
	app := fakeapp.Start(t, func(r fakeapp.Request) fakeapp.Reply { return fakeapp.Reply{Raw: `{"v":1,"ok":false}`} })
	_, err := New(app.Path).Call(context.Background(), "status", nil, nil)
	var ae *AppError
	if !errors.As(err, &ae) || ae.Code != "unknown_error" {
		t.Fatalf("err = %v, want unknown_error", err)
	}
}

func TestIsMutating(t *testing.T) {
	for _, v := range []string{"status", "route.list", "rule.list", "domain.list", "service.list", "routes.active", "logs"} {
		if IsMutating(v) {
			t.Errorf("%s is a read verb", v)
		}
	}
	for _, v := range []string{"domain.add", "domain.rm", "domain.enable", "domain.disable", "service.enable", "service.disable",
		"routes.clear", "refresh", "dns.refresh", "mode", "default", "route.add", "route.set", "route.enable", "route.disable",
		"route.rm", "rule.add", "rule.rm", "something.new"} {
		if !IsMutating(v) {
			t.Errorf("%s must count as mutating", v)
		}
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
