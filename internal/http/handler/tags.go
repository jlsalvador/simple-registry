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
	"regexp"
	"slices"
	"strconv"

	httpErrors "github.com/jlsalvador/simple-registry/pkg/http/errors"
	"github.com/jlsalvador/simple-registry/pkg/rbac"
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
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	username, err := m.cfg.Rbac.GetUsernameFromHttpRequest(r)
	if err, ok := err.(*httpErrors.HttpError); ok {
		w.WriteHeader(err.Status)
		return
	}

	// "repo" must be a valid repository name.
	repo := r.PathValue("name")
	if !regexp.MustCompile(registry.RegExpName).MatchString(repo) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// Check if the user can list tags from this manifest.
	if !m.cfg.Rbac.IsAllowed(username, "tags", repo, netHttp.MethodGet) {
		if username == rbac.AnonymousUsername {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		}
	}

	tags, err := m.cfg.Data.TagsList(repo)
	if err != nil {
		// Some repos may not exist, Docker expects 404
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(netHttp.StatusNotFound)
			return
		}

		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	// Filter tags by user permissions.
	tags = slices.DeleteFunc(tags, func(t string) bool {
		resource := repo + ":" + t
		return !m.cfg.Rbac.IsAllowed(username, "tags", resource, netHttp.MethodGet)
	})

	slices.Sort(tags)

	// "last" is an optional parameter.
	// If it's provided, remove all tags BEFORE and INCLUDING it.
	last := r.URL.Query().Get("last")
	if last != "" {
		// Find the index of "last".
		foundIndex := -1
		for i, v := range tags {
			if v == last {
				foundIndex = i
				break
			}
		}

		// If "last" was found, start the slice AFTER it.
		if foundIndex != -1 {
			// Remove values from tags up to and including "last".
			tags = tags[foundIndex+1:]
		}
	}

	// "n" is an optional parameter.
	// If it's provided, limit the number of tags returned.
	n := r.URL.Query().Get("n")
	if n != "" {
		nInt, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			w.WriteHeader(netHttp.StatusBadRequest)
			return
		}
		// Limit the number of tags returned.
		if nInt < 0 || nInt > int64(len(tags)) {
			w.WriteHeader(netHttp.StatusBadRequest)
			return
		}
		tags = tags[:nInt]
	}

	response := map[string]any{
		"name": repo,
		"tags": tags,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(netHttp.StatusOK)
	json.NewEncoder(w).Encode(response)
}
