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
  cd src
  go mod vendor
  ```

- This ensures all dependencies are available for reproducible builds.

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

### 3. Build and Run

1. **Build the custom collector Docker image (default: linux/amd64):**
   ```sh
   make build
   # or, for a specific platform:
   make build PLATFORM=linux/arm64
   # or, directly:
   docker build --platform=linux/amd64 -t otelcol-custom .
   ```
2. **Run the collector with your config:**
   ```sh
   docker run --rm -v $(pwd)/collector/config.yaml:/otel/config.yaml -p 4317:4317 -p 4318:4318 -p 55679:55679 otelcol-custom
   ```
3. **Send traces to the collector** (e.g., using an OTLP-compatible client).
4. **View generated logs** in your configured log exporter (e.g., OTLP, file, etc).

For more advanced configuration and usage, see the [Full Documentation](SpanEventsToLog_Documentation.md).

---

## Features
- Converts span events (e.g., exceptions) to logs with trace context.
- Supports attribute and event filtering using OTTL.
- Template-based log body generation.
- Designed for use in OpenTelemetry Collector pipelines.

---

## Prerequisites

- **Go 1.21+**: [Install Go](https://golang.org/dl/)
- **Docker**: For building images and cross-platform builds ([Install Docker](https://docs.docker.com/get-docker/))
- **OpenTelemetry Collector Builder (OCB)**: For custom collector distributions
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
├── build.sh                   # Build script
├── test-build.sh              # Test script
├── Dockerfile                 # Docker build for custom collector
├── metadata.yaml              # Metadata for the connector
├── spanEventstoLog.iml        # IDE/project file
├── src/                       # All Go code, tests, and test data
│   ├── connector.go           # Main connector implementation (SpanEventConnector)
│   ├── factory.go             # Factory for creating connector instances
│   ├── config.go              # Configuration and validation
│   ├── go.mod, go.sum         # Go dependencies
│   ├── simple_test.go, standalone_simple_test.go, realistic_standalone_test.go  # Tests
│   ├── REALISTIC_TEST_SUMMARY.md, TEST_FAILURE_ANALYSIS.md  # Test docs
│   ├── realistic_traces/      # Realistic test data (JSON traces, nested by time)
│   │   └── month=05/day=23/hour=13/minute=.../traces_*.json
│   └── internal/              # (if used by Go code)
├── ocb/                       # OCB (OpenTelemetry Collector Builder) files
│   ├── manifest.yaml          # OCB manifest for custom collector
│   ├── OCB_BUILD_README.md    # OCB build instructions
│   └── OCB_MANIFEST_SUMMARY.md# OCB manifest summary
├── deployment/                # Deployment manifests
│   ├── docker-compose.yml     # Local dev/test environment
│   └── k8s-deployment.yaml    # Kubernetes deployment manifest
├── collector/                 # Collector configuration
│   └── config.yaml            # Example collector config
└── .idea/                     # IDE config (can be ignored)
```

---

## Building the Connector

### Native Build
Builds the connector binary for your current platform (output in `dist/`).
```sh
make build
```
- Output: `dist/otelcol-custom`

### Cross-Platform Build
Build for a specific architecture (e.g., x86_64/amd64 or arm64):
```sh
make build PLATFORM=x86_64   # For x86_64/amd64
make build PLATFORM=arm      # For arm64
```
- Output: `dist/otelcol-custom` for the selected platform

---

## Testing

### Test Types
- **Simple/Standalone Tests**: Validate configuration and basic logic (no real data).
- **Realistic Tests**: Validate connector logic using real trace data in `src/realistic_traces/`.
- **All Tests**: Run all Go tests in `src/`.

### Running Tests

- **All tests:**
  ```sh
  make test
  ```
- **Simple/standalone tests only:**
  ```sh
  make test-simple
  ```
- **Realistic tests (real data):**
  ```sh
  make test-real
  ```

**Test Output:**
- Success: All tests should pass with `ok` or `PASS`.
- Failure: Review the output for errors and check your test data in `src/realistic_traces/`.

---

## Generating a Collector Distribution (OCB)

You can generate a custom OpenTelemetry Collector distribution with this connector using the OpenTelemetry Collector Builder (OCB):

```sh
make ocb-dist
# or for a specific platform:
make ocb-dist PLATFORM=x86_64
make ocb-dist PLATFORM=arm
```
- The manifest (`ocb/manifest.yaml`) includes this connector and all required components.
- Output is in the `dist/` directory.
- Requires the `builder` tool (see Prerequisites).

---

## Building and Running the Docker Image

### Build Docker Image
Builds a Docker image for the custom collector:
```sh
make docker-build
# or for a specific platform:
make docker-build PLATFORM=x86_64
make docker-build PLATFORM=arm
```
- Image name: `otelcol-custom:latest`

### Run with Docker Compose
Launch the collector and supporting services (Prometheus, Jaeger, Grafana if configured):
```sh
docker-compose up
```
- Uses `deployment/docker-compose.yml`.

---

## Workflow Summary

1. **Build the connector:**
   ```sh
   make build
   # or cross-platform: make build PLATFORM=x86_64
   ```
2. **Run tests:**
   ```sh
   make test
   # or: make test-simple, make test-real
   ```
3. **Generate collector distribution (optional):**
   ```sh
   make ocb-dist
   ```
4. **Build Docker image:**
   ```sh
   make docker-build
   ```
5. **Run locally (Docker Compose):**
   ```sh
   docker-compose up
   ```
6. **Deploy to Kubernetes (optional):**
   See deployment section below.

---

## Deploying to Kubernetes

1. **Build the collector and Docker image:**
   ```sh
   make ocb-dist
   make docker-build
   # Push your image to a registry if needed
   ```
2. **Apply the Kubernetes manifests:**
   ```sh
   kubectl create namespace monitoring
   kubectl apply -f deployment/k8s-deployment.yaml
   ```
3. **Monitor the deployment:**
   ```sh
   kubectl get pods -n monitoring
   kubectl logs -f deployment/otelcol-custom -n monitoring
   ```

---

## Configuration Example

See `config.yaml` for a full example. The connector is configured as:
```yaml
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
```

---

## Basic Example: Trace Pipeline with SpanEventsToLog

This example shows a minimal pipeline that receives traces, processes them, and uses the `spaneventstolog` connector to generate logs from span events:

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

- **traces pipeline**: Receives traces via OTLP, passes them through the `spaneventstolog` connector, and exports both the original traces and generated logs.
- **logs pipeline**: Receives logs generated by the connector and exports them.

You can expand this example with processors, additional exporters, or more advanced filtering as needed.

---
## Realistic Test Data
- Place your real trace JSON files in `src/realistic_traces/`.
- The realistic tests will use these files to validate connector logic against real-world scenarios.

---

## Troubleshooting
- Ensure Go 1.21+ is installed for builds/tests.
- For cross-platform builds, ensure Docker supports the target architecture.
- For OCB, ensure `builder` is installed: `go install go.opentelemetry.io/collector/cmd/builder@latest`
- For Kubernetes, update image references in `deployment/k8s-deployment.yaml` if pushing to a registry.
- If you encounter permission or architecture errors, check your Docker and Go installation and ensure you are using the correct `PLATFORM` value.
- For test failures, review the test output and ensure your test data is correct.

---

## Questions?
Open an issue or check the documentation for more details on configuration and usage. 