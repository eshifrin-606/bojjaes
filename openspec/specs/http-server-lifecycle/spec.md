# http-server-lifecycle

## Purpose

Govern how the server process comes up, holds connections, and goes down: where it learns the port
to listen on, what routing table it owns, the timeouts it imposes on every connection, and what it
does when the platform tells it to stop.

This capability exists because the process runs on a platform that decides these things for it. Fly
injects the listen port through `PORT` and stops a machine by sending `SIGTERM`, on a single-CPU
256mb machine where an idle connection is a real cost. It governs only the lifecycle — the listen
address, the mux the routes are registered on, the connection timeouts, and the drain — and nothing
about what any route answers: the page is `matchup-page`'s, and it is unchanged by the server around
it.
## Requirements
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

The mux SHALL serve exactly one route: `GET /{season}/{week}` for the matchup page. The pattern SHALL
be method-qualified, so a non-matching method is answered by the mux with 405 rather than reaching
the handler.

Building the mux SHALL be a function of its dependencies and SHALL have no other effect — no
listening, no logging, no reading of the environment — so a test can build one and drive it with
`httptest`.

#### Scenario: The matchup route is reachable on the owned mux

- **WHEN** a `GET` for a season and week is served by a freshly built mux
- **THEN** the matchup handler answers it, and `DefaultServeMux` has no route registered

#### Scenario: The matchup route rejects the wrong method

- **WHEN** a `POST` for a season and week is served by a freshly built mux
- **THEN** the mux answers 405 without invoking the matchup handler

#### Scenario: The retired scores path is not served

- **WHEN** a `POST /scores` is served by a freshly built mux
- **THEN** the mux answers 404, because no route matches the path

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

### Requirement: Lineup seeding completes before the server listens

When a lineup volume is configured, the process SHALL complete boot seeding (per `lineup-source`)
before it opens its listener, so no request is ever answered from a partly seeded store.

If seeding fails — the volume or its manifest cannot be read or written — the process SHALL log the
error and exit with a non-zero status without opening the listener. A volume the app cannot write
to therefore blocks reads as well as writes (ADR 0005 consequences), which is preferred to serving
a store of unknown state.

With no volume configured, nothing is seeded and startup proceeds as before.

#### Scenario: The tree handed to the server is already seeded

- **WHEN** the server's lineup tree is built with a volume configured over an empty directory
- **THEN** by the time the tree is returned, it already serves every embedded week

#### Scenario: A seeding failure stops startup

- **WHEN** `LINEUP_VOLUME` names a directory the process cannot create `lineups/` or the manifest
  in
- **THEN** building the lineup tree returns an error, and the process exits non-zero without
  listening

#### Scenario: No volume, no seeding

- **WHEN** `LINEUP_VOLUME` is unset
- **THEN** the lineup tree is the embedded tree and no file is written

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

