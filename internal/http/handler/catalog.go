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
	netHttp "net/http"

	"github.com/jlsalvador/simple-registry/pkg/http"
	"github.com/jlsalvador/simple-registry/pkg/rbac"
)

// Index just for authorization testing and ping.
// If anonymous access is allowed, it will return a 200 OK.
//
// # Route pattern:
//
//	"GET /v2/"
//
// # HTTP status codes:
//   - 200 OK
//   - 401 Unauthorized
func (m *ServeMux) Index(
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	username, err := m.cfg.Rbac.GetUsernameFromHttpRequest(r)
	if err, ok := err.(*http.HttpError); ok {
		w.WriteHeader(err.Status)
		return
	}

	// Check if user is allowed to access this registry.
	if !m.cfg.Rbac.IsAllowed(username, "", "", netHttp.MethodGet) {
		if username == rbac.AnonymousUsername {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		}
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
	username, err := m.cfg.Rbac.GetUsernameFromHttpRequest(r)
	if err, ok := err.(*http.HttpError); ok {
		w.WriteHeader(err.Status)
		return
	}

	// Check if user has permission to access the catalog.
	if !m.cfg.Rbac.IsAllowed(username, "catalog", "", netHttp.MethodGet) {
		if username == rbac.AnonymousUsername {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		}
	}

	// Fetch repositories from storage.
	repos, err := m.cfg.Data.RepositoriesList()
	if err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	response := map[string][]string{
		"repositories": repos,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(netHttp.StatusOK)
	json.NewEncoder(w).Encode(response)
}
