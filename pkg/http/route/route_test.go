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

package route_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/http/route"
)

func TestNewRoute(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		pattern string
	}{
		{
			name:    "creates simple GET route",
			method:  http.MethodGet,
			pattern: "^/users$",
		},
		{
			name:    "creates POST route with parameters",
			pattern: "^/users/(?P<id>[0-9]+)$",
			method:  http.MethodPost,
		},
		{
			name:    "creates route with multiple parameters",
			pattern: "^/users/(?P<userId>[0-9]+)/posts/(?P<postId>[0-9]+)$",
			method:  http.MethodGet,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := func(w http.ResponseWriter, r *http.Request) {}
			rt := route.NewRoute(tt.method, tt.pattern, handler)

			if rt.Method != tt.method {
				t.Errorf("expected method %s, got %s", tt.method, rt.Method)
			}
		})
	}
}

func TestRoute_Handler_MatchingPath(t *testing.T) {
	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}

	rt := route.NewRoute(http.MethodGet, "^/users$", handler)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	rt.Handler(w, req)

	if !handlerCalled {
		t.Error("Handler should have been called for matching route")
	}

	if !rt.IsMatchUrl {
		t.Error("IsMatchUrl should be true for matching URL")
	}

	if !rt.IsMatchMethod {
		t.Error("IsMatchMethod should be true for matching method")
	}
}

func TestRoute_Handler_NonMatchingPath(t *testing.T) {
	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}

	rt := route.NewRoute(http.MethodGet, "^/users$", handler)

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()

	rt.Handler(w, req)

	if handlerCalled {
		t.Error("Handler should not have been called for non-matching route")
	}

	if rt.IsMatchUrl {
		t.Error("IsMatchUrl should be false for non-matching URL")
	}

	if rt.IsMatchMethod {
		t.Error("IsMatchMethod should be false when URL doesn't match")
	}
}

func TestRoute_Handler_MatchingPathWrongMethod(t *testing.T) {
	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}

	rt := route.NewRoute(http.MethodGet, "^/users$", handler)

	req := httptest.NewRequest(http.MethodPost, "/users", nil)
	w := httptest.NewRecorder()

	rt.Handler(w, req)

	if handlerCalled {
		t.Error("Handler should not have been called for wrong method")
	}

	if !rt.IsMatchUrl {
		t.Error("IsMatchUrl should be true when URL matches")
	}

	if rt.IsMatchMethod {
		t.Error("IsMatchMethod should be false when method doesn't match")
	}
}

func TestRoute_Handler_PathParameters(t *testing.T) {
	var capturedID string
	handler := func(w http.ResponseWriter, r *http.Request) {
		capturedID = r.PathValue("id")
	}

	rt := route.NewRoute(http.MethodGet, "^/users/(?P<id>[0-9]+)$", handler)

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()

	rt.Handler(w, req)

	if capturedID != "123" {
		t.Errorf("expected path parameter id='123', got '%s'", capturedID)
	}
}

func TestRoute_Handler_MultiplePathParameters(t *testing.T) {
	var capturedUserID, capturedPostID string
	handler := func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = r.PathValue("userId")
		capturedPostID = r.PathValue("postId")
	}

	rt := route.NewRoute(
		http.MethodGet,
		"^/users/(?P<userId>[0-9]+)/posts/(?P<postId>[0-9]+)$",
		handler,
	)

	req := httptest.NewRequest(http.MethodGet, "/users/456/posts/789", nil)
	w := httptest.NewRecorder()

	rt.Handler(w, req)

	if capturedUserID != "456" {
		t.Errorf("expected userId='456', got '%s'", capturedUserID)
	}

	if capturedPostID != "789" {
		t.Errorf("expected postId='789', got '%s'", capturedPostID)
	}
}

func TestRoute_Handler_HeadRequest(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		// Handler writes a body, which should be discarded for HEAD.
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("This is the response body"))
	}

	rt := route.NewRoute(http.MethodGet, "^/users$", handler)

	req := httptest.NewRequest(http.MethodHead, "/users", nil)
	w := httptest.NewRecorder()

	rt.Handler(w, req)

	if !rt.IsMatchUrl {
		t.Error("IsMatchUrl should be true for HEAD matching GET route")
	}

	if !rt.IsMatchMethod {
		t.Error("IsMatchMethod should be true for HEAD matching GET route")
	}

	// The body should have been discarded.
	if w.Body.Len() != 0 {
		t.Errorf("HEAD request should have no body, got %d bytes", w.Body.Len())
	}

	// Status code should still be set.
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRoute_Handler_HeadRequestWithNonGetRoute(t *testing.T) {
	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	}

	rt := route.NewRoute(http.MethodPost, "^/users$", handler)

	req := httptest.NewRequest(http.MethodHead, "/users", nil)
	w := httptest.NewRecorder()

	rt.Handler(w, req)

	if handlerCalled {
		t.Error("HEAD should not match POST route")
	}

	if !rt.IsMatchUrl {
		t.Error("IsMatchUrl should be true when URL matches")
	}

	if rt.IsMatchMethod {
		t.Error("IsMatchMethod should be false when HEAD doesn't match POST")
	}
}

func TestRoute_Handler_ComplexPattern(t *testing.T) {
	tests := []struct {
		name          string
		pattern       string
		requestPath   string
		shouldMatch   bool
		expectedParam map[string]string
	}{
		{
			name:        "matches UUID pattern",
			pattern:     "^/users/(?P<id>[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})$",
			requestPath: "/users/550e8400-e29b-41d4-a716-446655440000",
			shouldMatch: true,
			expectedParam: map[string]string{
				"id": "550e8400-e29b-41d4-a716-446655440000",
			},
		},
		{
			name:        "doesn't match invalid UUID",
			pattern:     "^/users/(?P<id>[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})$",
			requestPath: "/users/invalid-uuid",
			shouldMatch: false,
		},
		{
			name:        "matches slug pattern",
			pattern:     "^/posts/(?P<slug>[a-z0-9-]+)$",
			requestPath: "/posts/my-awesome-post",
			shouldMatch: true,
			expectedParam: map[string]string{
				"slug": "my-awesome-post",
			},
		},
		{
			name:        "matches optional trailing slash",
			pattern:     "^/api/users/?$",
			requestPath: "/api/users/",
			shouldMatch: true,
		},
		{
			name:        "matches without trailing slash",
			pattern:     "^/api/users/?$",
			requestPath: "/api/users",
			shouldMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturedParams := make(map[string]string)
			handler := func(w http.ResponseWriter, r *http.Request) {
				for key := range tt.expectedParam {
					capturedParams[key] = r.PathValue(key)
				}
			}

			rt := route.NewRoute(http.MethodGet, tt.pattern, handler)

			req := httptest.NewRequest(http.MethodGet, tt.requestPath, nil)
			w := httptest.NewRecorder()

			rt.Handler(w, req)

			if rt.IsMatchUrl != tt.shouldMatch {
				t.Errorf("expected IsMatchUrl=%v, got %v", tt.shouldMatch, rt.IsMatchUrl)
			}

			if tt.shouldMatch {
				for key, expectedValue := range tt.expectedParam {
					if capturedParams[key] != expectedValue {
						t.Errorf("expected param %s='%s', got '%s'", key, expectedValue, capturedParams[key])
					}
				}
			}
		})
	}
}

func TestRoute_Handler_HandlerPanic(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		panic("handler panic")
	}

	rt := route.NewRoute(http.MethodGet, "^/users$", handler)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	// Should panic, this is expected behavior.
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected handler to panic, but it didn't")
		}
	}()

	rt.Handler(w, req)
}

func TestRoute_Handler_WritesResponse(t *testing.T) {
	expectedBody := "Hello, World!"
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(expectedBody))
	}

	rt := route.NewRoute(http.MethodGet, "^/hello$", handler)

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := httptest.NewRecorder()

	rt.Handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if ct := w.Header().Get("Content-Type"); ct != "text/plain" {
		t.Errorf("expected Content-Type 'text/plain', got '%s'", ct)
	}

	if body := w.Body.String(); body != expectedBody {
		t.Errorf("expected body '%s', got '%s'", expectedBody, body)
	}
}

func BenchmarkRoute_Handler_SimpleMatch(b *testing.B) {
	handler := func(w http.ResponseWriter, r *http.Request) {}
	rt := route.NewRoute(http.MethodGet, "^/users$", handler)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	w := httptest.NewRecorder()

	for b.Loop() {
		rt.Handler(w, req)
	}
}

func BenchmarkRoute_Handler_WithParameters(b *testing.B) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		_ = r.PathValue("id")
	}
	rt := route.NewRoute(http.MethodGet, "^/users/(?P<id>[0-9]+)$", handler)

	req := httptest.NewRequest(http.MethodGet, "/users/123", nil)
	w := httptest.NewRecorder()

	for b.Loop() {
		rt.Handler(w, req)
	}
}

func BenchmarkRoute_Handler_NoMatch(b *testing.B) {
	handler := func(w http.ResponseWriter, r *http.Request) {}
	rt := route.NewRoute(http.MethodGet, "^/users$", handler)

	req := httptest.NewRequest(http.MethodGet, "/posts", nil)
	w := httptest.NewRecorder()

	for b.Loop() {
		rt.Handler(w, req)
	}
}
