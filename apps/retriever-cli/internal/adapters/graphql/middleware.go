package graphql

import (
	"bytes"
	"io"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"strconv"
)

// middleware wraps a handler with one guard.
type middleware func(http.Handler) http.Handler

// maxBodyBytes caps a request body at 1 MiB. A larger body is refused before it is parsed.
const maxBodyBytes = 1 << 20

// requireLoopbackHost refuses any Host header but 127.0.0.1:<port> and localhost:<port>. This closes
// DNS rebinding, which the Origin rule does not. A host without the port is refused too.
func requireLoopbackHost(port int) middleware {
	allowed := map[string]bool{
		net.JoinHostPort("127.0.0.1", strconv.Itoa(port)): true,
		net.JoinHostPort("localhost", strconv.Itoa(port)): true,
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !allowed[r.Host] {
				writeErrors(w, http.StatusForbidden, "request host is not allowed")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// rejectBrowserOrigin refuses any request that carries an Origin header, whatever its value, before
// any upstream call. Only the presence of the header is checked, so "null" is refused too.
func rejectBrowserOrigin() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(r.Header.Values("Origin")) > 0 {
				writeErrors(w, http.StatusForbidden, "browser origins are not allowed")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requirePOST refuses every method but POST with 405 and an Allow header. OPTIONS is refused too,
// so there is no preflight answer and no CORS.
func requirePOST() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.Header().Set("Allow", http.MethodPost)
				writeErrors(w, http.StatusMethodNotAllowed, "only POST is allowed")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// requireJSON refuses a request whose media type is not application/json with 415. Parameters such as
// charset are allowed.
func requireJSON() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mediaType != "application/json" {
				writeErrors(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// limitBody refuses a body over maxBytes with 413. It reads the body once, so a later guard sees the
// same bytes, and it reads no more than maxBytes+1 bytes.
func limitBody(maxBytes int64) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				writeErrors(w, http.StatusRequestEntityTooLarge, "request body exceeds the size limit")
				return
			}
			data, err := io.ReadAll(io.LimitReader(r.Body, maxBytes+1))
			if err != nil {
				writeErrors(w, http.StatusBadRequest, "could not read request body")
				return
			}
			if int64(len(data)) > maxBytes {
				writeErrors(w, http.StatusRequestEntityTooLarge, "request body exceeds the size limit")
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(data))
			r.ContentLength = int64(len(data))
			next.ServeHTTP(w, r)
		})
	}
}

// readBody returns the request body and leaves an equal body in place for the next handler.
func readBody(r *http.Request) ([]byte, error) {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(data))
	return data, nil
}

// logRefusals records every response with a 4xx status: the method and the status, never the request
// content. It wraps the whole chain, so it sees each refusal.
func logRefusals(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			if rec.status >= http.StatusBadRequest {
				logger.Info("graphql request refused", "status", rec.status, "method", r.Method)
			}
		})
	}
}

// statusRecorder keeps the first status a handler writes.
type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (s *statusRecorder) WriteHeader(code int) {
	if !s.wrote {
		s.status = code
		s.wrote = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	if !s.wrote {
		s.status = http.StatusOK
		s.wrote = true
	}
	return s.ResponseWriter.Write(b)
}
