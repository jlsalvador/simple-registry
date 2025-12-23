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
	netHttp "net/http"

	httpErrors "github.com/jlsalvador/simple-registry/pkg/http/errors"
	"github.com/jlsalvador/simple-registry/pkg/rbac"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func getContentType(manifest []byte) string {
	var aux struct {
		MediaType string `json:"mediaType"`
		Manifests []any  `json:"manifests"`
	}
	if err := json.Unmarshal(manifest, &aux); err != nil {
		return "application/vnd.oci.image.manifest.v1+json"
	}
	return aux.MediaType
}

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
	if !registry.RegExprName.MatchString(repo) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}
	rbacRepo := repo

	// "reference" must be a digest or a tag.
	reference := r.PathValue("reference")
	if registry.RegExprDigest.MatchString(reference) {
		// Do nothing if reference is a digest.
	} else if registry.RegExprTag.MatchString(reference) {
		rbacRepo += ":" + reference
	} else {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// Check if the user is allowed to pull this manifest.
	if !m.cfg.Rbac.IsAllowed(username, "manifests", rbacRepo, netHttp.MethodGet) {
		if username == rbac.AnonymousUsername {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		}
	}

	// Get the manifest blob from the data storage.
	blob, size, digest, err := m.cfg.Data.ManifestGet(repo, reference)
	if err != nil {
		if httpErr, ok := err.(httpErrors.HttpError); ok {
			w.WriteHeader(httpErr.Status)
			return
		}
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(netHttp.StatusNotFound)
			return
		}

		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}
	defer blob.Close()

	manifest, err := io.ReadAll(blob)
	if err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}
	contentType := getContentType(manifest)

	// Write the manifest blob to the response.
	header := w.Header()
	header.Set("Content-Type", contentType)
	header.Set("Content-Length", fmt.Sprint(size))
	header.Set("Docker-Content-Digest", digest)
	w.WriteHeader(netHttp.StatusOK)
	w.Write(manifest)
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
	if !registry.RegExprName.MatchString(repo) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}
	rbacRepo := repo

	// "reference" must be a digest or a tag.
	reference := r.PathValue("reference")
	if registry.RegExprDigest.MatchString(reference) {
		// "reference" is a valid digest.
	} else if registry.RegExprTag.MatchString(reference) {
		// "reference" is a tag.
		rbacRepo += ":" + reference
	} else {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// Check if the user can to push manifests to the repository.
	if !m.cfg.Rbac.IsAllowed(username, "manifests", rbacRepo, netHttp.MethodPut) {
		if username == rbac.AnonymousUsername {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		}
	}

	// Store manifest.
	defer r.Body.Close()
	dgst, err := m.cfg.Data.ManifestPut(repo, reference, r.Body)
	if err != nil {
		if httpErr, ok := err.(httpErrors.HttpError); ok {
			w.WriteHeader(httpErr.Status)
			return
		}

		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	// Re-read the just written manifest.
	f, _, _, err := m.cfg.Data.ManifestGet(repo, reference)
	if err != nil {
		if httpErr, ok := err.(httpErrors.HttpError); ok {
			w.WriteHeader(httpErr.Status)
			return
		}

		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}
	defer f.Close()

	var manifest = &registry.Manifest{}
	if err := json.NewDecoder(f).Decode(manifest); err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/v2/%s/manifests/%s", repo, dgst)
	header := w.Header()
	if manifest.Subject != nil && manifest.Subject.Digest != "" {
		header.Set("OCI-Subject", manifest.Subject.Digest)
	}
	header.Set("Location", location)
	header.Set("Docker-Content-Digest", dgst)
	w.WriteHeader(netHttp.StatusCreated)
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
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	username, err := m.cfg.Rbac.GetUsernameFromHttpRequest(r)
	if err != nil {
		if err, ok := err.(*httpErrors.HttpError); ok {
			w.WriteHeader(err.Status)
			return
		}

		w.WriteHeader(httpErrors.ErrInternalServerError.Status)
		return
	}

	// "repo" must be a valid repository name.
	repo := r.PathValue("name")
	if !registry.RegExprName.MatchString(repo) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}
	rbacRepo := repo

	// "reference" must be a digest or a tag.
	reference := r.PathValue("reference")
	if registry.RegExprDigest.MatchString(reference) {
		// Do nothing.
	} else if registry.RegExprTag.MatchString(reference) {
		rbacRepo += ":" + reference
	} else {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// Check if the user can delete manifests from the repository.
	if !m.cfg.Rbac.IsAllowed(username, "manifests", rbacRepo, netHttp.MethodDelete) {
		if username == rbac.AnonymousUsername {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		}
	}

	if err := m.cfg.Data.ManifestDelete(repo, reference); err != nil {
		if httpErr, ok := err.(httpErrors.HttpError); ok {
			w.WriteHeader(httpErr.Status)
			return
		}
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(netHttp.StatusNotFound)
			return
		}

		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	w.WriteHeader(netHttp.StatusAccepted)
}
