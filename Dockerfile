# Stage 1: Build the custom collector with OCB
FROM golang:1.23 as builder

WORKDIR /workspace

# Install OCB (OpenTelemetry Collector Builder)
RUN go install go.opentelemetry.io/collector/cmd/builder@latest

# Copy OCB manifest and all source code (including local connector)
COPY ocb/manifest.yaml .
COPY src/ ./src
COPY . .

# Generate vendor directory for reproducible builds and OCB compatibility
RUN cd src && go mod vendor

RUN /go/bin/builder --skip-compilation --config manifest.yaml
RUN cat dist/go.mod
RUN cat dist/main.go
RUN cd dist && go mod tidy



# Stage 2: Create a minimal runtime image
FROM gcr.io/distroless/base-debian11

WORKDIR /otel
COPY --from=builder /workspace/dist/otelcol-custom_linux_amd64 ./otelcol-custom

# Copy configuration files
COPY collector/config.yaml .
# (Add other configs as needed)

EXPOSE 4317 4318 55679

ENTRYPOINT ["./otelcol-custom", "--config", "config.yaml"] 