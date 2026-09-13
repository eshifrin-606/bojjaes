## MODIFIED Requirements

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
