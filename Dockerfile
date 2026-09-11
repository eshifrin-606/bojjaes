ARG GO_VERSION=1
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /usr/src/app
# The glob lets the build succeed before the module has a go.sum.
COPY go.mod go.sum* ./
RUN go mod download && go mod verify
COPY . .
# Set explicitly: with gcc in the builder, the default links libc, and on a base without it the
# container fails with the misleading "exec /server: no such file or directory".
RUN CGO_ENABLED=0 go build -v -o /server ./cmd/server

# Distroless rather than scratch: the server needs CA roots to reach https://api.sleeper.app.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /server /server
# Repeats the tag's user so a switch to the plain tag cannot silently run the server as root.
USER nonroot:nonroot
ENTRYPOINT ["/server"]
