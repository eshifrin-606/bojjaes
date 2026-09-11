## ADDED Requirements

### Requirement: The image is built from the server's own package

The image SHALL be built by compiling the server's `main` package at `./cmd/server`, from the
repository as checked out, with no step that depends on a `main` package at the repository root.

The repository root is a module, not a command; `go build .` there fails with no Go files. The
image is correct only if it names the package that is the server.

#### Scenario: A clean checkout builds

- **WHEN** the image is built from a clean checkout, with the Go version `fly.toml` supplies as its
  build argument
- **THEN** the build succeeds and the image's entrypoint is the compiled server

### Requirement: The binary is statically linked

The server binary in the image SHALL be statically linked, with cgo disabled at build time, so it
requires no C library or dynamic loader from the stage it runs in.

This SHALL be set explicitly in the build. Whether the builder image happens to have a C toolchain
does not decide it: with a toolchain present, Go links the network and user packages against libc by
default.

#### Scenario: The binary has no dynamic dependencies

- **WHEN** the built binary is inspected in the builder stage
- **THEN** it is reported as not a dynamic executable

#### Scenario: The binary starts in a final stage with no libc

- **WHEN** a container is started from the image
- **THEN** the server process starts and listens, rather than failing to exec for a missing loader

### Requirement: The final stage carries only the binary and what it needs from its host

The image's final stage SHALL contain the server binary and a CA certificate bundle, on a base with
no shell and no package manager. It SHALL NOT contain the repository's source, its lineup files, or
any other file the server reads at runtime.

The server needs one thing from its host: CA roots to verify its only upstream,
`https://api.sleeper.app`. Everything else it reads — lineups, templates, CSS, the timezone
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

### Requirement: The server runs as a non-root user

The image SHALL declare a non-root user, and the server process SHALL run as that user.

The server binds an unprivileged port and writes nothing to disk, so it needs no root privileges.
The user SHALL be declared in the image itself rather than inherited only from the base image's tag,
so a change of base tag cannot silently make the process root.

#### Scenario: The image's configured user is not root

- **WHEN** the image's configuration is inspected
- **THEN** its user is a non-root user and not empty, `root`, or `0`

### Requirement: The server is the container's init process

The image SHALL start the server binary directly as the container's first process, in exec form,
with no shell or wrapper between the runtime and the binary.

`http-server-lifecycle` drains in-flight requests on `SIGTERM`. That guarantee reaches the platform
only if the signal the platform sends goes to the server rather than to a parent that ignores or
does not forward it.

#### Scenario: Stopping the container is a clean, prompt exit

- **WHEN** a running container from the image is stopped with the runtime's default stop signal and
  no request in flight
- **THEN** the container exits with status 0 well before the runtime's kill timeout

#### Scenario: The listen port follows PORT inside the container

- **WHEN** a container is started with `PORT=3000` and port 3000 published
- **THEN** `GET /2025/15` is answered on port 3000

### Requirement: The module download layer is keyed on both module files

The build SHALL copy `go.mod` and `go.sum` together, before the rest of the source, and download
modules in a layer that depends only on those two files. The build SHALL succeed when `go.sum`
does not exist.

A download layer keyed on `go.mod` alone is reused after `go.sum` changes, so verified checksums and
cached modules can disagree. Today the module has no dependencies and so has no `go.sum`; the build
must not require one in order to be ready for one.

#### Scenario: The build succeeds with no go.sum

- **WHEN** the image is built from a checkout that has `go.mod` and no `go.sum`
- **THEN** the build succeeds

#### Scenario: A change to go.sum re-runs the module download

- **WHEN** the image is built, a `go.sum` is added with no other change, and the image is built
  again
- **THEN** the second build runs the module download step afresh rather than reusing it from cache

#### Scenario: A source-only change reuses the module download

- **WHEN** the image is built, a `.go` file under `internal/` is changed, and the image is built
  again
- **THEN** the module download step is reused from cache and only the steps after the source copy
  run
