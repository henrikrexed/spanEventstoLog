package spaneventstologconnector

import (
	"context"

	"github.com/open-telemetry/opentelemetry-collector-contrib/connector/spaneventstologconnector/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
)

// NewFactory returns a connector.Factory for the SpanEventConnector
func NewFactory() connector.Factory {
	return connector.NewFactory(
		metadata.Type,
		createDefaultConfig,
		connector.WithTracesToLogs(createTracesToLogs, metadata.TracesToLogsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		SpanConditions:         []string{},
		EventConditions:        []string{},
		IncludeSpanAttributes:  true,
		IncludeEventAttributes: true,
		LogLevel:               "Info",
		LogBodyTemplate:        "Span Event: {{.EventName}}",
	}
}

func createTracesToLogs(
	_ context.Context,
	set connector.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (connector.Traces, error) {
	return NewSpanEventConnector(set, cfg, nextConsumer)
}
