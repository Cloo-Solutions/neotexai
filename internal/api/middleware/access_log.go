package middleware

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

type accessLogEntry struct {
	Timestamp  string `json:"ts"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	Status     int    `json:"status"`
	Bytes      int    `json:"bytes"`
	DurationMS int64  `json:"duration_ms"`
	RequestID  string `json:"request_id,omitempty"`
	OrgID      string `json:"org_id,omitempty"`
	RemoteAddr string `json:"remote_addr,omitempty"`
	UserAgent  string `json:"user_agent,omitempty"`
}

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// AccessLog emits structured JSON logs for HTTP requests.
func AccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}

		orgID := GetOrgID(r.Context())
		if orgID == "" {
			orgID = r.Header.Get("X-Org-ID")
		}

		entry := accessLogEntry{
			Timestamp:  start.UTC().Format(time.RFC3339Nano),
			Method:     r.Method,
			Path:       r.URL.Path,
			Status:     status,
			Bytes:      rec.bytes,
			DurationMS: time.Since(start).Milliseconds(),
			RequestID:  GetRequestID(r.Context()),
			OrgID:      orgID,
			RemoteAddr: clientIP(r),
			UserAgent:  r.UserAgent(),
		}

		payload, err := json.Marshal(entry)
		if err != nil {
			log.Printf("access_log_marshal_error: %v", err)
			return
		}
		log.Println(string(payload))
	})
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
