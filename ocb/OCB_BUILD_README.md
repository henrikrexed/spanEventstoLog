# 🚀 Custom OpenTelemetry Collector with SpanEventsToLog Connector

**An OpenTelemetry Collector distribution including the SpanEventsToLog connector and all required components.**

---

| Status  | Data flow      | Stability | Distributions | Code owners     |
|---------|----------------|-----------|---------------|-----------------|
| Beta    | traces → logs  | Beta      | contrib       | henrikrexed     |

---

## Purpose

This custom collector distribution includes the **SpanEventsToLog** connector, which transforms span events (such as exceptions or custom events) into log records. This enables:
- **Error/event monitoring**
- **Log enrichment** with trace context
- **Flexible filtering** using OTTL
- **Realistic validation** with real-world trace data

---

## Configuration Options

| Option                   | Type      | Description                                                                                                    | Required | Example |
|--------------------------|-----------|----------------------------------------------------------------------------------------------------------------|----------|---------|
| `span_conditions`        | []string  | OTTL conditions for filtering spans. If empty, all spans are processed.                                        | No       | `["attributes[\"http.status_code\"] == 503"]` |
| `event_conditions`       | []string  | OTTL conditions for filtering individual span events. If empty, all events are processed.                      | No       | `["name == \"exception\""]` |
| `include_span_attributes`| bool      | Include span attributes in the generated log record.                                                           | No       | `true`  |
| `include_event_attributes`| bool     | Include event attributes in the generated log record.                                                          | No       | `true`  |
| `log_level`              | string    | Severity level for generated log records. One of: Trace, Debug, Info, Warn, Error, Fatal.                      | No       | `"Error"` |
| `log_body_template`      | string    | Go template for the log body. Placeholders: `{{.EventName}}`, `{{.SpanName}}`, `{{.EventAttributes}}`, `{{.SpanAttributes}}`. | No       | `"Error in {{.SpanName}}: {{.EventName}}" |

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
      exporters: [otlp,spaneventstolog]
    logs:
      receivers: [spaneventstolog]
      exporters: [otlp]
```

### 3. Build and Run

1. **Build the custom collector:**
   ```sh
   chmod +x build.sh
   ./build.sh
   ```
2. **Run the collector with your config:**
   ```sh
   ./dist/otelcol-custom_$(go env GOOS)_$(go env GOARCH) --config collector/config.yaml
   ```
3. **Send traces to the collector** (e.g., using an OTLP-compatible client).
4. **View generated logs** in your configured log exporter (e.g., OTLP, file, etc).

For more advanced configuration and usage, see the [Main README](../README.md) and [Full Documentation](../SpanEventsToLog_Documentation.md).

---

## 📋 Components Included

### **Receivers**
- **OTLP Receiver**: Accepts traces, metrics, and logs via gRPC (4317) and HTTP (4318)
- **Filelog Receiver**: Reads log files with regex parsing capabilities

### **Processors**
- **Memory Limiter**: Prevents out-of-memory conditions
- **Batch**: Batches telemetry data for efficient transmission
- **K8s Attributes**: Adds Kubernetes metadata to telemetry
- **Resource**: Adds resource-level attributes
- **Cumulative to Delta**: Converts cumulative metrics to delta
- **Transform**: Modifies telemetry data using OTTL
- **Filter**: Filters telemetry data based on conditions

### **Connectors**
- **SpanEventsToLog**: Converts span events to log records with configurable filtering

### **Exporters**
- **OTLP Exporter**: Sends data via gRPC
- **OTLP HTTP Exporter**: Sends data via HTTP

### **Extensions**
- **Memory Limiter**: Monitors memory usage

## 🚀 **Quick Start**

### **1. Build the Custom Collector**

```bash
# Make build script executable (if needed)
chmod +x build.sh

# Run the build script
./build.sh
```

This will:
- Install OCB if not already installed
- Build the custom collector with all components
- Generate binaries for multiple platforms

### **2. Run Locally**

```bash
# Run the custom collector
./dist/otelcol-custom_$(go env GOOS)_$(go env GOARCH) --config config.yaml
```

### **3. Build Docker Image**

```bash
# Build the Docker image
docker build -t otelcol-custom:latest .

# Run the container
docker run -p 4317:4317 -p 4318:4318 -p 55679:55679 otelcol-custom:latest
```

### **4. Deploy to Kubernetes**

```bash
# Create namespace (if needed)
kubectl create namespace monitoring

# Deploy the custom collector
kubectl apply -f k8s-deployment.yaml
```

## 📁 **File Structure**

```
.
├── manifest.yaml              # OCB manifest with all components
├── config.yaml               # Collector configuration
├── Dockerfile                # Multi-stage Docker build
├── k8s-deployment.yaml      # Kubernetes deployment
├── build.sh                  # Build automation script
├── OCB_BUILD_README.md      # This file
└── dist/                     # Generated binaries (after build)
```

## 🔧 **Configuration Details**

### **SpanEventsToLog Connector Configuration**

The connector is configured to monitor connection errors:

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

### **Pipeline Configuration**

The collector processes three pipelines:

1. **Traces Pipeline**: 
   - Receives traces via OTLP and filelog
   - Processes with k8s attributes, resource, transform, filter, batch
   - Connects through spaneventstolog to convert events to logs
   - Exports via OTLP

2. **Logs Pipeline**:
   - Receives logs via filelog
   - Processes with k8s attributes, resource, transform, batch
   - Exports via OTLP

3. **Metrics Pipeline**:
   - Receives metrics via OTLP
   - Processes with cumulative to delta, transform, batch
   - Exports via OTLP

## 🧪 **Testing**

### **Test the Connector**

Send a test trace with exception events:

```bash
curl -X POST http://localhost:4318/v1/traces \
  -H 'Content-Type: application/json' \
  -d '{
    "resourceSpans": [{
      "resource": {
        "attributes": [{
          "key": "service.name",
          "value": {"stringValue": "test-service"}
        }]
      },
      "scopeSpans": [{
        "spans": [{
          "traceId": "00000000000000000000000000000001",
          "spanId": "0000000000000001",
          "name": "test-span",
          "kind": 1,
          "status": {"code": 2},
          "attributes": [{
            "key": "http.status_code",
            "value": {"intValue": 503}
          }],
          "events": [{
            "name": "exception",
            "attributes": [{
              "key": "exception.type",
              "value": {"stringValue": "requests.exceptions.ConnectionError"}
            }, {
              "key": "exception.message",
              "value": {"stringValue": "Connection timeout"}
            }]
          }]
        }]
      }]
    }]
  }'
```

### **Monitor Logs**

Check the generated logs:

```bash
# If running locally
tail -f /var/log/app/*.log

# If running in Kubernetes
kubectl logs -f deployment/otelcol-custom -n monitoring
```

## 📊 **Real-World Usage**

### **Based on Real Span Data Analysis**

The configuration is based on analysis of real span data that showed:

- **Service**: `loadgenerator` (Python OpenTelemetry)
- **Endpoints**: `/api/cart`, `/api/products/*`, `/api/checkout`
- **Events**: Exception events with `ConnectionError`
- **Status Codes**: 503 (Service Unavailable)
- **Error Types**: `requests.exceptions.ConnectionError`

### **Use Cases**

1. **Error Monitoring**: Monitor all exception events in error spans
2. **Service-Specific Monitoring**: Track errors for specific services
3. **Connection Error Detection**: Focus on connection-related issues
4. **Template-Based Logging**: Generate structured logs from span events

## 🔍 **Troubleshooting**

### **Common Issues**

1. **Build Failures**: Ensure Go 1.21+ is installed
2. **OCB Not Found**: Run `go install go.opentelemetry.io/collector/cmd/builder@latest`
3. **Port Conflicts**: Change ports in config.yaml if needed
4. **Memory Issues**: Adjust memory_limiter settings

### **Health Checks**

The collector exposes health checks on port 55679:

```bash
curl http://localhost:55679/
```

### **Logs**

Check collector logs for issues:

```bash
# Local
./dist/otelcol-custom_$(go env GOOS)_$(go env GOARCH) --config config.yaml --log-level debug

# Docker
docker logs otelcol-custom

# Kubernetes
kubectl logs deployment/otelcol-custom -n monitoring
```

## 📈 **Performance Considerations**

- **Memory**: Configure memory_limiter based on your workload
- **Batch Size**: Adjust batch processor settings for throughput vs latency
- **Filtering**: Use filter processor to reduce data volume
- **Sampling**: Consider adding sampling for high-volume traces

## 🔄 **Updates and Maintenance**

### **Updating Components**

To update component versions, modify `manifest.yaml`:

```yaml
exporters:
  - gomod: go.opentelemetry.io/collector/exporter/otlpexporter v0.131.0  # Update version
```

### **Adding New Components**

Add new components to `manifest.yaml`:

```yaml
processors:
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/processor/newprocessor v0.130.0
```

Then update `config.yaml` to use the new component.

## 📚 **Additional Resources**

- [OpenTelemetry Collector Builder Documentation](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder)
- [OpenTelemetry Collector Configuration](https://opentelemetry.io/docs/collector/configuration/)
- [SpanEventsToLog Connector Documentation](./README.md) 