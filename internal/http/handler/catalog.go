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

package handler

import (
	"encoding/json"
	"errors"
	"io/fs"
	netHttp "net/http"
	"slices"

	"github.com/jlsalvador/simple-registry/pkg/http"
)

// Index returns if the registry requires authentication.
//
// # Route pattern:
//
//	"GET /v2/"
//
// # HTTP status codes:
//   - 200 OK           - The request is authenticated.
//   - 401 Unauthorized - The request is not authenticated.
//   - 403 Forbidden    - The request is unproperly authenticated.
func (m *ServeMux) Index(
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	if !m.IsValidAuth(r) {
		ChallengeRequest(w, r)
		return
	}

	w.WriteHeader(netHttp.StatusOK)
}

// CatalogList returns a list of the repositories.
//
// # Route pattern:
//
//	"GET /v2/_catalog"
//
// # HTTP status codes:
//   - 200 OK
//   - 401 Unauthorized
//   - 403 Forbidden
//   - 500 Internal Server Error
func (m *ServeMux) CatalogList(
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	// The catalog request is not bound to a repository, so it is checked
	// against an empty scope: rolebindings granting catalog access use the
	// "^$" scope for it (see docs/role-based-access-control.md).
	username, ok := m.RequestUsername(r)
	if !ok || !m.cfg.Rbac.IsAllowed(username, "catalog", "", netHttp.MethodGet) {
		ChallengeRequest(w, r)
		return
	}

	// Fetch repositories from storage.
	repos, err := m.cfg.Data.RepositoriesList()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// If the directory does not exist, return an empty list.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(netHttp.StatusOK)
			w.Write([]byte(`{"repositories":[]}`))
			return
		}

		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	// Only the repositories allowed for the user are listed. The username was
	// resolved above, so this does not validate the credentials per repository.
	repos = slices.DeleteFunc(repos, func(repo string) bool {
		return !m.cfg.Rbac.IsAllowed(username, "catalog", repo, netHttp.MethodGet)
	})

	slices.Sort(repos)

	repos = http.PaginateString(repos, r)

	response := map[string][]string{
		"repositories": repos,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(netHttp.StatusOK)
	json.NewEncoder(w).Encode(response)
}
