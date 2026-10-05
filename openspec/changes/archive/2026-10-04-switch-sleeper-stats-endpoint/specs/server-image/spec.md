## MODIFIED Requirements

### Requirement: The final stage carries only the binary and what it needs from its host

The image's final stage SHALL contain the server binary and a CA certificate bundle, on a base with
no shell and no package manager. It SHALL NOT contain the repository's source, its lineup files, or
any other file the server reads at runtime.

The server needs one thing from its host: CA roots to verify its only upstream,
`https://api.sleeper.com`. Everything else it reads — lineups, templates, CSS, the timezone
database — is compiled in. A base without CA roots breaks every stat fetch, and a base with a full
userland adds attack surface and size and serves no purpose.

#### Scenario: A matchup page renders with live stats from the container

- **WHEN** a container from the image is running and `GET /2025/15` is requested through its
  published port
- **THEN** the response is 200 and carries both lineups with their point totals, and the server log
  shows no certificate verification error

#### Scenario: The image has no shell

- **WHEN** a container is started from the image with `sh` as its entrypoint
- **THEN** it fails to start because no shell exists

#### Scenario: No source or lineup file is in the image

- **WHEN** the filesystem of a container created from the image is listed
- **THEN** it contains the server binary and the base image's files, and no `.go` or `.csv` file
