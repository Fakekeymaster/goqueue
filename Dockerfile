# ── Stage 1: Builder ──────────────────────────────────────────
# We use a full Go image to compile the binary.
# This stage is temporary — it won't exist in the final image.
FROM golang:1.26-alpine AS builder

# Set working directory inside the container
WORKDIR /app

# Copy dependency files first — before the source code.
# Docker caches each layer. If go.mod and go.sum haven't changed,
# Docker reuses the cached "go mod download" layer and skips it.
# This makes rebuilds fast when only source code changes.
COPY go.mod go.sum ./
RUN go mod download

# Now copy the rest of the source code
COPY . .

# Build the binary
# CGO_ENABLED=0  — disable C bindings, produces a static binary
# GOOS=linux     — compile for Linux (the container OS)
# -ldflags       — strip debug info to shrink binary size
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o goqueue .


# ── Stage 2: Final image ──────────────────────────────────────
# We use a minimal alpine image — not the Go image.
# The Go image is ~800MB. Alpine is ~10MB.
# We only need the compiled binary, not the Go toolchain.
FROM alpine:3.19

# ca-certificates — needed for HTTPS calls
# tzdata         — needed for time zone handling
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy ONLY the compiled binary from the builder stage.
# Nothing else — no source code, no Go toolchain, no build cache.
COPY --from=builder /app/goqueue .

# Document which port the app uses.
# This doesn't actually publish the port — docker run -p does that.
EXPOSE 8080

# Default command when container starts.
# Can be overridden: docker run goqueue submit --name ...
ENTRYPOINT ["./goqueue"]
CMD ["server"]