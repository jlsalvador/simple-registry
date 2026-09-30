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
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/internal/http/handler"
	"github.com/jlsalvador/simple-registry/pkg/rbac"
)

func TestCatalog(t *testing.T) {
	tests := []struct {
		name     string
		requests []testRequestBuilder
	}{
		{
			name: "successful index with proper basic auth",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodGet, "/v2/", nil)
						r.SetBasicAuth(testUser, testPwd)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "unsuccessful index with invalid basic auth",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodGet, "/v2/", nil)
						r.SetBasicAuth("invalid", "invalid")
						return r
					},
					http.StatusForbidden,
				},
			},
		},
		{
			name: "successful index with proper bearer auth",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						// Request to fetch token realm.
						return httptest.NewRequest(http.MethodGet, "/v2/", nil)
					},
					http.StatusUnauthorized,
				},
				{
					func(prevResp *http.Response) *http.Request {
						// Fetch realm from previous request.
						realmRegExp := regexp.MustCompile(`realm="([^"]+)"`)
						auth := prevResp.Header.Get("WWW-Authenticate")
						match := realmRegExp.FindStringSubmatch(auth)
						if len(match) != 2 || match[1] == "" {
							t.Error("can not fetch realm from previous request")
						}
						realm := match[1]

						// Fetch token with valid credentials from realm.
						r := httptest.NewRequest(http.MethodGet, realm, nil)
						r.SetBasicAuth(testUser, testPwd)
						return r
					},
					http.StatusOK,
				},
				{
					func(prevResp *http.Response) *http.Request {
						// Fetch token from previous response.
						defer prevResp.Body.Close()
						var resp struct {
							Token string `json:"token"`
						}
						if err := json.NewDecoder(prevResp.Body).Decode(&resp); err != nil {
							t.Fatal(err)
						}

						// Request to /v2/ with bearer token.
						r := httptest.NewRequest(http.MethodGet, "/v2/", nil)
						r.Header.Set("Authorization", "Bearer "+resp.Token)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "index requires token, even when anonymous user is enabled",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						return httptest.NewRequest(http.MethodGet, "/v2/", nil)
					},
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "successful catalog list with empty repository",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodGet, "/v2/_catalog", nil)
						r.SetBasicAuth(testUser, testPwd)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "successful catalog list with some images",
			requests: []testRequestBuilder{
				{
					func(prevResp *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", bytes.NewReader([]byte("hola")))
						r.SetBasicAuth(testUser, testPwd)
						r.Header.Set("Content-Type", "application/octet-stream")
						r.Header.Set("Content-Length", "4")
						r.Header.Set("Content-Range", "0-4")
						return r
					},
					http.StatusAccepted,
				},
				testRequestBuilderPutManifests,
				{
					func(_ *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodGet, "/v2/_catalog", nil)
						r.SetBasicAuth(testUser, testPwd)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "unsuccessful catalog list without auth",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodGet, "/v2/_catalog", nil)
						return r
					},
					http.StatusUnauthorized,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mux := testSetupTestServeMux(t)

			var prev *http.Response
			for i, trb := range tt.requests {
				w := httptest.NewRecorder()
				r := trb.requestFn(prev)
				mux.ServeHTTP(w, r)
				prev = w.Result()

				if prev.StatusCode != trb.statusCode {
					body, _ := io.ReadAll(prev.Body)
					t.Errorf("step: %d, want %d, got %d. body: %s", i+1, trb.statusCode, prev.StatusCode, string(body))
				}
			}
		})
	}
}

// TestCatalogListsOnlyAllowedRepositories checks that /v2/_catalog only
// returns the repositories the user is allowed to read. The catalog request
// itself is not bound to a repository, so it is granted by a "^$" scope,
// while the listing is limited by the repository scopes.
func TestCatalogListsOnlyAllowedRepositories(t *testing.T) {
	dataDir := t.TempDir()

	cfg, err := config.New(
		config.WithAdminName(testUser),
		config.WithAdminPwd([]byte(testPwd)),
		config.WithDataDir(dataDir),
		config.WithHttpTokenSecret([]byte(testTokenSecret)),
	)
	if err != nil {
		t.Fatal(err)
	}

	// Two repositories: one inside the scoped binding, one private.
	for _, repo := range []string{"public/image", "private/image"} {
		if err := os.MkdirAll(
			filepath.Join(dataDir, "repositories", repo, "_manifests"),
			0o750,
		); err != nil {
			t.Fatal(err)
		}
	}

	cfg.Rbac.Users = append(cfg.Rbac.Users, rbac.User{
		Name:         "catalog_user",
		PasswordHash: testPwdHash,
	})
	cfg.Rbac.Roles = append(cfg.Rbac.Roles, rbac.Role{
		Name:      "catalog_reader",
		Resources: []string{"catalog"},
		Verbs:     []string{http.MethodGet},
	})
	cfg.Rbac.RoleBindings = append(cfg.Rbac.RoleBindings,
		rbac.RoleBinding{
			Name:     "catalog_gate",
			Subjects: []rbac.Subject{{Kind: "User", Name: "catalog_user"}},
			RoleName: "catalog_reader",
			Scopes:   []regexp.Regexp{*regexp.MustCompile(`^$`)},
		},
		rbac.RoleBinding{
			Name:     "catalog_public",
			Subjects: []rbac.Subject{{Kind: "User", Name: "catalog_user"}},
			RoleName: "catalog_reader",
			Scopes:   []regexp.Regexp{*regexp.MustCompile(`^public/.+$`)},
		},
	)

	mux := handler.NewHandler(*cfg)

	catalog := func(user string) []string {
		t.Helper()

		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/v2/_catalog", nil)
		r.SetBasicAuth(user, testPwd)
		mux.ServeHTTP(w, r)

		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("%s: want %d, got %d. body: %s",
				user, http.StatusOK, w.Result().StatusCode, w.Body.String())
		}

		var resp struct {
			Repositories []string `json:"repositories"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}

		return resp.Repositories
	}

	if got := catalog("catalog_user"); !slices.Equal(got, []string{"public/image"}) {
		t.Errorf("scoped user: want [public/image], got %v", got)
	}

	if got := catalog(testUser); !slices.Contains(got, "public/image") ||
		!slices.Contains(got, "private/image") {
		t.Errorf("admin: want both repositories, got %v", got)
	}
}
