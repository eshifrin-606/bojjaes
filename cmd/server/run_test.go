package main

import (
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"
)

func listenLocal(t *testing.T) net.Listener {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return l
}

func TestRunServesRequests(t *testing.T) {
	l := listenLocal(t)
	srv := newServer(l.Addr().String(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok")
	}))
	stop := make(chan os.Signal, 1)
	go run(srv, listenerFunc(l), stop, time.Second)
	t.Cleanup(func() { srv.Close() })

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + l.Addr().String())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
}

func TestRunDrainsAnInFlightRequest(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{})
	l := listenLocal(t)
	srv := newServer(l.Addr().String(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		io.WriteString(w, "complete")
	}))
	stop := make(chan os.Signal, 1)
	runErr := make(chan error, 1)
	go func() { runErr <- run(srv, listenerFunc(l), stop, 5*time.Second) }()
	t.Cleanup(func() { srv.Close() })

	client := &http.Client{Timeout: 5 * time.Second}
	body := make(chan string, 1)
	go func() {
		resp, err := client.Get("http://" + l.Addr().String())
		if err != nil {
			body <- "error: " + err.Error()
			return
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		body <- string(b)
	}()

	<-entered
	stop <- os.Interrupt

	// Returning here would mean the process exits with the response half-written.
	select {
	case err := <-runErr:
		t.Fatalf("run returned (%v) while a request was still in flight", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(release)

	if got := <-body; got != "complete" {
		t.Errorf("body = %q, want %q", got, "complete")
	}
	if err := <-runErr; err != nil {
		t.Errorf("run returned %v, want nil", err)
	}
}

func TestRunStopsAcceptingAfterTheSignal(t *testing.T) {
	l := listenLocal(t)
	srv := newServer(l.Addr().String(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok")
	}))
	addr := l.Addr().String()
	stop := make(chan os.Signal, 1)
	runErr := make(chan error, 1)
	go func() { runErr <- run(srv, listenerFunc(l), stop, 5*time.Second) }()

	stop <- os.Interrupt
	if err := <-runErr; err != nil {
		t.Fatalf("run returned %v, want nil", err)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	if resp, err := client.Get("http://" + addr); err == nil {
		resp.Body.Close()
		t.Errorf("request after shutdown was answered %d, want a failure", resp.StatusCode)
	}
}

func TestRunFailsWhenTheDrainTimesOut(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	l := listenLocal(t)
	srv := newServer(l.Addr().String(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
	}))
	stop := make(chan os.Signal, 1)
	runErr := make(chan error, 1)
	go func() { runErr <- run(srv, listenerFunc(l), stop, 20*time.Millisecond) }()
	t.Cleanup(func() { close(release); srv.Close() })

	client := &http.Client{Timeout: 5 * time.Second}
	go func() {
		if resp, err := client.Get("http://" + l.Addr().String()); err == nil {
			resp.Body.Close()
		}
	}()

	<-entered
	stop <- os.Interrupt

	if err := <-runErr; err == nil {
		t.Error("run returned nil, want the drain timeout reported as an error")
	}
}

func listenerFunc(l net.Listener) func() (net.Listener, error) {
	return func() (net.Listener, error) { return l, nil }
}

func TestRunReportsABindFailure(t *testing.T) {
	taken := listenLocal(t)
	t.Cleanup(func() { taken.Close() })
	srv := newServer(taken.Addr().String(), http.NewServeMux())
	stop := make(chan os.Signal, 1)

	err := run(srv, func() (net.Listener, error) {
		return net.Listen("tcp", taken.Addr().String())
	}, stop, time.Second)

	if err == nil {
		t.Error("run returned nil, want the bind failure on an address already in use")
	}
}
