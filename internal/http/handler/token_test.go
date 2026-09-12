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
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/http/handler"
)

func TestGenerateToken(t *testing.T) {
	secret := []byte(testTokenSecret)
	user := testUser
	scope := "repository:test/repo:pull"

	token, err := handler.GenerateToken(secret, user, scope)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if token == "" {
		t.Error("GenerateToken() returned empty token")
	}
}

func TestToken(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		scope          string
		expectedStatus int
		expectToken    bool
	}{
		{
			name:           "no auth header",
			authHeader:     "",
			scope:          "repository:test/repo:pull",
			expectedStatus: http.StatusUnauthorized,
			expectToken:    false,
		},
		{
			name:           "invalid basic auth",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte("wrong:wrong")),
			scope:          "repository:test/repo:pull",
			expectedStatus: http.StatusForbidden,
			expectToken:    false,
		},
		{
			name:           "valid basic auth",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte(testUser+":"+testPwd)),
			scope:          "repository:test/repo:pull",
			expectedStatus: http.StatusOK,
			expectToken:    true,
		},
		{
			name:           "valid basic auth with multiple scopes",
			authHeader:     "Basic " + base64.StdEncoding.EncodeToString([]byte(testUser+":"+testPwd)),
			scope:          "repository:test/repo:pull,push",
			expectedStatus: http.StatusOK,
			expectToken:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mux := testSetupTestServeMux(t)

			r := httptest.NewRequest(http.MethodGet, "/token?scope="+tt.scope, nil)
			if tt.authHeader != "" {
				r.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, r)

			if w.Code != tt.expectedStatus {
				t.Errorf("Token() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.expectToken {
				var resp map[string]string
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Errorf("Failed to decode response: %v", err)
				}
				if resp["token"] == "" {
					t.Error("Expected token in response")
				}
			}
		})
	}
}
