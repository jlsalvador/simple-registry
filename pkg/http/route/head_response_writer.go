package route

import "net/http"

// HeadResponseWriter is a custom http.ResponseWriter that discards the
// response body for a HEAD request.
type HeadResponseWriter struct {
	http.ResponseWriter
}

// Write discards the response body for a HEAD request.
func (h *HeadResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}
