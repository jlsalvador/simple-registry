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
	// Check if user has permission to access the catalog.
	if !m.IsRequestAllowed(r, "catalog", "", netHttp.MethodGet) {
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

	slices.Sort(repos)

	repos = http.PaginateString(repos, r)

	response := map[string][]string{
		"repositories": repos,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(netHttp.StatusOK)
	json.NewEncoder(w).Encode(response)
}
