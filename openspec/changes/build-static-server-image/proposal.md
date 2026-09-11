## Why

The `Dockerfile` that `fly launch` generated cannot produce an image, so nothing can be deployed. Its
`go build .` compiles the repo root, which has no `main` package; the server is `./cmd/server`.
Past that one line, the file builds for a machine this app doesn't run on. The binary links against
glibc because cgo is left on. It ships on a full `debian:bookworm` userland, running as root. And it
copies `go.mod` without `go.sum`, so its module-download layer goes stale the moment a dependency is
added.

The server's previous changes made a minimal image possible: lineups, templates, CSS, and the
timezone database are all compiled in (`internal/lineup` embeds its tree; `internal/web` imports
`time/tzdata`), and `PORT` comes from the environment. The image only has to carry the binary and
whatever the binary still needs from its host. What it still needs is one thing: the CA roots
to verify `https://api.sleeper.app`.

## What Changes

- The builder stage compiles `./cmd/server` rather than `.`.
- The build sets `CGO_ENABLED=0`, so the binary is statically linked and needs no libc in the
  final stage.
- The final stage becomes `gcr.io/distroless/static-debian12:nonroot` in place of
  `debian:bookworm`: no shell or package manager, CA roots present, and a non-root user. Plain
  `scratch` is rejected because it has no CA bundle and the server's only upstream is HTTPS.
- `go.sum` is copied alongside `go.mod` before `go mod download`. It must not fail today, when no
  `go.sum` exists, and must invalidate the download layer once one does.
- The final stage holds only the binary: no source, no lineup files, and no working directory the
  server depends on.
- Merging this change is the first deploy. `fly launch` also added
  `.github/workflows/fly-deploy.yml`, which runs `flyctl deploy --remote-only` on every push to
  `main`. Every run so far has failed at the generated `go build .`. The workflow stays as it is,
  so the image has to build on Fly's builder before merge (`fly deploy --build-only`), and the
  post-merge run is where it is confirmed serving.
- Out of scope, each its own backlog line: the `[[http_service.checks]]` in `fly.toml`, what `/` and
  an unknown week render, and skipping the deploy for docs-only pushes.

## Capabilities

### New Capabilities

- `server-image`: what the container image the server is deployed as must be — built from the
  server's own package, a statically linked binary, a final stage with nothing but that binary and
  the CA roots it needs, run as a non-root user, and a dependency layer keyed on both module files.

### Modified Capabilities

None. `http-server-lifecycle` already requires the `PORT` reading the image relies on, and
`lineup-source` already requires that no lineup is read from the working directory. This change
relies on both and changes neither.

## Impact

- `Dockerfile`: rewritten; no other file in the build changes.
- `.github/workflows/fly-deploy.yml`: unchanged. This change's merge is the first run that deploys.
- `backlog.md`: the Dockerfile line comes out, and "First deploy, then open it on a phone" loses its
  "First deploy".
- `fly.toml`: unchanged. Its `GO_VERSION = '1.26.5'` build arg still selects the builder image, and
  `internal_port = 8080` still matches the `PORT` it sets.
- `.dockerignore`: unchanged. The generated file already excludes a locally built `server` binary
  from the build context.
- No Go code, no new module dependency, no change to any route or response.
- Verification needs a local Docker daemon for the property checks, and `fly deploy --build-only`
  on Fly's remote builder as the gate before merge.
