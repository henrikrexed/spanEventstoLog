# Stage 1: Build the custom collector with OCB
FROM --platform=$BUILDPLATFORM golang:1.23 AS builder

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0

WORKDIR /workspace

# Install OCB (OpenTelemetry Collector Builder)
RUN go install go.opentelemetry.io/collector/cmd/builder@latest

# Copy OCB manifest
ARG MANIFEST=ocb/manifest.yaml
COPY ${MANIFEST} ./manifest.yaml

# Copy Go module files first for better layer caching
COPY go.mod go.sum ./

# Copy source code files to the workspace (module root)
COPY connector.go factory.go config.go ./
COPY internal/ ./internal/

# Make git tags available for Go module resolution (if needed)
RUN git config --global --add safe.directory /workspace || true

# Generate vendor directory for reproducible builds and OCB compatibility
RUN go mod vendor

# Build the custom collector (compile for the requested target platform)
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} /go/bin/builder --config manifest.yaml
RUN echo "Built artifacts:" && ls -la dist || true

# Stage 2: Create a minimal runtime image
FROM gcr.io/distroless/base-debian11

ARG TARGETOS
ARG TARGETARCH

WORKDIR /otel
COPY --from=builder /workspace/dist/otelcol-custom ./otelcol-custom

# Copy configuration files
COPY collector/config.yaml .
# (Add other configs as needed)

EXPOSE 4317 4318 55679

ENTRYPOINT ["./otelcol-custom", "--config", "config.yaml"] 