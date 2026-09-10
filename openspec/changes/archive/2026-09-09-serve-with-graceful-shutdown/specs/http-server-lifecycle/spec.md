## ADDED Requirements

### Requirement: The listen port comes from the environment, with a local default

The address the server listens on SHALL be derived from the `PORT` environment variable. When `PORT`
holds a value, the server SHALL listen on that port on all interfaces. When `PORT` is unset or
empty, the server SHALL listen on port `8080`.

The platform, not the binary, decides where the process listens: Fly injects `PORT` and routes its
proxy to it. A hardcoded port makes the binary correct only by coincidence.

Resolution SHALL be a pure reading of the variable's value — no validation of the port number, no
network call — so it can be exercised without binding a socket.

#### Scenario: PORT is set

- **WHEN** `PORT` holds `3000`
- **THEN** the resolved listen address is `:3000`

#### Scenario: PORT is unset

- **WHEN** `PORT` is not present in the environment
- **THEN** the resolved listen address is `:8080`

#### Scenario: PORT is present but empty

- **WHEN** `PORT` is set to the empty string
- **THEN** the resolved listen address is `:8080`, the same as if it were unset

### Requirement: Routes are registered on a mux the process owns

The server's routes SHALL be registered on a `*http.ServeMux` created by the program and handed to
its server, not on the package-level `DefaultServeMux`.

Global mux registration makes the routing table process-wide state: it cannot be built twice in one
test binary without a duplicate-pattern panic, and any linked package can add a route to it
unnoticed. An owned mux makes the routing table a value that can be constructed and exercised
directly.

The mux SHALL serve the same routes as before this change: `POST /scores` for the batch scoring API
and `GET /{season}/{week}` for the matchup page. Method-qualified patterns SHALL be kept, so a
non-matching method is answered by the mux with 405 rather than reaching a handler.

Building the mux SHALL be a function of its dependencies and SHALL have no other effect — no
listening, no logging, no reading of the environment — so a test can build one and drive it with
`httptest`.

#### Scenario: The matchup route is reachable on the owned mux

- **WHEN** a `GET` for a season and week is served by a freshly built mux
- **THEN** the matchup handler answers it, and `DefaultServeMux` has no route registered

#### Scenario: The scores route rejects the wrong method

- **WHEN** a `GET /scores` is served by a freshly built mux
- **THEN** the mux answers 405 without invoking the batch handler

#### Scenario: Two muxes can be built in one process

- **WHEN** the mux is built twice
- **THEN** both are usable and neither construction panics

### Requirement: Connections are bounded by explicit timeouts

The server SHALL be an `*http.Server` configured with a read-header timeout, a read timeout, a write
timeout, and an idle timeout. It SHALL NOT be started through a helper that leaves these at their
zero value.

`http.ListenAndServe` gives a server with every timeout unset, which means a client that opens a
connection and sends nothing holds a goroutine and a file descriptor until it chooses to leave. On a
256mb single-CPU machine a handful of such connections is the whole budget.

The write timeout SHALL be long enough to cover a matchup page whose upstream stat fetch is a cache
miss.

#### Scenario: A constructed server carries non-zero timeouts

- **WHEN** the server is constructed for a given address and handler
- **THEN** its read-header, read, write, and idle timeouts are all non-zero, and its address and
  handler are the ones given

#### Scenario: A client that sends no request headers is disconnected

- **WHEN** a client connects to the running server and sends no bytes for longer than the
  read-header timeout
- **THEN** the server closes the connection rather than holding it open

### Requirement: SIGTERM drains in-flight requests before the process exits

The server SHALL stop on receipt of `SIGTERM` or `SIGINT`. On receipt it SHALL stop accepting new
connections and allow requests already in flight to finish, within a bounded grace period, before
the process exits.

Fly stops a machine by sending `SIGTERM` and waiting; a process that dies on the signal cuts every
open response mid-body. Draining turns a deploy or an idle auto-stop into a request the reader never
notices.

A request in flight when the signal arrives SHALL receive its complete response. A request arriving
after shutdown has begun SHALL NOT be served.

If the grace period elapses with requests still in flight, the server SHALL abandon them and the
process SHALL exit non-zero. A server that stops for any reason other than a clean shutdown SHALL
also exit non-zero; a clean shutdown SHALL exit zero.

Shutdown SHALL be driven by a caller-supplied signal rather than by the process's real signal
disposition, so the drain can be exercised in a test without signalling the test binary.

#### Scenario: An in-flight request completes after the signal

- **WHEN** a request is being served and the shutdown signal arrives before the handler returns
- **THEN** the client receives that response in full and the run returns without error

#### Scenario: A request arriving after the signal is not served

- **WHEN** the shutdown signal has been delivered and a new request is attempted
- **THEN** the connection is refused or fails rather than being answered

#### Scenario: A handler that outlasts the grace period fails the run

- **WHEN** a handler is still running when the shutdown grace period elapses
- **THEN** the run returns an error rather than reporting a clean stop

#### Scenario: A listen failure is reported

- **WHEN** the server cannot bind its address because the port is already in use
- **THEN** the run returns that error rather than blocking or reporting a clean stop
