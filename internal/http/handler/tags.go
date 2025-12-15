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
	"net/http"
	"regexp"
	"sort"
	"strconv"

	"github.com/jlsalvador/simple-registry/internal/rbac"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

// TagsList returns a list of tags for the given repository.
//
// # Route pattern:
//
//	"GET /v2/{name}/tags/list"
//
// # Path params:
//   - {name}		must be a valid repository name.
//
// # Url query params:
//   - "n" optional. Must be an int.
//   - "last" optional. Must be a tag name.
//
// # HTTP status codes:
//   - 200 OK
//   - 404 Not Found
//   - 401 Unauthorized
//   - 500 Internal Server Error
func (m *ServeMux) TagsList(
	w http.ResponseWriter,
	r *http.Request,
) {
	username, err := m.cfg.Rbac.GetUsernameFromHttpRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// "repo" must be a valid repository name.
	repo := r.PathValue("name")
	if !regexp.MustCompile(registry.RegExpName).MatchString(repo) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tags, err := m.cfg.Data.TagsList(repo)
	if err != nil {
		// Some repos may not exist, Docker expects 404
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Sort tags.
	sort.SliceStable(tags, func(i, j int) bool {
		return i > j
	})

	// "last" is an optional parameter.
	// If it's provided, remove all tags before it.
	last := r.URL.Query().Get("last")
	if last != "" {
		// Remove values from tags until "last" is found.
		for i, v := range tags {
			if v == last {
				tags = tags[i:]
				break
			}
		}
	}

	// "n" is an optional parameter.
	// If it's provided, limit the number of tags returned.
	n := r.URL.Query().Get("n")
	if n != "" {
		nInt, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// Limit the number of tags returned.
		if nInt < 0 || nInt > int64(len(tags)) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		tags = tags[:nInt]
	}

	// For each tag, check ACL
	for _, t := range tags {
		// ACL: user must have permission to list tags on this repository
		resource := repo + ":" + t
		if !m.cfg.Rbac.IsAllowed(username, "tags", resource, rbac.ActionPull) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
	}

	response := map[string]any{
		"name": repo,
		"tags": tags,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
