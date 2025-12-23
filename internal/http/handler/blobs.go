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
	netHttp "net/http"

	"github.com/jlsalvador/simple-registry/pkg/rbac"
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
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	username, err := m.authenticate(w, r)
	if err != nil {
		return
	}

	// "repo" must be a valid repository name.
	repo := r.PathValue("name")
	if !registry.RegExprName.MatchString(repo) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// "digest" must be a valid digest.
	digest := r.PathValue("digest")
	if !registry.RegExprDigest.MatchString(digest) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// Check if the user have permission to pull the repository.
	if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, netHttp.MethodGet) {
		if username == rbac.AnonymousUsername {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		}
	}

	blob, size, err := m.cfg.Data.BlobsGet(repo, digest)
	if err != nil {
		// Docker expects 404 when the blob does not exist.
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(netHttp.StatusNotFound)
			return
		}

		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}
	defer blob.Close()

	w.Header().Set("Docker-Content-Digest", digest)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.WriteHeader(netHttp.StatusOK)
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
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	username, err := m.authenticate(w, r)
	if err != nil {
		return
	}

	// "repo" must be a valid repository name.
	repo := r.PathValue("name")
	if !registry.RegExprName.MatchString(repo) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// "digest" must be a valid digest.
	digest := r.PathValue("digest")
	if !registry.RegExprDigest.MatchString(digest) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// Check if the user have permission to delete blobs.
	if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, netHttp.MethodDelete) {
		if username == rbac.AnonymousUsername {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		}
	}

	err = m.cfg.Data.BlobsDelete(repo, digest)
	if err != nil {
		// Docker expects 404 when the blob does not exist.
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(netHttp.StatusNotFound)
			return
		}

		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	// Docker Registry spec: return 202 Accepted
	w.WriteHeader(netHttp.StatusAccepted)
}
