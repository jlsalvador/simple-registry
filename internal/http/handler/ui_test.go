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

package handler_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeMux_RedirectToUI(t *testing.T) {
	h := testSetupTestServeMux(t)

	// Testing the RedirectToUI through the "/" route.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected status 303 (SeeOther), got %d", rr.Code)
	}

	if loc := rr.Header().Get("Location"); loc != "/ui" {
		t.Errorf("expected location /ui, got %s", loc)
	}
}

func TestServeMux_UI_Logic(t *testing.T) {
	h := testSetupTestServeMux(t)

	tests := []struct {
		name           string
		urlPath        string
		expectedStatus int
		contentType    string
	}{
		{
			name:           "Root UI path must returns SPA",
			urlPath:        "/ui",
			expectedStatus: http.StatusOK,
			contentType:    "text/html",
		},
		{
			name:           "Root UI path with slash must returns SPA",
			urlPath:        "/ui/",
			expectedStatus: http.StatusOK,
			contentType:    "text/html",
		},
		{
			name:           "Direct index.html access must redirect to SPA",
			urlPath:        "/ui/index.html",
			expectedStatus: http.StatusMovedPermanently,
			contentType:    "",
		},
		{
			name:           "Direct favicon.ico access",
			urlPath:        "/ui/favicon.ico",
			expectedStatus: http.StatusOK,
			contentType:    "image/vnd.microsoft.icon",
		},
		{
			name:           "SPA Fallback for non-existent files",
			urlPath:        "/ui/dashboard/settings",
			expectedStatus: http.StatusOK,
			contentType:    "text/html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.urlPath, nil)
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d. Body: %s", tt.urlPath, tt.expectedStatus, rr.Code, rr.Body.String())
			}

			ct := rr.Header().Get("Content-Type")
			if !strings.Contains(ct, tt.contentType) {
				t.Errorf("%s: expected content-type %s, got %s", tt.urlPath, tt.contentType, ct)
			}
		})
	}
}

func TestServeMux_AllowedMethods(t *testing.T) {
	h := testSetupTestServeMux(t)

	t.Run("Method Not Allowed for UI", func(t *testing.T) {
		// UI only supports GET in your route registration.
		req := httptest.NewRequest(http.MethodPost, "/ui", nil)
		rr := httptest.NewRecorder()

		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", rr.Code)
		}

		allow := rr.Header().Get("Allow")
		if !strings.Contains(allow, "GET") || !strings.Contains(allow, "HEAD") {
			t.Errorf("Allow header should contain GET and HEAD, got %s", allow)
		}
	})

	t.Run("404 for unknown routes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/unknown/path", nil)
		rr := httptest.NewRecorder()

		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rr.Code)
		}
	})
}
