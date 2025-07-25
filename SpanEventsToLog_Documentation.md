# SpanEventsToLog Connector: Full Documentation

## Overview

The **SpanEventsToLog** connector transforms OpenTelemetry span events (such as exceptions or custom events) into log records. This enables error/event monitoring, log enrichment with trace context, and flexible filtering using OTTL expressions.

---

## Configuration Options

The connector is configured under the `connectors` section in your OpenTelemetry Collector configuration. The available options are:

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

## OTTL Conditions

- **span_conditions**: Filter which spans are considered. Example: `attributes["http.status_code"] == 503`.
- **event_conditions**: Filter which events within a span are considered. Example: `name == "exception"`.
- See [OTTL documentation](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/README.md) for syntax and available functions.

---

## Log Body Template

- Uses Go's `text/template` syntax.
- Allowed placeholders:
  - `{{.EventName}}`: Name of the span event
  - `{{.SpanName}}`: Name of the span
  - `{{.EventAttributes}}`: Map of event attributes (access as `{{.EventAttributes.key}}`)
  - `{{.SpanAttributes}}`: Map of span attributes (access as `{{.SpanAttributes.key}}`)
- Example:
  ```yaml
  log_body_template: "Connection Error in {{.SpanName}}: {{.EventName}} - {{.EventAttributes.exception.message}}"
  ```

---

## Example Configurations

### Basic Example
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

### Exception-Only Example
```yaml
connectors:
  spaneventstolog:
    event_conditions:
      - "name == \"exception\""
    log_level: "Error"
    log_body_template: "Exception: {{.EventAttributes.exception.type}} in {{.SpanName}}"
```

### Custom Event Attribute Example
```yaml
connectors:
  spaneventstolog:
    event_conditions:
      - "attributes[\"custom.event\"] == true"
    include_event_attributes: true
    log_level: "Info"
    log_body_template: "Custom event {{.EventName}}: {{.EventAttributes.description}}"
```

### Minimal Example (not recommended)
```yaml
connectors:
  spaneventstolog:
    span_conditions:
      - "attributes[\"http.status_code\"] == 500"
```

---

## Usage in a Pipeline

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
      processors: []
      exporters: [spaneventstolog]
    logs:
      receivers: [spaneventstolog]
      exporters: [otlp]
```

---

## Best Practices & Tips
- Use specific OTTL conditions to avoid excessive log generation.
- Test your `log_body_template` with sample data to ensure correct output.
- Use `include_span_attributes` and `include_event_attributes` judiciously to avoid overly verbose logs.
- Validate your configuration with `otelcol-custom --config <your-config.yaml>` before deploying.

---

## Troubleshooting
- **Startup fails with OTTL error**: Check your `span_conditions` and `event_conditions` for syntax errors.
- **No logs generated**: Ensure your conditions match incoming spans/events and that the connector is included in the pipeline.
- **Invalid log level**: Use only the allowed values: Trace, Debug, Info, Warn, Error, Fatal.
- **Template errors**: Only use allowed placeholders in `log_body_template`.

---

## References
- Example config: [`collector/config.yaml`](collector/config.yaml)
- Project README: [`README.md`](README.md)
- OTTL documentation: [OpenTelemetry OTTL](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/README.md) 