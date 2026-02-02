// Copyright 2026 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
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

package log_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/http/log"
)

func TestLoggingMiddleware_Integration(t *testing.T) {
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("OK"))
	})

	middleware := log.LoggingMiddleware(mockHandler)
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	req.Header.Set("User-Agent", "Test-Agent/1.0")
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusCreated)
	}
	if body := rr.Body.String(); body != "OK" {
		t.Errorf("handler returned wrong body: got %v want %v",
			body, "OK")
	}
}

func TestLoggingMiddleware_Integration_DefaultValues(t *testing.T) {
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No status code set. Must be 200 OK by default.
		w.Write([]byte("OK"))
	})

	middleware := log.LoggingMiddleware(mockHandler)
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	req.Header.Del("User-Agent")
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	if body := rr.Body.String(); body != "OK" {
		t.Errorf("handler returned wrong body: got %v want %v",
			body, "OK")
	}
}
