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

// Tests in the handler package itself, so they can build a *ServeMux
// directly and call IsRequestAllowed/GetClaimFromToken without going
// through the logging middleware wrapper returned by NewHandler.
package handler

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/pkg/rbac"

	"golang.org/x/crypto/bcrypt"
)

const (
	authTestUser        = "authuser"
	authTestPwd         = "authpwd"
	authTestUserNoPerms = "without"
	authTestPwdNoPerms  = "without"
	authTestTokenSecret = "authTestTokenSecret"
)

// authTestBuildMux builds a *ServeMux with an admin user (authTestUser),
// a user without permissions (authTestUserNoPerms) and, optionally, an
// anonymous user that can only access "public/*" repositories.
func authTestBuildMux(t *testing.T, withAnonymous bool) *ServeMux {
	t.Helper()

	cfg, err := config.New(
		config.WithAdminName(authTestUser),
		config.WithAdminPwd([]byte(authTestPwd)),
		config.WithDataDir(t.TempDir()),
		config.WithHttpTokenSecret([]byte(authTestTokenSecret)),
	)
	if err != nil {
		t.Fatal(err)
	}

	noPermsHash, err := bcrypt.GenerateFromPassword([]byte(authTestPwdNoPerms), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Rbac.Users = append(cfg.Rbac.Users, rbac.User{
		Name:         authTestUserNoPerms,
		PasswordHash: string(noPermsHash),
	})

	if withAnonymous {
		cfg.Rbac.Users = append(cfg.Rbac.Users, rbac.User{
			Name: rbac.AnonymousUsername,
		})

		cfg.Rbac.Roles = append(cfg.Rbac.Roles, rbac.Role{
			Name:      "everything",
			Resources: []string{"*"},
			Verbs: []string{
				http.MethodHead, http.MethodGet, http.MethodPost,
				http.MethodPut, http.MethodPatch, http.MethodDelete,
			},
		})
		cfg.Rbac.RoleBindings = append(cfg.Rbac.RoleBindings, rbac.RoleBinding{
			Name:     "anonymous_to_just_one_repo",
			Subjects: []rbac.Subject{{Kind: "User", Name: rbac.AnonymousUsername}},
			RoleName: "everything",
			Scopes:   []regexp.Regexp{*regexp.MustCompile(`^public/.+$`)},
		})
	}

	mux := &ServeMux{cfg: *cfg, mux: http.NewServeMux()}
	mux.registerRoutes()
	return mux
}

// Helper function to create a valid token.
func authTestValidToken(t *testing.T, user, scope string) string {
	t.Helper()
	token, err := GenerateToken([]byte(authTestTokenSecret), user, scope)
	if err != nil {
		t.Fatal(err)
	}
	return token
}

// Helper function to create an expired token.
func authTestExpiredToken(t *testing.T) string {
	t.Helper()
	return authTestBuildToken(t, map[string]any{
		"sub":   authTestUser,
		"scope": "repository:test/repo:pull",
		"iat":   time.Now().Add(-2 * time.Hour).Unix(), // Expired 2 hours ago.
	})
}

// Helper function to create a token with a non-string sub claim.
func authTestNonStringSubToken(t *testing.T) string {
	t.Helper()
	return authTestBuildToken(t, map[string]any{
		"sub":   123, // Not string.
		"scope": "repository:test/repo:pull",
		"iat":   time.Now().Unix(),
	})
}

// Helper function to create a token with a malformed (non-JSON) payload.
func authTestInvalidPayloadToken(t *testing.T) string {
	t.Helper()
	headerJSON := `{"alg":"HS512","typ":"JWT"}`
	header := base64.RawURLEncoding.EncodeToString([]byte(headerJSON))
	payload := base64.RawURLEncoding.EncodeToString([]byte("invalidjson"))
	return authTestSign(t, header, payload)
}

func authTestBuildToken(t *testing.T, payloadMap map[string]any) string {
	t.Helper()
	headerJSON := `{"alg":"HS512","typ":"JWT"}`
	header := base64.RawURLEncoding.EncodeToString([]byte(headerJSON))
	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		t.Fatal(err)
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	return authTestSign(t, header, payload)
}

func authTestSign(t *testing.T, header, payload string) string {
	t.Helper()
	signingInput := header + "." + payload
	h := hmac.New(sha512.New, []byte(authTestTokenSecret))
	h.Write([]byte(signingInput))
	signatureB64 := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return signingInput + "." + signatureB64
}

func TestGetClaimFromToken(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		wantClaims  bool
		wantSubject string
	}{
		{
			name:        "valid token",
			authHeader:  "Bearer " + authTestValidToken(t, "admin", "repository:test/repo:pull"),
			wantClaims:  true,
			wantSubject: "admin",
		},
		{
			name:       "invalid token format",
			authHeader: "Bearer invalid.token.here",
			wantClaims: false,
		},
		{
			name:       "missing bearer prefix",
			authHeader: "invalid.token.here",
			wantClaims: false,
		},
		{
			name:       "empty auth header",
			authHeader: "",
			wantClaims: false,
		},
		{
			name:       "expired token",
			authHeader: "Bearer " + authTestExpiredToken(t),
			wantClaims: false,
		},
		{
			name:       "invalid payload base64",
			authHeader: "Bearer " + base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS512","typ":"JWT"}`)) + ".invalidbase64." + base64.RawURLEncoding.EncodeToString([]byte("sig")),
			wantClaims: false,
		},
		{
			name:       "invalid json in payload",
			authHeader: "Bearer " + authTestInvalidPayloadToken(t),
			wantClaims: false,
		},
		{
			name:       "sub not string",
			authHeader: "Bearer " + authTestNonStringSubToken(t),
			wantClaims: true, // Claims ok, but sub is not a string.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mux := authTestBuildMux(t, true)

			r := httptest.NewRequest(http.MethodGet, "/v2/test/blobs/sha256:123", nil)
			if tt.authHeader != "" {
				r.Header.Set("Authorization", tt.authHeader)
			}

			claims, ok := mux.GetClaimFromToken(r)

			if ok != tt.wantClaims {
				t.Errorf("GetClaimFromToken() ok = %v, want %v", ok, tt.wantClaims)
			}

			if ok && tt.wantSubject != "" {
				subject, isStr := claims["sub"].(string)
				if !isStr || subject != tt.wantSubject {
					t.Errorf("GetClaimFromToken() subject = %v, want %v", subject, tt.wantSubject)
				}
			}

			if tt.name == "sub not string" {
				if _, isStr := claims["sub"].(string); isStr {
					t.Errorf("GetClaimFromToken() sub = %v, want non-string", claims["sub"])
				}
			}
		})
	}
}

func TestIsRequestAllowed(t *testing.T) {
	tests := []struct {
		name        string
		authHeader  string
		noAnonymous bool
		resource    string
		scope       string
		want        bool
	}{
		{
			name:     "anonymous allowed on public repo",
			resource: "blobs",
			scope:    "public/busybox",
			want:     true,
		},
		{
			name:     "anonymous not allowed on private repo",
			resource: "blobs",
			scope:    "private/busybox",
			want:     false,
		},
		{
			name:        "no auth and anonymous disabled",
			noAnonymous: true,
			want:        false,
		},
		{
			name:       "valid basic auth as admin",
			authHeader: testBasicAuthHeader(t, authTestUser, authTestPwd),
			resource:   "blobs",
			scope:      "testrepo",
			want:       true,
		},
		{
			name:       "basic auth user without permissions",
			authHeader: testBasicAuthHeader(t, authTestUserNoPerms, authTestPwdNoPerms),
			resource:   "blobs",
			scope:      "testrepo",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mux := authTestBuildMux(t, !tt.noAnonymous)

			r := httptest.NewRequest(http.MethodGet, "/v2/public/image", nil)
			if tt.authHeader != "" {
				r.Header.Set("Authorization", tt.authHeader)
			}

			got := mux.IsRequestAllowed(r, tt.resource, tt.scope, http.MethodGet)
			if got != tt.want {
				t.Errorf("IsRequestAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsBearerAllowed(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		resource   string
		scope      string
		verb       string
		want       bool
	}{
		{
			name:       "valid bearer token with correct permissions",
			authHeader: "Bearer " + authTestValidToken(t, authTestUser, "repository:testrepo:pull"),
			resource:   "blobs",
			scope:      "testrepo",
			verb:       http.MethodGet,
			want:       true,
		},
		{
			name:       "bearer token for user without permissions",
			authHeader: "Bearer " + authTestValidToken(t, authTestUserNoPerms, "repository:testrepo:pull"),
			resource:   "blobs",
			scope:      "testrepo",
			verb:       http.MethodGet,
			want:       false,
		},
		{
			name:       "bearer token for scope not allowed to user",
			authHeader: "Bearer " + authTestValidToken(t, rbac.AnonymousUsername, "repository:otherrepo:pull"),
			resource:   "blobs",
			scope:      "otherrepo",
			verb:       http.MethodGet,
			want:       false,
		},
		{
			name:       "bearer token for anonymous on public repo",
			authHeader: "Bearer " + authTestValidToken(t, rbac.AnonymousUsername, "repository:public/busybox:pull"),
			resource:   "blobs",
			scope:      "public/busybox",
			verb:       http.MethodGet,
			want:       true,
		},
		{
			name:       "invalid token",
			authHeader: "Bearer invalid.token.here",
			resource:   "blobs",
			scope:      "testrepo",
			verb:       http.MethodGet,
			want:       false,
		},
		{
			name:       "expired token",
			authHeader: "Bearer " + authTestExpiredToken(t),
			resource:   "blobs",
			scope:      "testrepo",
			verb:       http.MethodGet,
			want:       false,
		},
		{
			name:       "token with non-string sub",
			authHeader: "Bearer " + authTestNonStringSubToken(t),
			resource:   "blobs",
			scope:      "testrepo",
			verb:       http.MethodGet,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mux := authTestBuildMux(t, true)

			r := httptest.NewRequest(http.MethodGet, "/v2/test/blobs/sha256:123", nil)
			r.Header.Set("Authorization", tt.authHeader)

			got := mux.IsRequestAllowed(r, tt.resource, tt.scope, tt.verb)
			if got != tt.want {
				t.Errorf("IsRequestAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsBasicAuthAllowed(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		want       bool
	}{
		{
			name:       "valid basic auth",
			authHeader: testBasicAuthHeader(t, authTestUser, authTestPwd),
			want:       true,
		},
		{
			name:       "invalid basic auth header",
			authHeader: "Basic invalidbase64",
			want:       false,
		},
		{
			name:       "wrong credentials",
			authHeader: testBasicAuthHeader(t, authTestUser, "wrong"),
			want:       false,
		},
		{
			name:       "user without permissions",
			authHeader: testBasicAuthHeader(t, authTestUserNoPerms, authTestPwdNoPerms),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mux := authTestBuildMux(t, true)

			r := httptest.NewRequest(http.MethodGet, "/v2/", nil)
			r.Header.Set("Authorization", tt.authHeader)

			got := mux.IsRequestAllowed(r, "", "", http.MethodGet)
			if got != tt.want {
				t.Errorf("IsRequestAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func testBasicAuthHeader(t *testing.T, user, pwd string) string {
	t.Helper()
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pwd))
}
