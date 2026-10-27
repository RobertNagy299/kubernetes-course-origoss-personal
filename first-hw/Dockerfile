
# Stage 1: Build the Go application.
FROM golang:1.27 AS builder

WORKDIR /src

# Download dependencies first to improve build-cache reuse.
COPY go.mod go.sum ./
RUN go mod download

# Copy source files only into the build stage.
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build a static Linux binary.
# CGO_ENABLED=0 avoids a dependency on the system C library.
# -trimpath removes local filesystem paths from the binary.
# -ldflags="-s -w" removes debug information to reduce binary size.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/server \
    ./cmd/server

# Stage 2: Minimal runtime image.
FROM scratch AS runtime

COPY --from=builder /out/server /server

USER 65532:65532

EXPOSE 8080

ENTRYPOINT ["/server"]