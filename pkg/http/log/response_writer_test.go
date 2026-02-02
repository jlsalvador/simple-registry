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

func TestLoggingResponseWriter_InternalLogic(t *testing.T) {
	t.Run("explicit status code", func(t *testing.T) {
		rr := httptest.NewRecorder()

		lrw := &log.LoggingResponseWriter{ResponseWriter: rr}

		lrw.WriteHeader(http.StatusTeapot)
		lrw.Write([]byte("Short"))

		if lrw.Status != http.StatusTeapot {
			t.Errorf("expected status %d, got %d", http.StatusTeapot, lrw.Status)
		}
		if lrw.Bytes != 5 {
			t.Errorf("expected 5 bytes, got %d", lrw.Bytes)
		}
	})

	t.Run("implicit status OK", func(t *testing.T) {
		rr := httptest.NewRecorder()
		lrw := &log.LoggingResponseWriter{ResponseWriter: rr}

		// We do not call WriteHeader; it should default to 200 upon writing.
		lrw.Write([]byte("Hello"))

		if lrw.Status != http.StatusOK {
			t.Errorf("expected status %d (implicit), got %d", http.StatusOK, lrw.Status)
		}
		if lrw.Bytes != 5 {
			t.Errorf("expected 5 bytes, got %d", lrw.Bytes)
		}
	})
}
