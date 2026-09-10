## Why

The server hardcodes `:8080`, registers routes on the package-level `DefaultServeMux`, and calls `http.ListenAndServe`, which has no timeouts and no way to stop. Fly.io injects the listen port through `PORT` and stops a machine by sending `SIGTERM`; today that signal kills the process mid-request, and a slow or stalled client can hold a connection indefinitely on a 256mb machine.

## What Changes

- Read the listen address from the `PORT` environment variable, falling back to `8080` when it is unset or empty.
- Build the routes on a private `*http.ServeMux` instead of `DefaultServeMux`, so the process owns its routing table and route registration is testable.
- Serve from an explicit `*http.Server` configured with `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, and `IdleTimeout`.
- On `SIGTERM` or `SIGINT`, stop accepting new connections and let in-flight requests finish within a bounded shutdown grace period, then exit; exit non-zero if shutdown times out or the server fails for any reason other than a clean close.
- Extract the wiring out of `main()` into testable functions so the port resolution, mux routing, and shutdown behavior can be driven by tests.

## Capabilities

### New Capabilities
- `http-server-lifecycle`: how the server chooses its listen port, the timeouts it enforces on connections, and how it shuts down when the platform signals it to stop.

### Modified Capabilities

(none — request/response behavior of the existing routes is unchanged)

## Impact

- `cmd/server/main.go`: rewritten wiring; new sibling files for the testable pieces plus their tests.
- No change to `internal/api`, `internal/web`, or any handler contract.
- `fly.toml` already sets `PORT = '8080'` and `internal_port = 8080`; both stay valid.
- Standard library only — no new dependencies.
