## 1. Listen address from PORT

- [x] 1.1 Add `cmd/server/addr.go` with `resolveAddr(getenv func(string) string) string` returning
      the empty string, and `cmd/server/addr_test.go` with a test for `PORT=3000`. Run it: it fails
      on the value (`"" != ":3000"`), not on a missing symbol.
- [x] 1.2 Make `resolveAddr` return `":" + getenv("PORT")`. Run: green.
- [x] 1.3 Add a test for `PORT` absent expecting `":8080"`. Run: red — it returns `":"`.
- [x] 1.4 Add the empty-value fallback to `defaultPort`. Run: green.
- [x] 1.5 Add a test for `PORT` present but empty expecting `":8080"`. Run: it should already pass —
      if it does not, the fallback is keyed on presence rather than emptiness; fix that.

## 2. An owned mux

- [x] 2.1 Add `cmd/server/mux.go` with `newMux(stats api.StatsSource, lineups lineup.Source)
      *http.ServeMux` returning an empty `http.NewServeMux()`, and `cmd/server/mux_test.go` driving
      `GET /2025/15` through it with `httptest.NewRecorder` and a stub stats source. Run: red on the
      404 the empty mux returns.
- [x] 2.2 Register `GET /{season}/{week}` on the mux. Run: green.
- [x] 2.3 Add a test that `GET /scores` yields 405. Run: red — the empty-of-that-route mux answers
      404.
- [x] 2.4 Register `POST /scores` with the method-qualified pattern. Run: green, and confirm the
      405 comes from the mux rather than a handler.
- [x] 2.5 Add a test that calls `newMux` twice in one test function and serves a request through
      each. Run: green — it would panic against `DefaultServeMux`, and this is the test that pins
      the mux as owned.

## 3. Explicit server with timeouts

- [x] 3.1 Add `cmd/server/server.go` with `newServer(addr string, h http.Handler) *http.Server`
      returning `&http.Server{}`, and `cmd/server/server_test.go` asserting `Addr` and `Handler`
      pass through. Run: red on the zero values.
- [x] 3.2 Set `Addr` and `Handler`. Run: green.
- [x] 3.3 Add a test asserting `ReadHeaderTimeout` is non-zero. Run: red.
- [x] 3.4 Add the named `readHeaderTimeout` constant and set the field. Run: green.
- [x] 3.5 Repeat 3.3–3.4 once per remaining field — `ReadTimeout`, `WriteTimeout`, `IdleTimeout` —
      each as its own red-then-green pair with its own named constant. Values per design.md.
- [x] 3.6 Add a socket-level test: bind `127.0.0.1:0`, serve with a very short read-header timeout,
      open a raw `net.Dial` connection, send no bytes, and assert the server closes it. Run: red
      first against a server built with the field zeroed, then green with `newServer`.

## 4. Run and drain

- [x] 4.1 Add `cmd/server/run.go` with `run(srv *http.Server, l net.Listener, stop <-chan os.Signal,
      grace time.Duration) error` that serves and returns `nil` immediately, plus
      `cmd/server/run_test.go` asserting a request against the running server succeeds. Run: red —
      nothing is being served.
- [x] 4.2 Implement the serve goroutine and the `select` on the error channel and `stop`. Run:
      green.
- [x] 4.3 Add a test: a handler that blocks on a channel; fire `stop` mid-request; release the
      handler; assert the client reads the complete response body and `run` returned `nil`. Run:
      red if the shutdown path is not yet draining.
- [x] 4.4 Implement `srv.Shutdown` with a `context.WithTimeout(ctx, grace)` and return its error,
      mapping `http.ErrServerClosed` to `nil`. Run: green.
- [x] 4.5 Add a test that a request attempted after `stop` has been delivered is not served. Run and
      confirm green — this pins that shutdown stops accepting, not just drains.
- [x] 4.6 Add a test with a handler that sleeps past a millisecond-scale `grace`, asserting `run`
      returns a non-nil error. Run: red if `Shutdown`'s error is being swallowed; green once it is
      returned.
- [x] 4.7 Add a test that a bind failure — a second server pointed at an already-listening address —
      is returned from `run` rather than blocking. Run: red, then green by returning the serve
      goroutine's non-`ErrServerClosed` error.

## 5. Wire up main

- [x] 5.1 Rewrite `main()` to build the cache, call `resolveAddr(os.Getenv)`, `newMux`, `newServer`,
      register the real signal channel for `SIGTERM` and `SIGINT`, and call `run`; on a non-nil
      error log it and `os.Exit(1)`. Remove the `addr` constant, the `http.Handle` calls, and the
      `log.Fatal(http.ListenAndServe(...))`.
- [x] 5.2 Run `go build ./...` and `go test ./...`; confirm the whole suite is green.
- [x] 5.3 Run the binary locally with no `PORT` and confirm `GET http://localhost:8080/2025/15`
      still renders; run it with `PORT=3000` and confirm it listens there instead.
- [x] 5.4 With the binary running and a request in flight, send it `SIGTERM` and confirm the
      response completes and the process exits zero.
- [x] 5.5 Review the finished files against the repo's comment guidance: keep the "why" comments
      (why the cache is wrapped once, why the mux is owned, why the signal channel is a parameter),
      delete anything that restates the code.
