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
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"regexp"

	"github.com/jlsalvador/simple-registry/internal/rbac"
	"github.com/jlsalvador/simple-registry/pkg/digest"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

// ManifestsGet returns the manifest blob.
//
// # Route pattern:
//
//	"GET /v2/{name}/manifests/{reference}"
//
// # Path params:
//   - {name}		must be a valid repository name.
//   - {reference}	must be a digest or a tag name.
//
// # HTTP status codes:
//   - 200 OK
//   - 404 Not Found
//   - 401 Unauthorized
//   - 500 Internal Server Error
func (m *ServeMux) ManifestsGet(
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
	rbacRepo := repo

	// "reference" must be a digest or a tag.
	reference := r.PathValue("reference")
	if _, _, err := digest.Parse(reference); err == nil {
		// Do nothing.
	} else if regexp.MustCompile(registry.RegExpTag).MatchString(reference) {
		rbacRepo += ":" + reference
	} else {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if the user is allowed to pull this manifest.
	if !m.cfg.Rbac.IsAllowed(username, "manifests", rbacRepo, rbac.ActionPull) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Get the manifest blob from the data storage.
	blob, size, err := m.cfg.Data.ManifestGet(repo, reference)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer blob.Close()

	// Write the manifest blob to the response.
	header := w.Header()
	header.Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
	header.Set("Content-Length", fmt.Sprint(size))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, blob)
}

// ManifestsPut write a manifest to the data storage.
//
// # Route pattern:
//
//	"PUT /v2/<name>/manifests/<reference>"
//
// # Path params:
//   - {name}		must be a valid repository name.
//   - {reference}	must be a digest or a tag name.
//
// # HTTP status codes:
//   - 201 Created
//   - 400 Bad Request
//   - 401 Unauthorized
//   - 404 Not Found
//   - 413 Payload Too Large
//   - 500 Internal Server Error
func (m *ServeMux) ManifestsPut(
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
	rbacRepo := repo

	// "reference" must be a digest or a tag.
	reference := r.PathValue("reference")
	if _, _, err := digest.Parse(reference); err == nil {
		// "reference" is a valid digest.
	} else if regexp.MustCompile(registry.RegExpTag).MatchString(reference) {
		// "reference" is a tag.
		rbacRepo += ":" + reference
	} else {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if the user can to push manifests to the repository.
	if !m.cfg.Rbac.IsAllowed(username, "manifests", rbacRepo, rbac.ActionPush) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Store manifest.
	defer r.Body.Close()
	dgst, err := m.cfg.Data.ManifestPut(repo, reference, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Re-read the just written manifest.
	f, _, err := m.cfg.Data.ManifestGet(repo, reference)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	var manifest = &registry.Manifest{}
	if err := json.NewDecoder(f).Decode(manifest); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/v2/%s/manifests/%s", repo, dgst)
	header := w.Header()
	if manifest.Subject != nil && manifest.Subject.Digest != "" {
		header.Set("OCI-Subject", manifest.Subject.Digest)
	}
	header.Set("Location", location)
	header.Set("Docker-Content-Digest", dgst)
	w.WriteHeader(http.StatusCreated)
}

// ManifestsDelete deletes a manifest from the registry.
//
// # Route pattern:
//
//	"DELETE /v2/{name}/manifests/{reference}"
//
// # Path params:
//   - {name}		must be a valid repository name.
//   - {reference}	must be a digest or a tag name.
//
// # HTTP status codes:
//   - 202 Accepted
//   - 400 Bad Request
//   - 401 Unauthorized
//   - 403 Forbidden
//   - 404 Not Found
//   - 405 Method Not Allowed
func (m *ServeMux) ManifestsDelete(
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
	rbacRepo := repo

	// "reference" must be a digest or a tag.
	reference := r.PathValue("reference")
	if _, _, err := digest.Parse(reference); err == nil {
		// Do nothing.
	} else if regexp.MustCompile(registry.RegExpTag).MatchString(reference) {
		rbacRepo += ":" + reference
	} else {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !m.cfg.Rbac.IsAllowed(username, "manifests", rbacRepo, rbac.ActionDelete) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := m.cfg.Data.ManifestDelete(repo, reference); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
