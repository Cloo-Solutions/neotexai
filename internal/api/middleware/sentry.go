package middleware

import (
	"fmt"
	"net/http"

	"github.com/getsentry/sentry-go"
)

// SentryMiddleware creates a transaction for each HTTP request and captures errors/panics.
// It adds request context (org_id, request_id, method, path, user_agent) to events.
// Gracefully degrades if Sentry is not initialized.
func SentryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clone the hub to isolate this request's context
		hub := sentry.GetHubFromContext(r.Context())
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
		}

		// Create a transaction for this HTTP request
		transactionName := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		options := []sentry.SpanOption{
			sentry.WithOpName("http.server"),
			sentry.WithTransactionSource(sentry.SourceURL),
		}

		// Continue from incoming trace headers if present
		if sentryTrace := r.Header.Get("sentry-trace"); sentryTrace != "" {
			options = append(options, sentry.ContinueFromHeaders(sentryTrace, r.Header.Get("baggage")))
		}

		transaction := sentry.StartTransaction(r.Context(), transactionName, options...)
		defer transaction.Finish()

		// Set hub on context with transaction
		ctx := transaction.Context()
		ctx = sentry.SetHubOnContext(ctx, hub)
		r = r.WithContext(ctx)

		// Add request context to Sentry event
		hub.Scope().SetContext("request", map[string]interface{}{
			"method":      r.Method,
			"path":        r.URL.Path,
			"query":       r.URL.RawQuery,
			"remote_addr": r.RemoteAddr,
		})

		// Add request_id tag if available
		if requestID := GetRequestID(r.Context()); requestID != "" {
			hub.Scope().SetTag("request_id", requestID)
			transaction.SetTag("request_id", requestID)
		}

		// Add org_id tag if available (will be set after auth middleware runs)
		// We'll update it in a deferred function after the request completes

		// Add user_agent tag
		if userAgent := r.UserAgent(); userAgent != "" {
			hub.Scope().SetTag("user_agent", userAgent)
		}

		// Capture panics and send to Sentry
		defer func() {
			if err := recover(); err != nil {
				transaction.Status = sentry.SpanStatusInternalError
				hub.RecoverWithContext(r.Context(), err)
				// Re-panic to allow other recovery middleware to handle it
				panic(err)
			}
		}()

		// Wrap response writer to capture status code
		rec := &sentryResponseRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		// Set transaction status based on HTTP status code
		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}

		transaction.Status = httpStatusToSpanStatus(status)
		transaction.SetData("http.response.status_code", status)

		// Update org_id tag after auth middleware has run
		if orgID := GetOrgID(r.Context()); orgID != "" {
			hub.Scope().SetTag("org_id", orgID)
			transaction.SetTag("org_id", orgID)
		}

		// Capture 5xx errors as messages (actual exceptions are captured elsewhere)
		if status >= 500 {
			hub.CaptureMessage(fmt.Sprintf("HTTP %d: %s", status, http.StatusText(status)))
		}
	})
}

// httpStatusToSpanStatus converts HTTP status code to Sentry span status.
func httpStatusToSpanStatus(status int) sentry.SpanStatus {
	switch {
	case status >= 200 && status < 300:
		return sentry.SpanStatusOK
	case status == 400:
		return sentry.SpanStatusInvalidArgument
	case status == 401:
		return sentry.SpanStatusUnauthenticated
	case status == 403:
		return sentry.SpanStatusPermissionDenied
	case status == 404:
		return sentry.SpanStatusNotFound
	case status == 409:
		return sentry.SpanStatusAlreadyExists
	case status == 429:
		return sentry.SpanStatusResourceExhausted
	case status == 499:
		return sentry.SpanStatusCanceled
	case status >= 400 && status < 500:
		return sentry.SpanStatusInvalidArgument
	case status == 500:
		return sentry.SpanStatusInternalError
	case status == 501:
		return sentry.SpanStatusUnimplemented
	case status == 503:
		return sentry.SpanStatusUnavailable
	case status == 504:
		return sentry.SpanStatusDeadlineExceeded
	case status >= 500:
		return sentry.SpanStatusInternalError
	default:
		return sentry.SpanStatusUnknown
	}
}

// sentryResponseRecorder wraps http.ResponseWriter to capture status code
type sentryResponseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *sentryResponseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *sentryResponseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}
