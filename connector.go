package spaneventstologconnector

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspanevent"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/otel/metric"
	"go.uber.org/zap"
)

// SpanEventConnector is the main connector implementation
// Implements connector.Traces

type SpanEventConnector struct {
	config       *Config
	logger       *zap.Logger
	consumer     consumer.Logs
	bodyTemplate *template.Template
	spanOttl     []*ottl.Statement[ottlspan.TransformContext]
	eventOttl    []*ottl.Statement[ottlspanevent.TransformContext]

	// Telemetry counters
	spansHandledCounter metric.Int64Counter
	logsProducedCounter metric.Int64Counter
}

// NewSpanEventConnector creates a new SpanEventConnector instance
func NewSpanEventConnector(
	set connector.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (connector.Traces, error) {
	config := cfg.(*Config)

	// Parse log body template
	bodyTemplate, err := template.New("logBody").Parse(config.LogBodyTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse log body template: %w", err)
	}

	// Parse OTTL span conditions
	spanParser, err := ottlspan.NewParser(ottlfuncs.StandardFuncs[ottlspan.TransformContext](), set.TelemetrySettings)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTTL span parser: %w", err)
	}
	var spanOttl []*ottl.Statement[ottlspan.TransformContext]
	for _, cond := range config.SpanConditions {
		stmt, err := spanParser.ParseStatement(cond)
		if err != nil {
			return nil, fmt.Errorf("invalid span_condition OTTL: %q: %w", cond, err)
		}
		spanOttl = append(spanOttl, stmt)
	}

	// Parse OTTL event conditions
	eventParser, err := ottlspanevent.NewParser(ottlfuncs.StandardFuncs[ottlspanevent.TransformContext](), set.TelemetrySettings)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTTL event parser: %w", err)
	}
	var eventOttl []*ottl.Statement[ottlspanevent.TransformContext]
	for _, cond := range config.EventConditions {
		stmt, err := eventParser.ParseStatement(cond)
		if err != nil {
			return nil, fmt.Errorf("invalid event_condition OTTL: %q: %w", cond, err)
		}
		eventOttl = append(eventOttl, stmt)
	}

	// Initialize metrics instruments when a MeterProvider is available
	var spansHandled metric.Int64Counter
	var logsProduced metric.Int64Counter
	if set.MeterProvider != nil {
		meter := set.MeterProvider.Meter("github.com/henrikrexed/spanEventstoLog")
		// Best-effort instrument creation; ignore errors to avoid breaking data path
		if c, err := meter.Int64Counter(
			"spaneventstolog.spans_handled",
			metric.WithDescription("Number of spans that passed connector span conditions"),
			metric.WithUnit("{spans}"),
		); err == nil {
			spansHandled = c
		}
		if c, err := meter.Int64Counter(
			"spaneventstolog.logs_produced",
			metric.WithDescription("Number of logs produced from span events"),
			metric.WithUnit("{logs}"),
		); err == nil {
			logsProduced = c
		}
	}

	return &SpanEventConnector{
		config:              config,
		logger:              set.Logger,
		consumer:            nextConsumer,
		bodyTemplate:        bodyTemplate,
		spanOttl:            spanOttl,
		eventOttl:           eventOttl,
		spansHandledCounter: spansHandled,
		logsProducedCounter: logsProduced,
	}, nil
}

func (c *SpanEventConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

func (c *SpanEventConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	logs := plog.NewLogs()

	var numSpansHandled int64
	var numLogsProduced int64

	resourceSpansSlice := td.ResourceSpans()
	for i := 0; i < resourceSpansSlice.Len(); i++ {
		resourceSpans := resourceSpansSlice.At(i)
		resource := resourceSpans.Resource()
		scopeSpansSlice := resourceSpans.ScopeSpans()
		for j := 0; j < scopeSpansSlice.Len(); j++ {
			scopeSpans := scopeSpansSlice.At(j)
			scope := scopeSpans.Scope()
			spansSlice := scopeSpans.Spans()
			for k := 0; k < spansSlice.Len(); k++ {
				span := spansSlice.At(k)
				if !c.matchesSpanConditionsWithContext(span, resource, scope, scopeSpans, resourceSpans) {
					continue
				}
				numSpansHandled++
				for l := 0; l < span.Events().Len(); l++ {
					event := span.Events().At(l)
					if c.matchesEventConditionsWithContext(event, span, resource, scope, scopeSpans, resourceSpans) {
						c.createLogRecord(event, span, resource, scope, logs)
						numLogsProduced++
					}
				}
			}
		}
	}

	if logs.ResourceLogs().Len() > 0 {
		return c.consumer.ConsumeLogs(ctx, logs)
	}

	// Record metrics outside of the tight loops
	if c.spansHandledCounter != nil && numSpansHandled > 0 {
		c.spansHandledCounter.Add(ctx, numSpansHandled)
	}
	if c.logsProducedCounter != nil && numLogsProduced > 0 {
		c.logsProducedCounter.Add(ctx, numLogsProduced)
	}

	return nil
}

func (c *SpanEventConnector) matchesSpanConditionsWithContext(span ptrace.Span, resource pcommon.Resource, scope pcommon.InstrumentationScope, scopeSpans ptrace.ScopeSpans, resourceSpans ptrace.ResourceSpans) bool {
	if len(c.spanOttl) == 0 {
		return true
	}
	ctx := ottlspan.NewTransformContext(span, scope, resource, scopeSpans, resourceSpans)
	for _, stmt := range c.spanOttl {
		result, _, err := stmt.Execute(context.Background(), ctx)
		if err != nil {
			c.logger.Error("Failed to execute span OTTL condition", zap.Error(err))
			continue
		}
		if ok, _ := result.(bool); ok {
			return true
		}
	}
	return false
}

func (c *SpanEventConnector) matchesEventConditionsWithContext(event ptrace.SpanEvent, span ptrace.Span, resource pcommon.Resource, scope pcommon.InstrumentationScope, scopeSpans ptrace.ScopeSpans, resourceSpans ptrace.ResourceSpans) bool {
	if len(c.eventOttl) == 0 {
		return true
	}
	ctx := ottlspanevent.NewTransformContext(event, span, scope, resource, scopeSpans, resourceSpans)
	for _, stmt := range c.eventOttl {
		result, _, err := stmt.Execute(context.Background(), ctx)
		if err != nil {
			c.logger.Error("Failed to execute event OTTL condition", zap.Error(err))
			continue
		}
		if ok, _ := result.(bool); ok {
			return true
		}
	}
	return false
}

func (c *SpanEventConnector) createLogRecord(
	event ptrace.SpanEvent,
	span ptrace.Span,
	resource pcommon.Resource,
	scope pcommon.InstrumentationScope,
	logs plog.Logs,
) {
	rl := logs.ResourceLogs().AppendEmpty()
	resource.CopyTo(rl.Resource())

	sl := rl.ScopeLogs().AppendEmpty()
	scope.CopyTo(sl.Scope())

	logRecord := sl.LogRecords().AppendEmpty()

	// Set basic log record fields
	logRecord.SetTimestamp(event.Timestamp())
	logRecord.SetSeverityText(c.config.LogLevel)
	logRecord.SetSeverityNumber(c.getSeverityNumber(c.config.LogLevel))

	// Set the log body using template
	body := c.generateLogBody(event, span)
	logRecord.Body().SetStr(body)

	// Add trace context
	logRecord.SetTraceID(span.TraceID())
	logRecord.SetSpanID(span.SpanID())
	logRecord.SetFlags(0) // Simplified: don't use trace state

	// Add basic attributes
	attrs := logRecord.Attributes()
	attrs.PutStr("span.name", span.Name())
	attrs.PutStr("span.kind", span.Kind().String())
	attrs.PutStr("event.name", event.Name())

	// Include span attributes if configured
	if c.config.IncludeSpanAttributes {
		span.Attributes().Range(func(k string, v pcommon.Value) bool {
			v.CopyTo(attrs.PutEmpty("span." + k))
			return true
		})
	}

	// Include event attributes if configured
	if c.config.IncludeEventAttributes {
		event.Attributes().Range(func(k string, v pcommon.Value) bool {
			v.CopyTo(attrs.PutEmpty("event." + k))
			return true
		})
	}
}

func (c *SpanEventConnector) generateLogBody(event ptrace.SpanEvent, span ptrace.Span) string {
	if c.bodyTemplate == nil {
		return fmt.Sprintf("Span Event: %s", event.Name())
	}

	// Prepare template data
	data := struct {
		EventName       string
		SpanName        string
		EventAttributes map[string]string
		SpanAttributes  map[string]string
	}{
		EventName: event.Name(),
		SpanName:  span.Name(),
		EventAttributes: func() map[string]string {
			attrs := make(map[string]string)
			event.Attributes().Range(func(k string, v pcommon.Value) bool {
				attrs[k] = v.AsString()
				return true
			})
			return attrs
		}(),
		SpanAttributes: func() map[string]string {
			attrs := make(map[string]string)
			span.Attributes().Range(func(k string, v pcommon.Value) bool {
				attrs[k] = v.AsString()
				return true
			})
			return attrs
		}(),
	}

	var buf strings.Builder
	if err := c.bodyTemplate.Execute(&buf, data); err != nil {
		c.logger.Error("Failed to execute log body template", zap.Error(err))
		return fmt.Sprintf("Span Event: %s", event.Name())
	}

	return buf.String()
}

func (c *SpanEventConnector) getSeverityNumber(level string) plog.SeverityNumber {
	switch level {
	case "Trace":
		return plog.SeverityNumberTrace
	case "Debug":
		return plog.SeverityNumberDebug
	case "Info":
		return plog.SeverityNumberInfo
	case "Warn":
		return plog.SeverityNumberWarn
	case "Error":
		return plog.SeverityNumberError
	case "Fatal":
		return plog.SeverityNumberFatal
	default:
		return plog.SeverityNumberInfo
	}
}

func (c *SpanEventConnector) Shutdown(context.Context) error {
	return nil
}

func (c *SpanEventConnector) Start(context.Context, component.Host) error {
	return nil
}
