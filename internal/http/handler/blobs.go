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
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"regexp"

	"github.com/jlsalvador/simple-registry/internal/rbac"
	d "github.com/jlsalvador/simple-registry/pkg/digest"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

// BlobsGet retrieves a blob from the registry.
//
// # Route pattern:
//
//	"GET /v2/{name}/blobs/{digest}"
//
// # Path params:
//   - {name}		must be a valid repository name.
//   - {digest}		must be a valid digest.
//
// # HTTP status codes:
//   - 200 OK
//   - 400 Bad Request
//   - 401 Unauthorized
//   - 403 Forbidden
//   - 404 Not Found
//   - 500 Internal Server Error
func (m *ServeMux) BlobsGet(
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

	// "digest" must be a valid digest.
	digest := r.PathValue("digest")
	if _, _, err := d.Parse(digest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if the user have permission to pull the repository.
	if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, rbac.ActionPull) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	blob, size, err := m.cfg.Data.BlobsGet(repo, digest)
	if err != nil {
		// Docker expects 404 when the blob does not exist.
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer blob.Close()

	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, blob)
}

// BlobsDelete deletes a blob from the registry.
//
// # Route pattern:
//
//	"DELETE /v2/{name}/blobs/{digest}"
//
// # Path params:
//   - {name}		must be a valid repository name.
//   - {digest}		must be a valid digest.
//
// # HTTP status codes:
//   - 202 Accepted
//   - 400 Bad Request
//   - 403 Forbidden
//   - 404 Not Found
//   - 401 Unauthorized
//   - 500 Internal Server Error
func (m *ServeMux) BlobsDelete(
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

	// "digest" must be a valid digest.
	digest := r.PathValue("digest")
	if _, _, err := d.Parse(digest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if the user have permission to delete blobs.
	if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, rbac.ActionDelete) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = m.cfg.Data.BlobsDelete(repo, digest)
	if err != nil {
		// Docker expects 404 when the blob does not exist.
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Docker Registry spec: return 202 Accepted
	w.WriteHeader(http.StatusAccepted)
}
