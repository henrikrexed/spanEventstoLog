// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package spaneventstologconnector

import (
	"bytes"
	"errors"
	"fmt"
	"text/template"
	"text/template/parse"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspanevent"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.uber.org/zap"
)

// Config defines the configuration for the SpanEventsToLog connector
type Config struct {
	// SpanConditions defines OTTL conditions for filtering spans
	// If empty, all spans will be processed
	SpanConditions []string `mapstructure:"span_conditions"`

	// EventConditions defines OTTL conditions for filtering individual span events
	// If empty, all events will be processed
	EventConditions []string `mapstructure:"event_conditions"`

	// IncludeSpanAttributes determines if span attributes should be included in the log record
	IncludeSpanAttributes bool `mapstructure:"include_span_attributes"`

	// IncludeEventAttributes determines if event attributes should be included in the log record
	IncludeEventAttributes bool `mapstructure:"include_event_attributes"`

	// LogLevel sets the severity level for generated log records
	LogLevel string `mapstructure:"log_level"`

	// LogBodyTemplate defines the template for the log body
	// Available placeholders: {{.EventName}}, {{.SpanName}}, {{.EventAttributes}}, {{.SpanAttributes}}
	LogBodyTemplate string `mapstructure:"log_body_template"`

	// prevent unkeyed literal initialization
	_ struct{}
}

// Validate implements component.Config
func (cfg *Config) Validate() error {
	if len(cfg.SpanConditions) == 0 && len(cfg.EventConditions) == 0 {
		return errors.New("at least one span condition or event condition must be specified")
	}

	if cfg.LogLevel != "" {
		validLevels := []string{"Trace", "Debug", "Info", "Warn", "Error", "Fatal"}
		found := false
		for _, level := range validLevels {
			if cfg.LogLevel == level {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("invalid log_level: %s, must be one of %v", cfg.LogLevel, validLevels)
		}
	}

	logger := zap.NewNop()
	settings := component.TelemetrySettings{Logger: logger}

	// Validate OTTL span conditions
	if len(cfg.SpanConditions) > 0 {
		parser, err := ottlspan.NewParser(ottlfuncs.StandardFuncs[ottlspan.TransformContext](), settings)
		if err != nil {
			return fmt.Errorf("failed to create OTTL span parser: %w", err)
		}
		for _, cond := range cfg.SpanConditions {
			if _, err := parser.ParseCondition(cond); err != nil {
				return fmt.Errorf("invalid span_condition OTTL: %q: %w", cond, err)
			}
		}
	}

	// Validate OTTL event conditions
	if len(cfg.EventConditions) > 0 {
		parser, err := ottlspanevent.NewParser(ottlfuncs.StandardFuncs[ottlspanevent.TransformContext](), settings)
		if err != nil {
			return fmt.Errorf("failed to create OTTL event parser: %w", err)
		}
		for _, cond := range cfg.EventConditions {
			if _, err := parser.ParseCondition(cond); err != nil {
				return fmt.Errorf("invalid event_condition OTTL: %q: %w", cond, err)
			}
		}
	}

	// Validate log body template
	if cfg.LogBodyTemplate != "" {
		tmpl, err := template.New("logBody").Parse(cfg.LogBodyTemplate)
		if err != nil {
			return fmt.Errorf("invalid log_body_template: %w", err)
		}
		// Try to execute the template with a dummy struct to catch invalid fields
		dummy := struct {
			EventName       string
			SpanName        string
			EventAttributes map[string]string
			SpanAttributes  map[string]string
		}{
			EventName:       "event",
			SpanName:        "span",
			EventAttributes: map[string]string{"key": "value"},
			SpanAttributes:  map[string]string{"key": "value"},
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, dummy); err != nil {
			return fmt.Errorf("invalid log_body_template (execution): %w", err)
		}

		// Strict validation: walk the template AST and ensure only allowed fields are referenced
		allowed := map[string]struct{}{
			"EventName":       {},
			"SpanName":        {},
			"EventAttributes": {},
			"SpanAttributes":  {},
		}
		for _, tree := range tmpl.Templates() {
			if tree == nil || tree.Tree == nil {
				continue
			}
			var walkNodes func(n parse.Node) error
			walkNodes = func(n parse.Node) error {
				if n == nil {
					return nil
				}
				switch node := n.(type) {
				case *parse.FieldNode:
					if len(node.Ident) > 0 {
						field := node.Ident[0]
						fmt.Printf("DEBUG: Found field node: .%s\n", field)
						if _, ok := allowed[field]; !ok {
							return fmt.Errorf("invalid field in log_body_template: .%s is not allowed", field)
						}
					}
				case *parse.VariableNode:
					if len(node.Ident) > 0 {
						field := node.Ident[0]
						fmt.Printf("DEBUG: Found variable node: .%s\n", field)
						if _, ok := allowed[field]; !ok {
							return fmt.Errorf("invalid field in log_body_template: .%s is not allowed", field)
						}
					}
				case *parse.ListNode:
					for _, child := range node.Nodes {
						if err := walkNodes(child); err != nil {
							return err
						}
					}
				case *parse.ActionNode:
					if err := walkNodes(node.Pipe); err != nil {
						return err
					}
				case *parse.PipeNode:
					for _, cmd := range node.Cmds {
						if err := walkNodes(cmd); err != nil {
							return err
						}
					}
				case *parse.CommandNode:
					for _, arg := range node.Args {
						if err := walkNodes(arg); err != nil {
							return err
						}
					}
				case *parse.IfNode:
					if err := walkNodes(node.Pipe); err != nil {
						return err
					}
					if err := walkNodes(node.List); err != nil {
						return err
					}
					if err := walkNodes(node.ElseList); err != nil {
						return err
					}
				case *parse.RangeNode:
					if err := walkNodes(node.Pipe); err != nil {
						return err
					}
					if err := walkNodes(node.List); err != nil {
						return err
					}
					if err := walkNodes(node.ElseList); err != nil {
						return err
					}
				case *parse.WithNode:
					if err := walkNodes(node.Pipe); err != nil {
						return err
					}
					if err := walkNodes(node.List); err != nil {
						return err
					}
					if err := walkNodes(node.ElseList); err != nil {
						return err
					}
				}
				return nil
			}
			if err := walkNodes(tree.Tree.Root); err != nil {
				return err
			}
		}
	}

	return nil
}

var _ confmap.Unmarshaler = (*Config)(nil)

// Unmarshal with custom logic to set default values
func (c *Config) Unmarshal(componentParser *confmap.Conf) error {
	if componentParser == nil {
		return nil
	}
	if err := componentParser.Unmarshal(c, confmap.WithIgnoreUnused()); err != nil {
		return err
	}

	// Set defaults if not specified
	if !componentParser.IsSet("include_span_attributes") {
		c.IncludeSpanAttributes = true
	}
	if !componentParser.IsSet("include_event_attributes") {
		c.IncludeEventAttributes = true
	}
	if !componentParser.IsSet("log_level") {
		c.LogLevel = "Info"
	}
	if !componentParser.IsSet("log_body_template") {
		c.LogBodyTemplate = "Span Event: {{.EventName}}"
	}

	return nil
}
