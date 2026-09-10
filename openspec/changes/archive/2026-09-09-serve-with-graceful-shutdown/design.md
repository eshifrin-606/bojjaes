## Context

`cmd/server/main.go` is currently a single `main()`: it builds the stats cache, registers two routes
on `DefaultServeMux`, and calls `log.Fatal(http.ListenAndServe(":8080", nil))`. Every part of that
line is a problem on Fly. The address is hardcoded while Fly supplies `PORT`; `ListenAndServe`
returns a server with all four timeouts at zero; and `log.Fatal` is the only stop path, so
`SIGTERM` — the signal Fly sends to stop a machine, including on the `auto_stop_machines = 'stop'`
idle path this app is configured for — kills the process with responses half-written.

The package has no test file, because nothing in it is callable. The house rule is a strict
red-green TDD loop, so the shape of this change is largely determined by what has to be reachable
from a test: port resolution, mux construction, server construction, and the run/shutdown loop each
need to be a function that a test can call without the test binary itself listening on a fixed port
or receiving a real signal.

Standard library only. `go 1.26.5`.

## Goals / Non-Goals

**Goals:**

- Listen on `PORT`, defaulting to `8080`.
- Own the mux; keep the two existing routes and their method qualification byte-for-byte in behavior.
- Serve from an explicit `*http.Server` with read-header, read, write, and idle timeouts.
- Drain in-flight requests on `SIGTERM`/`SIGINT` within a bounded grace period; exit non-zero on a
  failed listen or a drain that times out.
- Make each of those testable without a real signal or a fixed port.

**Non-Goals:**

- No change to handler behavior, routes, response shapes, or the cache.
- No readiness/health endpoint, no structured logging, no metrics — separate concerns, separate
  changes.
- No connection limit, no `MaxHeaderBytes` tuning, no per-request timeout middleware. Timeouts here
  are the coarse server-level bound; anything finer is speculative until there is traffic.
- No change to `fly.toml` or the `Dockerfile`.

## Decisions

### Decision: split `main` into `resolveAddr`, `newMux`, `newServer`, and `run`

`main()` becomes a few lines: build the cache, call `run`, and translate its error into an exit code.
Everything else is a function with arguments and a return value.

- `resolveAddr(getenv func(string) string) string` — takes the lookup as a parameter rather than
  calling `os.Getenv` directly, so a test passes a fake map and never touches the process
  environment. `t.Setenv` would work too but serializes against `t.Parallel` and leaks intent into
  the test binary's environment; a function argument is smaller and has no such coupling.
- `newMux(stats api.StatsSource, lineups ...) *http.ServeMux` — pure construction, returns the mux.
  This is what makes the routing table testable with `httptest.NewRecorder` and, incidentally, what
  makes it possible to build twice in one test binary. `DefaultServeMux` panics on a duplicate
  pattern, so a second `newMux` under the old design would take the test binary down.
- `newServer(addr string, h http.Handler) *http.Server` — the one place the timeout constants are
  applied. A test asserts the fields are non-zero and that addr/handler pass through, which is a real
  behavioral assertion (it is exactly the regression `ListenAndServe` represents) without needing a
  socket.
- `run(srv *http.Server, listen func() (net.Listener, error), stop <-chan os.Signal, grace time.Duration) error`
  — the loop, below. `listen` is a parameter for the same reason `getenv` is: a test binds port `0`, or
  hands back a bind failure, without either being the process's real address. Passing an already-open
  listener instead would put the bind outside `run`, where the bind-failure path could not be reached.

Alternative considered: extracting this into a new `internal/httpserver` package. Rejected — this is
the composition root's own concern and there is one binary; a package would add an import edge for
nothing. The file split within `package main` is enough to make it testable.

### Decision: `run` takes a signal channel, not a signal disposition

`run` receives `stop <-chan os.Signal` and `main` supplies the real one via `signal.NotifyContext`
or `signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)`. A test supplies a plain channel and closes
or sends on it. This is the only way to exercise the drain without the test binary raising `SIGTERM`
at itself, which would race with the test framework.

`run`'s shape:

1. Call `listen()`; return its error directly — this is the bind-failure path.
2. Start `srv.Serve(l)` in a goroutine, sending its error on a buffered channel.
3. `select` on that error channel and on `stop`.
4. On a serve error that is not `http.ErrServerClosed`, return it.
5. On `stop`, call `srv.Shutdown(ctx)` with a context bounded by the grace period, and return its
   error. `Shutdown` returns `context.DeadlineExceeded` when the drain times out, which becomes the
   non-zero exit.

### Decision: tests bind port `0` and read back the assigned address

The drain scenarios need a real listener. Hardcoding a port makes tests flaky under parallel runs
and unusable on a busy machine. So a test helper creates `net.Listen("tcp", "127.0.0.1:0")`, hands it
to `run` through the `listen` argument, and reads `l.Addr().String()` for the client URL. The
bind-failure scenario does the inverse: `listen` re-binds an already-open port-0 listener's address.

`httptest.NewServer` is not a substitute — it owns its own server and shutdown, so it can test
handlers but not our shutdown logic, which is the thing under test.

### Decision: timeout values

| Field | Value | Reason |
| --- | --- | --- |
| `ReadHeaderTimeout` | 5s | A slowloris bound. Headers arrive in one packet or the client is not serious. |
| `ReadTimeout` | 10s | Bodies here are a small JSON array of player IDs. |
| `WriteTimeout` | 30s | Must cover a matchup page whose stat fetch is a cache miss: an upstream Sleeper call plus render. The cache's own fetch timeout is the real bound; this sits above it. |
| `IdleTimeout` | 60s | Keep-alive reuse for the page's self-refresh, without holding connections indefinitely on a 256mb machine. |
| shutdown grace | 15s | Comfortably above `WriteTimeout` minus a typical request, and below Fly's own kill timeout so we exit on our terms rather than being `SIGKILL`ed. |

These are constants in the file, named, not literals at the call site.

### Decision: `main` exits via `os.Exit(1)` after logging, not `log.Fatal` mid-flow

`run` returns an error; `main` logs it and exits 1. `log.Fatal` inside the serve path would skip the
drain entirely, which is the bug being fixed.

## Risks / Trade-offs

- **`WriteTimeout` too short cuts a slow cache-miss page mid-render** → 30s is well above the
  cache's upstream timeout; if it ever bites, the symptom is a truncated page and the fix is to
  raise this one constant. Noted here so the next reader knows where to look.
- **A test that binds a real socket is slower and can fail in a sandbox that forbids listening** →
  Confined to the drain and bind-failure tests; port `0` on `127.0.0.1` is the least
  environment-sensitive form available. The timeout and routing tests stay socket-free.
- **The drain-timeout test needs a handler that outlives the grace period** → A 15s grace period
  makes that test 15s long. Mitigation: `run` takes the grace period as a parameter (or the server
  struct carries it), so the test passes a few milliseconds. This is why the grace period is an
  argument rather than a package constant read inside `run`.
- **Splitting `main` into four functions is more surface than a nine-line `main`** → Accepted
  deliberately: the surface is exactly what the TDD loop needs, and each function has one reason to
  change.
- **`Shutdown` does not close hijacked or WebSocket connections** → Not applicable; this server has
  neither.

## Migration Plan

Pure code change in one binary. Deploy is the existing `fly deploy`. Rollback is redeploying the
prior image. `fly.toml` already sets `PORT = '8080'` matching `internal_port`, so behavior on Fly is
unchanged on the happy path and only the stop path improves. Locally, running the binary with no
`PORT` set still listens on `:8080` exactly as today.

## Open Questions

- Should the shutdown grace period be driven by an environment variable so it can be tuned against
  Fly's kill timeout without a redeploy? Defaulting to a constant for now; an env knob is trivial to
  add later if a deploy is ever observed being `SIGKILL`ed.
