# 🚀 SpanEventsToLog Connector

**An OpenTelemetry Collector Connector for Transforming Span Events into Logs**

---

| Status  | Data flow      | Stability | Distributions | Code owners     |
|---------|----------------|-----------|---------------|-----------------|
| Beta    | traces → logs  | Beta      | contrib       | henrikrexed     |

---

## Build Requirements

- **Vendor directory is not tracked:** The `vendor` directory is intentionally excluded from version control. If you need it (for OCB or reproducible builds), generate it locally.
- **No Go or OCB required on host:** If you use Docker or Podman (recommended), you do not need Go or OCB installed locally.
- **Go 1.23+ required for local builds:** If you wish to build outside of Docker/Podman, ensure you have Go 1.23 or newer installed.
- **Container engine:** You can use either Docker or Podman for all build and run commands (see Makefile and usage examples).

---

## Dependency Management

- The `vendor` directory is **not** tracked in git.
- If you need to build with OCB or require vendored dependencies, generate it locally:

  ```sh
  go mod vendor
  ```

- This ensures all dependencies are available for reproducible builds.

---

## Getting Started

### 1. Minimal Example Configuration

Add the following to your OpenTelemetry Collector config (e.g., `collector/config.yaml`):

```yaml
connectors:
  spaneventstolog:
    span_conditions:
      - "attributes[\"http.status_code\"] == 503"
    event_conditions:
      - "name == \"exception\""
    include_span_attributes: true
    include_event_attributes: true
    log_level: "Error"
    log_body_template: "Error in {{.SpanName}}: {{.EventName}}"
```

### 2. Basic Pipeline Example

```yaml
receivers:
  otlp:
    protocols:
      grpc:
      http:

connectors:
  spaneventstolog:
    span_conditions:
      - "attributes[\"http.status_code\"] == 503"
    event_conditions:
      - "name == \"exception\""
    include_span_attributes: true
    include_event_attributes: true
    log_level: "Error"
    log_body_template: "Error in {{.SpanName}}: {{.EventName}}"

exporters:
  otlp:
    endpoint: "http://localhost:4317"
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      connectors: [spaneventstolog]
      exporters: [otlp]
    logs:
      receivers: [spaneventstolog]
      exporters: [otlp]
```

### 3. Build and Run (with Makefile)

1. **Build the Go binary locally:**
   ```sh
   make build-binary
   ```
   Produces: `dist/spanEventstoLog`

2. **Build Docker/Podman images:**
   ```sh
   # Full release (uses ocb/manifest.yaml)
   make build
   # or, for a specific platform:
   make build PLATFORM=linux/arm64
   # or, with Podman:
   make build CONTAINER_ENGINE=podman
   # or, with a custom version tag:
   make build VERSION=1.2.3
   # Combine options:
   make build CONTAINER_ENGINE=podman PLATFORM=linux/arm64 VERSION=1.2.3
   ```
   Produces: `hrexed/otelcol-spanconnector:<version>` and `hrexed/otelcol-spanconnector:latest`

   Minimal release (simple collector with only this connector, uses `ocb/manifest_minimal.yaml`):
   ```sh
   make release-minimal
   make release-minimal PLATFORM=linux/amd64
   make release-minimal CONTAINER_ENGINE=podman
   make release-minimal VERSION=1.2.3
   ```
   Produces: `hrexed/otelcol-spanconnector:<version>-minimal`

3. **Run the collector with your config:**
   ```sh
   $(CONTAINER_ENGINE) run --rm -v $(pwd)/collector/config.yaml:/otel/config.yaml -p 4317:4317 -p 4318:4318 -p 55679:55679 hrexed/otelcol-spanconnector:0.1.0
   # or, for a custom version:
   $(CONTAINER_ENGINE) run --rm -v $(pwd)/collector/config.yaml:/otel/config.yaml -p 4317:4317 -p 4318:4318 -p 55679:55679 hrexed/otelcol-spanconnector:1.2.3
   # minimal image:
   $(CONTAINER_ENGINE) run --rm -v $(pwd)/collector/config.yaml:/otel/config.yaml -p 4317:4317 -p 4318:4318 -p 55679:55679 hrexed/otelcol-spanconnector:0.1.0-minimal
   ```

4. **Send traces to the collector** (e.g., using an OTLP-compatible client).
5. **View generated logs** in your configured log exporter (e.g., OTLP, file, etc).

For more advanced configuration and usage, see the [Full Documentation](SpanEventsToLog_Documentation.md).

---

## Build, Test, and Versioning with Makefile

### Available Makefile Targets

- **Build:**
  ```sh
  make build-binary                    # Build Go binary locally
  make build [CONTAINER_ENGINE=docker|podman] [PLATFORM=linux/amd64|linux/arm64|...] [VERSION=x.y.z]
  make release-minimal [CONTAINER_ENGINE=docker|podman] [PLATFORM=linux/amd64|linux/arm64|...] [VERSION=x.y.z]
  ```

- **Test:**
  ```sh
  make test                            # Run all Go tests
  make test-real                       # Run realistic tests with real data
  ./test-build.sh                      # Comprehensive Docker image testing
  ```

- **Version bump:**
  ```sh
  make bump-patch   # 0.1.0 -> 0.1.1
  make bump-minor   # 0.1.0 -> 0.2.0
  make bump-major   # 0.1.0 -> 1.0.0
  ```

- **Clean:**
  ```sh
  make clean                           # Remove build artifacts
  ```

### Makefile Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `VERSION` | `0.1.0` | Version tag for Docker images |
| `PLATFORM` | `linux/$(host_arch)` | Target platform for builds |
| `CONTAINER_ENGINE` | `docker` | Container engine (docker/podman) |
| `DIST_DIR` | `dist` | Output directory for binaries |

---

## Purpose

The **SpanEventsToLog** connector (implemented as `SpanEventConnector`) is an OpenTelemetry Collector connector that transforms span events (such as exceptions or custom events) into log records. This enables:
- **Error/event monitoring**: Convert exceptions and important span events into logs for alerting and analysis.
- **Log enrichment**: Preserve trace context and span attributes in logs.
- **Flexible filtering**: Use OTTL to select which span events become logs.
- **Realistic validation**: Test against real-world trace data for robust production use.

---

## Configuration Options

| Option                   | Type      | Description                                                                                                    | Required | Example |
|--------------------------|-----------|----------------------------------------------------------------------------------------------------------------|----------|---------|
| `span_conditions`        | []string  | OTTL conditions for filtering spans. If empty, all spans are processed.                                        | No       | `["attributes[\"http.status_code\"] == 503"]` |
| `event_conditions`       | []string  | OTTL conditions for filtering individual span events. If empty, all events are processed.                      | No       | `["name == \"exception\""]` |
| `include_span_attributes`| bool      | Include span attributes in the generated log record.                                                           | No       | `true`  |
| `include_event_attributes`| bool     | Include event attributes in the generated log record.                                                          | No       | `true`  |
| `log_level`              | string    | Severity level for generated log records. One of: Trace, Debug, Info, Warn, Error, Fatal.                      | No       | `"Error"` |
| `log_body_template`      | string    | Go template for the log body. Placeholders: `{{.EventName}}`, `{{.SpanName}}`, `{{.EventAttributes}}`, `{{.SpanAttributes}}`. | No       | `"Error in {{.SpanName}}: {{.EventName}}"` |

### Validation Rules
- At least one of `span_conditions` or `event_conditions` must be specified.
- `log_level` must be one of: Trace, Debug, Info, Warn, Error, Fatal (case-sensitive).
- `log_body_template` can only reference: `.EventName`, `.SpanName`, `.EventAttributes`, `.SpanAttributes`.
- OTTL conditions are validated at startup; invalid expressions will cause startup failure.

---

## Features
- Converts span events (e.g., exceptions) to logs with trace context.
- Supports attribute and event filtering using OTTL.
- Template-based log body generation.
- Designed for use in OpenTelemetry Collector pipelines.

---

## Prerequisites

- **Go 1.23+**: [Install Go](https://golang.org/dl/)
- **Docker or Podman**: For building images and cross-platform builds
  - [Install Docker](https://docs.docker.com/get-docker/)
  - [Install Podman](https://podman.io/getting-started/installation)
- **OpenTelemetry Collector Builder (OCB)**: For custom collector distributions (optional, included in Docker build)
  ```sh
  go install go.opentelemetry.io/collector/cmd/builder@latest
  ```
- (Optional) **Kubernetes**: For deployment/testing in a cluster

---

## Project Structure

```
spanEventstoLog/
├── README.md                  # Project documentation (this file)
├── Makefile                   # Build, test, and distribution automation
├── Dockerfile                 # Docker build for custom collector
├── metadata.yaml              # Metadata for the connector
├── spanEventstoLog.iml        # IDE/project file
├── connector.go               # Main connector implementation (SpanEventConnector)
├── factory.go                 # Factory for creating connector instances
├── config.go                  # Configuration and validation
├── go.mod, go.sum            # Go dependencies
├── standalone_simple_test.go  # Basic configuration and validation tests
├── realistic_standalone_test.go  # Realistic tests with real-world data
├── test-build.sh              # Comprehensive Docker image testing script
├── REALISTIC_TEST_SUMMARY.md  # Test documentation and analysis
├── TEST_FAILURE_ANALYSIS.md   # Test failure analysis
├── internal/                  # Internal packages (if used by Go code)
├── ocb/                       # OCB (OpenTelemetry Collector Builder) files
│   ├── manifest.yaml          # OCB manifest for custom collector
│   ├── manifest_minimal.yaml  # Minimal OCB manifest
│   ├── OCB_BUILD_README.md    # OCB build instructions
│   └── OCB_MANIFEST_SUMMARY.md# OCB manifest summary
├── deployment/                # Deployment manifests
│   ├── docker-compose.yml     # Local dev/test environment
│   └── k8s-deployment.yaml    # Kubernetes deployment manifest
├── collector/                 # Collector configuration
│   └── config.yaml            # Example collector config
└── dist/                      # Build output directory (created by builds)
```

---

## Testing

### Test Types

- **Simple/Standalone Tests** (`standalone_simple_test.go`): Validate configuration and basic logic (no real data).
- **Realistic Tests** (`realistic_standalone_test.go`): Validate connector logic using real trace data with exception events.
- **All Tests**: Run all Go tests in the root directory.

### Running Tests

- **All tests:**
  ```sh
  make test
  ```

- **Realistic tests (real data):**
  ```sh
  make test-real
  ```

**Test Output:**
- Success: All tests should pass with `ok` or `PASS`.
- Failure: Review the output for errors and check your test data.

### Test Coverage

The realistic tests are based on analysis of real span data and cover:
- **Exception Monitoring**: Connection errors, HTTP 503 status codes
- **Service-Specific Monitoring**: Cart service, product service endpoints
- **OTTL Filtering**: Span and event condition validation
- **Template Generation**: Log body template processing
- **Real-World Scenarios**: Based on actual trace data from load generator service

For detailed test analysis, see [REALISTIC_TEST_SUMMARY.md](REALISTIC_TEST_SUMMARY.md).

### Comprehensive Testing with test-build.sh

The `test-build.sh` script provides comprehensive validation of the Docker image build and runtime functionality:

```sh
# Basic usage (uses existing image)
./test-build.sh

# Build image first, then test
./test-build.sh --build-image

# Use minimal manifest/image
./test-build.sh --minimal --build-image

# Test specific platform
./test-build.sh --platform linux/arm64 --build-image

# Use custom image tag
./test-build.sh --image myregistry/otelcol-custom:latest

# Show help
./test-build.sh --help
```

**What the script tests:**
- ✅ Docker image build (if `--build-image` is specified)
- ✅ Container startup and health checks
- ✅ OTLP endpoints accessibility (4317, 4318)
- ✅ Health check endpoint (55679)
- ✅ Platform-specific builds
- ✅ Minimal vs full image variants

**Script options:**
| Option | Description |
|--------|-------------|
| `--build-image` | Build Docker image via Makefile before testing |
| `--minimal` | Use minimal manifest/image (`release-minimal`) |
| `--platform <plat>` | Target platform (default: host architecture) |
| `--image <name:tag>` | Use custom image tag instead of Makefile defaults |
| `-h, --help` | Show help information |

**Example workflow:**
```sh
# 1. Build and test full image
./test-build.sh --build-image

# 2. Build and test minimal image for ARM64
./test-build.sh --minimal --platform linux/arm64 --build-image

# 3. Test existing image
./test-build.sh --image hrexed/otelcol-spanconnector:latest
```

---

## Deployment

### Local Development with Docker Compose

The project includes a complete local development environment:

```sh
# Start the full stack (collector, Jaeger, Prometheus, Grafana)
cd deployment
docker-compose up

# Or start just the collector
docker-compose up otelcol-custom
```

**Services included:**
- **otelcol-custom**: Custom OpenTelemetry Collector with SpanEventsToLog connector
- **jaeger**: Trace visualization (port 16686)
- **prometheus**: Metrics collection (port 9090)
- **grafana**: Visualization dashboard (port 3000, admin/admin)

### Kubernetes Deployment

1. **Create namespace:**
   ```sh
   kubectl create namespace monitoring
   ```

2. **Deploy the collector:**
   ```sh
   kubectl apply -f deployment/k8s-deployment.yaml
   ```

3. **Monitor the deployment:**
   ```sh
   kubectl get pods -n monitoring
   kubectl logs -f deployment/otelcol-custom -n monitoring
   ```

**Kubernetes Features:**
- **ConfigMap**: Collector configuration with SpanEventsToLog connector
- **Deployment**: Scalable collector deployment with resource limits
- **Service**: OTLP gRPC/HTTP endpoints and health check
- **ServiceAccount**: Kubernetes RBAC integration
- **Processors**: Memory limiter, batching, K8s attributes, resource enrichment
- **Exporters**: OTLP gRPC and HTTP with TLS configuration

### Production Considerations

- **Resource Limits**: Configured with 2Gi memory, 1000m CPU limits
- **Health Checks**: Liveness and readiness probes on port 55679
- **Security**: TLS configuration for OTLP exporters
- **Monitoring**: Integrated with Prometheus and Grafana
- **Logging**: File log receiver for application logs

---

## Configuration Examples

### Basic Error Monitoring

```yaml
connectors:
  spaneventstolog:
    span_conditions:
      - "attributes[\"http.status_code\"] == 503"
      - "status.code == STATUS_CODE_ERROR"
    event_conditions:
      - "name == \"exception\""
    include_span_attributes: true
    include_event_attributes: true
    log_level: "Error"
    log_body_template: "Error in {{.SpanName}}: {{.EventName}}"
```

### Service-Specific Monitoring

```yaml
connectors:
  spaneventstolog:
    span_conditions:
      - "IsMatch(attributes[\"http.url\"], \"/api/cart\")"
    event_conditions:
      - "name == \"exception\""
      - "attributes[\"exception.type\"] == \"requests.exceptions.ConnectionError\""
    include_span_attributes: true
    include_event_attributes: true
    log_level: "Error"
    log_body_template: "Cart Service Error: {{.EventName}} in {{.SpanName}} - {{.EventAttributes.exception.message}}"
```

### Comprehensive Pipeline

```yaml
receivers:
  otlp:
    protocols:
      grpc:
      http:

connectors:
  spaneventstolog:
    span_conditions:
      - "attributes[\"http.status_code\"] == 503"
      - "status.code == STATUS_CODE_ERROR"
    event_conditions:
      - "name == \"exception\""
      - "attributes[\"exception.type\"] == \"requests.exceptions.ConnectionError\""
    include_span_attributes: true
    include_event_attributes: true
    log_level: "Error"
    log_body_template: "Connection Error in {{.SpanName}}: {{.EventName}} - {{.EventAttributes.exception.message}}"

exporters:
  otlp:
    endpoint: "http://localhost:4317"
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      connectors: [spaneventstolog]
      exporters: [otlp]
    logs:
      receivers: [spaneventstolog]
      exporters: [otlp]
```

---

## Troubleshooting

### Build Issues
- **Go version**: Ensure Go 1.23+ is installed for local builds
- **Docker/Podman**: Verify container engine is running and accessible
- **Platform issues**: Use `PLATFORM` variable for cross-platform builds
- **Vendor directory**: Run `go mod vendor` if needed for OCB builds

### Test Issues
- **Test failures**: Review test output and check test data
- **Realistic tests**: Ensure real trace data is available for realistic tests
- **API compatibility**: Check for upstream OpenTelemetry API changes

### Deployment Issues
- **Kubernetes**: Verify namespace exists and RBAC permissions
- **Docker Compose**: Check port conflicts and volume mounts
- **Configuration**: Validate OTTL expressions and template syntax
- **Resource limits**: Adjust memory/CPU limits based on workload

### Common Commands
```sh
# Check build status
make build-binary

# Run tests
make test

# Clean build artifacts
make clean

# Check container status
docker ps
kubectl get pods -n monitoring

# View logs
docker logs otelcol-custom
kubectl logs -f deployment/otelcol-custom -n monitoring
```

---

## Questions?
Open an issue or check the documentation for more details on configuration and usage. 