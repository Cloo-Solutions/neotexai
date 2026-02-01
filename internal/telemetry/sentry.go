// Package telemetry provides Sentry-based distributed tracing utilities.
package telemetry

import (
	"context"
	"log"
	"time"

	"github.com/getsentry/sentry-go"
)

const (
	serviceName = "neotexai"
)

// Config holds the configuration for Sentry initialization.
type Config struct {
	DSN              string
	Environment      string
	TracesSampleRate float64
	Debug            bool
}

// Init initializes Sentry with tracing enabled.
// Returns a shutdown function to flush pending events.
// If DSN is empty, returns a no-op shutdown function.
func Init(cfg Config) (func(), error) {
	if cfg.DSN == "" {
		return func() {}, nil
	}

	if cfg.Environment == "" {
		cfg.Environment = "development"
	}

	if cfg.TracesSampleRate == 0 {
		cfg.TracesSampleRate = 1.0 // Default to sampling all traces
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		EnableTracing:    true,
		TracesSampleRate: cfg.TracesSampleRate,
		Debug:            cfg.Debug,
		ServerName:       serviceName,
		// Propagate traces to downstream services
		TracesSampler: sentry.TracesSampler(func(ctx sentry.SamplingContext) float64 {
			// Skip health check endpoints
			if ctx.Span.Name == "GET /health" || ctx.Span.Op == "http.server GET /health" {
				return 0.0
			}
			// If this is a child span, follow parent's sampling decision
			var emptySpanID sentry.SpanID
			if ctx.Span.ParentSpanID != emptySpanID {
				if ctx.Span.Sampled.Bool() {
					return 1.0
				}
				return 0.0
			}
			return cfg.TracesSampleRate
		}),
	})
	if err != nil {
		log.Printf("sentry: failed to initialize (continuing without tracing): %v", err)
		return func() {}, nil
	}

	shutdown := func() {
		sentry.Flush(5 * time.Second)
	}

	log.Printf("sentry: tracing initialized (environment: %s, sample_rate: %.2f)", cfg.Environment, cfg.TracesSampleRate)
	return shutdown, nil
}

// SpanAttributes contains common attributes for service spans.
type SpanAttributes struct {
	OrgID       string
	ProjectID   string
	KnowledgeID string
	Operation   string
}

// Span wraps sentry.Span to provide a consistent interface.
type Span struct {
	inner *sentry.Span
}

// End finishes the span.
func (s *Span) End() {
	if s.inner != nil {
		s.inner.Finish()
	}
}

// SetStatus sets the span status.
func (s *Span) SetStatus(status sentry.SpanStatus) {
	if s.inner != nil {
		s.inner.Status = status
	}
}

// SetError marks the span as errored and captures the exception.
func (s *Span) SetError(err error) {
	if s.inner != nil {
		s.inner.Status = sentry.SpanStatusInternalError
		if hub := sentry.GetHubFromContext(s.inner.Context()); hub != nil {
			hub.CaptureException(err)
		}
	}
}

// Context returns the span's context.
func (s *Span) Context() context.Context {
	if s.inner != nil {
		return s.inner.Context()
	}
	return context.Background()
}

// setAttributes sets common attributes on a span.
func setAttributes(span *sentry.Span, attrs SpanAttributes) {
	if span == nil {
		return
	}

	if attrs.OrgID != "" {
		span.SetTag("org_id", attrs.OrgID)
	}
	if attrs.ProjectID != "" {
		span.SetTag("project_id", attrs.ProjectID)
	}
	if attrs.KnowledgeID != "" {
		span.SetTag("knowledge_id", attrs.KnowledgeID)
	}
	if attrs.Operation != "" {
		span.SetData("operation", attrs.Operation)
	}
}

// StartSpan creates a new span with the given name.
// Returns the context with the span and a Span wrapper.
// If there's an existing transaction in context, creates a child span.
// Otherwise creates a new transaction.
func StartSpan(ctx context.Context, name string, attrs SpanAttributes) (context.Context, *Span) {
	// Check if there's already a span/transaction in context
	parentSpan := sentry.SpanFromContext(ctx)

	var span *sentry.Span
	if parentSpan != nil {
		// Create child span
		span = parentSpan.StartChild(name)
	} else {
		// Create new transaction
		span = sentry.StartSpan(ctx, name, sentry.WithTransactionName(name))
	}

	setAttributes(span, attrs)

	return span.Context(), &Span{inner: span}
}

// StartTransaction creates a new transaction (root span) with the given name.
// Use this for top-level operations like HTTP requests.
func StartTransaction(ctx context.Context, name string, op string) (context.Context, *Span) {
	options := []sentry.SpanOption{
		sentry.WithTransactionName(name),
	}
	if op != "" {
		options = append(options, sentry.WithOpName(op))
	}

	span := sentry.StartSpan(ctx, op, options...)
	return span.Context(), &Span{inner: span}
}

// CaptureError captures an error to Sentry with the current context.
func CaptureError(ctx context.Context, err error) {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.CaptureException(err)
	} else {
		sentry.CaptureException(err)
	}
}

// CaptureMessage captures a message to Sentry with the current context.
func CaptureMessage(ctx context.Context, message string) {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.CaptureMessage(message)
	} else {
		sentry.CaptureMessage(message)
	}
}

// AddBreadcrumb adds a breadcrumb to the current scope.
func AddBreadcrumb(ctx context.Context, category, message string) {
	breadcrumb := &sentry.Breadcrumb{
		Type:      "default",
		Category:  category,
		Message:   message,
		Level:     sentry.LevelInfo,
		Timestamp: time.Now(),
	}

	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		hub.AddBreadcrumb(breadcrumb, nil)
	} else {
		sentry.AddBreadcrumb(breadcrumb)
	}
}
