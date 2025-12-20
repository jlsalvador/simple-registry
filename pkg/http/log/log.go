// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"net/http"
	"time"

	"github.com/jlsalvador/simple-registry/internal/version"
	pkgLog "github.com/jlsalvador/simple-registry/pkg/log"
)

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lrw := &loggingResponseWriter{
			ResponseWriter: w,
		}

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)

		remoteAddr := getClientIP(r)

		userAgent := r.UserAgent()
		if userAgent == "" {
			userAgent = "-"
		}

		status := lrw.status
		if status == 0 {
			status = http.StatusOK
		}

		pkgLog.Info(
			"service.name", version.AppName,
			"service.version", version.AppVersion,
			"event.dataset", "http.access",
			"client.ip", remoteAddr,
			"http.request.method", r.Method,
			"url.original", r.URL.String(),
			"http.version", r.Proto,
			"http.response.status_code", status,
			"http.response.body.bytes", lrw.bytes,
			"http.request.body.bytes", r.ContentLength,
			"event.duration", duration.Nanoseconds(),
			"user_agent.original", userAgent,
		).Print()
	})
}
