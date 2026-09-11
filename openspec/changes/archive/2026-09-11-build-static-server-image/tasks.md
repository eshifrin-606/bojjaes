Each property below is observed failing, against the generated file or a deliberately wrong
intermediate, before it is observed holding. There is no test harness for a `Dockerfile`, so the
red and green are the build and run commands themselves. Build with
`docker build --build-arg GO_VERSION=1.26.5 -t bojjaes .`, the argument `fly.toml` supplies.

## 1. Prerequisites

- [x] 1.1 Start Docker Desktop and confirm `docker version` reports a server version. The daemon
      was not running when this change was proposed.

## 2. Build the server package

- [x] 2.1 Build the generated `Dockerfile` unchanged. Red: the build fails at `go build -v -o
      /run-app .` with no Go files in the root.
- [x] 2.2 Change the build to `go build -v -o /server ./cmd/server`, and the final stage's `COPY` and
      `CMD` to match (`/server`). Leave `debian:bookworm` in place for now. Build: green.
- [x] 2.3 Run `docker run --rm -p 8080:8080 -e PORT=8080 bojjaes` and `curl -i
      localhost:8080/2025/15`. Confirm the server logs that it is listening and the route is
      answered. This is the baseline the next steps must not lose. Observed: 502 "the week's stats
      could not be fetched", with `x509: certificate signed by unknown authority` in the log.
      `debian:bookworm` does not install `ca-certificates` (no `/etc/ssl/certs`), so the generated
      final stage cannot reach Sleeper either. The first 200 is 4.2.

## 3. Static binary

- [x] 3.1 Build only the builder stage (`docker build --target builder ...`) and run `ldd /server`
      in it. Record the output. Expect it to list `libc.so.6`, which shows the default build is
      dynamic with gcc present. If it is already static, record that too: the explicit setting in
      3.3 is still required, so the result does not depend on the builder's toolchain. Observed
      (local arm64): `libc.so.6` and `/lib/ld-linux-aarch64.so.1`; `go env CGO_ENABLED` is `1` and
      gcc is at `/usr/bin/gcc`.
- [x] 3.2 Switch the final stage to `gcr.io/distroless/static-debian12:nonroot` with an exec-form
      `ENTRYPOINT ["/server"]`, without setting cgo. Build and run. Red, if 3.1 showed dynamic
      linking: the container exits at once with `exec /server: no such file or directory`.
- [x] 3.3 Set `CGO_ENABLED=0` on the build step. Rebuild the builder stage. `ldd /server` reports
      `not a dynamic executable`. Build and run the full image: the server logs that it is
      listening. Green.

## 4. Final stage contents

- [x] 4.1 Temporarily make the final stage `FROM scratch`, keeping the `COPY` and `ENTRYPOINT`. Build,
      run, and `curl -i localhost:8080/2025/15`. Red: the request fails, and the server log shows
      `x509: certificate signed by unknown authority` from the Sleeper fetch. This is the evidence
      for rejecting `scratch` in design.md.
- [x] 4.2 Restore `gcr.io/distroless/static-debian12:nonroot`. Rebuild, run, and request
      `/2025/15`. Green: 200, both lineups with point totals, no certificate error in the log.
- [x] 4.3 `docker run --rm --entrypoint sh bojjaes` fails because no shell exists.
- [x] 4.4 `docker create --name bojjaes-fs bojjaes`, then `docker export bojjaes-fs | tar -t`.
      Confirm `server` is present and no `.go` or `.csv` path is. Afterwards `docker rm bojjaes-fs`.

## 5. User and init process

- [x] 5.1 Add `USER nonroot:nonroot` to the final stage. Run `docker image inspect -f
      '{{.Config.User}}' bojjaes` and confirm `nonroot:nonroot`. To see what the line guards
      against, check the same inspect against a build using the plain `static-debian12` tag: its
      user is empty or `0`, which means root. Then restore `:nonroot`. Observed: the plain tag
      without the `USER` line reports `0`.
- [x] 5.2 Start the container detached. Run `time docker stop <id>` and confirm it returns in well
      under the 10s default kill timeout. Then confirm `docker inspect -f '{{.State.ExitCode}}'
      <id>` is `0`: the server is PID 1 and received `SIGTERM`.
- [x] 5.3 Run with `-e PORT=3000 -p 3000:3000` and confirm `curl -i localhost:3000/2025/15` is
      answered.

## 6. Module download layer

- [x] 6.1 Change the first copy to `COPY go.mod go.sum ./` (literal). Build. Red: `COPY` fails
      because `go.sum` does not exist.
- [x] 6.2 Change it to `COPY go.mod go.sum* ./`. Build: green, with no `go.sum` in the checkout.
- [x] 6.3 Rebuild with no change, touch a `.go` file under `internal/` (a whitespace edit), and
      rebuild with `--progress=plain`. Confirm the `go mod download` step is `CACHED` and the build
      step is not. Revert the edit.
- [x] 6.4 Create an empty `go.sum` and rebuild with `--progress=plain`. Confirm the `go mod
      download` step runs rather than being `CACHED`. Delete the `go.sum`.

## 7. Before merge

Merging to `main` deploys, through `.github/workflows/fly-deploy.yml`. Everything in this section
has to hold before the merge, not after.

- [x] 7.1 Run `fly deploy --build-only` and confirm Fly's remote builder produces the `linux/amd64`
      image from `fly.toml`'s build arg. This is the gate: it runs the same build the workflow will,
      without deploying.
- [x] 7.2 Read the final `Dockerfile` against the repo's comment guidance. Delete the generated
      boilerplate. Keep a short why for `CGO_ENABLED=0` (the misleading exec error it prevents),
      for distroless over `scratch` (CA roots for Sleeper), and for `go.sum*` (the build must
      succeed before a `go.sum` exists).
- [x] 7.3 Confirm `git status` shows only `Dockerfile` and `backlog.md` changed outside this change
      directory, and that `go build ./...` and `go test ./...` are still green.
- [x] 7.4 In `backlog.md`, remove the Dockerfile line from "Blocking the first deploy". Reword
      "First deploy, then open it on a phone" to drop "First deploy", because merging this change
      is the first deploy.

## 8. After merge

- [x] 8.1 `gh run watch` the Fly Deploy run for the merge commit and confirm it succeeds. Every
      earlier run of this workflow failed at `go build .`. Observed: run `34549042608` for
      `d4451e5` succeeded. The deploy created two machines in `ord`, Fly's default for an app's
      first deploy.
- [x] 8.2 `curl -i https://bojjaes.fly.dev/2025/15` returns 200 with both lineups and their point
      totals, and `fly logs` shows the server listening on `:8080` and no certificate verification
      error. Observed: bojjaes 68, aroma 41.5. Both machines log `Preparing to run: /server as
      nonroot` and `listening on :8080`, and the Sleeper fetch succeeds.
- [x] 8.3 ~~Leave the app idle until `auto_stop_machines` stops the machine. Confirm in `fly logs`
      and `fly machine status <id>` that it stopped with exit code 0 rather than being killed.~~
      Dropped: not worth holding the change open for an idle window. 5.2 already shows the server
      is PID 1 and exits 0 on `SIGTERM` locally, and Fly runs the same image. Revisit if stopped
      machines ever look killed rather than drained.
