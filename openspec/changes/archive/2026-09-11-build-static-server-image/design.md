## Context

`fly launch --no-deploy` (commit `99337dc`) generated a two-stage `Dockerfile`:

```dockerfile
ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm as builder
WORKDIR /usr/src/app
COPY go.mod ./
RUN go mod download && go mod verify
COPY . .
RUN go build -v -o /run-app .

FROM debian:bookworm
COPY --from=builder /run-app /usr/local/bin/
CMD ["run-app"]
```

It fails at `go build .`: the root holds `go.mod` and no Go files, and the server is `./cmd/server`.
The rest of the file is written for an app that reads from its host. This server reads almost
nothing from its host:

- lineups come from `//go:embed data` in `internal/lineup` (`lineup-source`: "The deployed binary
  carries its lineup tree");
- the page template and CSS are embedded in `internal/web`;
- `internal/web` imports `time/tzdata`, so `America/Chicago` resolves without `/usr/share/zoneinfo`;
- the listen port comes from `PORT` (`http-server-lifecycle`), which `fly.toml` sets to 8080;
- the process writes nothing to disk — the stats cache is in memory.

The one host dependency left is outbound TLS. The server's only upstream is
`https://api.sleeper.app`, and Go's `crypto/x509` on Linux loads roots from the system bundle
(`/etc/ssl/certs/ca-certificates.crt`).

The platform side is settled: `fly.toml` passes `GO_VERSION = '1.26.5'` as a build arg, runs on a
256mb shared-CPU machine, and stops idle machines with `SIGTERM`.

Deploys are already automatic. `fly launch` added `.github/workflows/fly-deploy.yml`, which runs
`flyctl deploy --remote-only` on every push to `main`. The repo secret works: the runs reach the
build. But every run since the file was added has failed at `go build .` with
`no Go files in /usr/src/app`. The first push that carries a working `Dockerfile` is therefore the
first deploy, and this change is that push.

This is not Go code, so the house red-green loop does not apply to it the way it applies to a
package. It gets the same treatment `scripts/*.sh` gets: no test harness. Each property is still
observed failing, against the generated file or a deliberately wrong intermediate, before it is
observed holding. `tasks.md` is ordered that way.

The Docker daemon is not running on the development machine at the time of writing; the tasks
start by bringing it up.

## Goals / Non-Goals

**Goals:**

- An image that builds from a clean checkout with the build arg `fly.toml` supplies.
- A statically linked binary, set explicitly rather than inherited from the builder's toolchain.
- A final stage holding the binary and CA roots, with no shell, running as non-root.
- The server as PID 1, so `SIGTERM` reaches the drain `http-server-lifecycle` already implements.
- A module download layer that stays correct when a `go.sum` first appears.
- A merge that deploys a serving page. The image is proven on Fly's builder before merge, and the
  workflow run it triggers is confirmed serving afterwards.

**Non-Goals:**

- `fly.toml` changes, including the health check. That check is its own backlog line and needs a
  route decision this change should not make.
- Changes to the deploy workflow. It keeps deploying on every push to `main`. Skipping docs-only
  pushes with `paths-ignore` is its own backlog line.
- Binary-size tuning (`-ldflags "-s -w"`, `-trimpath`). The base image choice is where the size is.
  Stripping a ~10MB binary is not worth changing what a panic trace looks like before the first
  deploy.
- Trimming the build context in `.dockerignore` (`docs/`, `openspec/`, `.git`). It affects the
  upload and nothing in the image, because the final stage copies one file.
- Keeping the `GO_VERSION` default in the `Dockerfile` in step with `go.mod`. `fly.toml` supplies
  the real value, and the Go toolchain directive in `go.mod` already refuses a builder that is too
  old.
- Multi-arch images. `CGO_ENABLED=0` makes the binary build for whatever platform the builder
  targets. Fly builds `linux/amd64`, and a local build on Apple Silicon produces `arm64`, which is
  fine for running it locally.

## Decisions

### Decision: `gcr.io/distroless/static-debian12:nonroot` as the final stage

The backlog line named two options, scratch and distroless. The CA-roots requirement decides it.

| Option | CA roots | Shell | Non-root user | Size over binary | Verdict |
| --- | --- | --- | --- | --- | --- |
| `debian:bookworm` (generated) | **no** | yes | no (root) | ~120MB | Rejected: a full userland that nothing uses, and without `ca-certificates` installed it fails the Sleeper fetch just as `scratch` does |
| `scratch` | **no** | no | none defined | 0 | Rejected: every Sleeper fetch fails `x509: certificate signed by unknown authority` |
| `scratch` + `COPY --from=builder /etc/ssl/certs/ca-certificates.crt` + a hand-written `/etc/passwd` | yes | no | hand-rolled | ~0.2MB | Rejected: re-implements distroless static by hand, and the CA bundle is frozen at whatever the builder image had |
| `alpine` | yes | yes | must create | ~8MB | Rejected: a shell and `apk` for no runtime use |
| **`distroless/static-debian12:nonroot`** | **yes** | **no** | **yes (65532)** | **~2MB** | **Chosen** |

Distroless `static` is exactly the scratch-plus-certs row, maintained upstream. It carries
`ca-certificates`, `tzdata`, `/etc/passwd` with a `nonroot` user, and a writable `/tmp`, with no
shell and no package manager. The `tzdata` is redundant with `time/tzdata`, and harmless. The server
keeps importing `time/tzdata`, so its correctness never depends on the base image.

The Debian release in the tag only sets how old the CA bundle and tzdata are; the binary is static
and links nothing from it.

### Decision: `CGO_ENABLED=0` set in the build, not inferred

`golang:*-bookworm` ships gcc. With a C toolchain present, `go build` defaults to `CGO_ENABLED=1`, and
`net` (DNS resolution) and `os/user` link against glibc. The result is a dynamically linked binary
that needs `/lib64/ld-linux-x86-64.so.2`. On distroless `static` that loader does not exist, and the
container fails at exec with the misleading `exec /server: no such file or directory`.

Setting `CGO_ENABLED=0` on the build forces the pure-Go resolver and a static binary, whatever
toolchain the builder image carries. Alternative considered: `-tags netgo,osusergo`. It gets to the
same place for these two packages, but it leaves cgo on for anything added later, so a future cgo
dependency would fail at runtime rather than at build.

### Decision: `COPY go.mod go.sum* ./` before `go mod download`

`COPY go.mod go.sum ./` fails today, because no `go.sum` exists and `COPY` refuses a missing
literal source. The glob `go.sum*` matches zero or one file. `go.mod` is always present, so the
instruction as a whole matches something and succeeds either way. When `go.sum` appears, it becomes
part of the layer's cache key and the download re-runs.

Alternatives considered:

- Commit an empty `go.sum` now. It works, but it adds a file whose only job is to serve the
  `Dockerfile`, and a later `go mod tidy` is not obliged to keep it.
- Drop the download layer and let `go build` fetch. That is correct with zero dependencies, but it
  throws away the layer caching this backlog line is about the moment there is one.

`go mod download && go mod verify` stays as generated.

### Decision: binary at `/server`, exec-form `ENTRYPOINT`, explicit `USER`

```dockerfile
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /server /server
USER nonroot:nonroot
ENTRYPOINT ["/server"]
```

- **Exec form** makes the binary PID 1, so `SIGTERM` from Fly reaches `run`'s drain. Shell form is
  impossible here anyway because there is no shell, but exec form says so on purpose rather than by
  accident.
- **`ENTRYPOINT`, not `CMD`**: the image is this one program. `CMD` would let a stray argument in
  `fly.toml`'s `processes` replace the binary rather than be passed to it.
- **`USER nonroot:nonroot`** repeats what the `:nonroot` tag already sets. It is there so a later
  switch to the plain `static` tag, which runs as root, does not silently change who the process is.
- **`/server`**, not `/usr/local/bin/run-app`: the image holds one file of its own, and the path is
  shorter to read in `fly logs` and `docker inspect`.
- No `EXPOSE`. The port belongs to `PORT` and `fly.toml`'s `internal_port`, and `EXPOSE 8080` would
  be a third place that could disagree.
- No `WORKDIR` in the final stage. Nothing is read relative to it.

### Decision: keep the builder stage otherwise as generated

`ARG GO_VERSION=1` and `golang:${GO_VERSION}-bookworm` stay. `fly.toml` already pins the version,
and the builder's distro is irrelevant to a static binary. `go build -v` stays too, since the
package list is useful in a remote builder's log.

## Risks / Trade-offs

- **[`CGO_ENABLED=0` forecloses cgo dependencies]** → The one foreseeable one is SQLite for the
  storage ADR 0004 anticipates on a Fly volume. The pure-Go `modernc.org/sqlite` works under this
  image. `mattn/go-sqlite3` would need this decision revisited, and with it the base image
  (`distroless/base` carries glibc). With cgo explicitly off, a cgo dependency fails at build time,
  where the reason is visible.
- **[No shell means `fly ssh console` has nothing to run]** → Accepted. There is nothing on the
  machine to inspect but the binary, and logs go to `fly logs`. If live debugging is ever needed,
  switch the tag to `:debug-nonroot`, which adds a busybox shell, for one deploy.
- **[The distroless tag floats]** → `:nonroot` moves when upstream rebuilds, which is how the CA
  bundle and tzdata get updated, and it also means two builds a month apart can differ. Accepted:
  the base is referenced by tag, not pinned by digest (`@sha256:…`). Upstream CA and tzdata updates
  arrive with no manual step, and nothing yet calls for diffing two images. If a rebuild ever
  breaks, pin the digest of the last good image then.
- **[An embedded asset could be missed and only fail in the container]** → The final stage has no
  source tree to fall back on, so a missed embed would surface on the first request in the image.
  The container smoke test in `tasks.md` requests a real matchup page. That exercises the lineup
  tree, the template, and the CSS together.
- **[The local verification image is arm64, Fly's is amd64]** → The binary is pure Go with no
  platform-conditional code in this repo, so a local arm64 run exercises the same program. The
  `fly deploy --build-only` task confirms the amd64 build on Fly's builder before merge, and the
  post-merge smoke test runs the amd64 image itself.
- **[The merge deploys before anyone has seen the page on Fly]** → There is no health check, and `/`
  and unknown weeks still render as errors, both separate backlog lines. That is acceptable for a
  URL nobody has been sent yet. The pre-merge `--build-only` gate runs the same build on the same
  builder the workflow uses, so the build cannot surprise the merge; and the post-merge tasks request a real matchup page on `bojjaes.fly.dev` before
  the URL goes to anyone.

## Migration Plan

There is no running deployment to migrate. Merging to `main` is the deploy: the workflow builds on
Fly's remote builder and starts the app's first machine.

There is no earlier release to roll back to. If the deployed page is wrong, push a fix, and that
push redeploys. To take the app down in the meantime, run `fly scale count 0`. Reverting this
commit is not a rollback: the generated file it restores does not build, so the workflow fails and
whatever is running stays running.
