package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"time"
)

// run takes the signal channel as a parameter, not a signal disposition, so a
// test can drive the drain without raising SIGTERM at the test binary. listen
// is a parameter for the same reason: a test binds port 0, or hands back a bind
// failure, without either being the process's real address.
func run(srv *http.Server, listen func() (net.Listener, error), stop <-chan os.Signal, grace time.Duration) error {
	l, err := listen()
	if err != nil {
		return err
	}

	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(l) }()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-stop:
		ctx, cancel := context.WithTimeout(context.Background(), grace)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}
