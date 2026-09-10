package main

import (
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestNewServerPassesThroughAddrAndHandler(t *testing.T) {
	h := http.NewServeMux()

	srv := newServer(":9999", h)

	if srv.Addr != ":9999" {
		t.Errorf("Addr = %q, want %q", srv.Addr, ":9999")
	}
	if srv.Handler != h {
		t.Errorf("Handler = %v, want the handler given", srv.Handler)
	}
}

func TestNewServerSetsReadHeaderTimeout(t *testing.T) {
	if got := newServer(":0", http.NewServeMux()).ReadHeaderTimeout; got == 0 {
		t.Error("ReadHeaderTimeout is zero; a silent client would hold the connection")
	}
}

func TestNewServerSetsReadTimeout(t *testing.T) {
	if got := newServer(":0", http.NewServeMux()).ReadTimeout; got == 0 {
		t.Error("ReadTimeout is zero; a slow body would be read forever")
	}
}

func TestNewServerSetsWriteTimeout(t *testing.T) {
	if got := newServer(":0", http.NewServeMux()).WriteTimeout; got == 0 {
		t.Error("WriteTimeout is zero; a stalled reader would hold the response open")
	}
}

func TestNewServerSetsIdleTimeout(t *testing.T) {
	if got := newServer(":0", http.NewServeMux()).IdleTimeout; got == 0 {
		t.Error("IdleTimeout is zero; keep-alive connections would never be reclaimed")
	}
}

// The field is only worth setting if it actually disconnects a silent client.
// The real bound is 5s, too slow for a test, so only the duration is synthetic.
func TestServerClosesAConnectionThatSendsNoHeaders(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := newServer(l.Addr().String(), http.NewServeMux())
	srv.ReadHeaderTimeout = 50 * time.Millisecond
	go srv.Serve(l)
	t.Cleanup(func() { srv.Close() })

	conn, err := net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))

	if _, err := io.ReadAll(conn); err != nil {
		t.Errorf("connection was not closed by the server: %v", err)
	}
}
