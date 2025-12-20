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
	"io/fs"
	netHttp "net/http"
	"regexp"

	"github.com/jlsalvador/simple-registry/internal/config"
	d "github.com/jlsalvador/simple-registry/pkg/digest"
	"github.com/jlsalvador/simple-registry/pkg/http"
	"github.com/jlsalvador/simple-registry/pkg/rbac"
	"github.com/jlsalvador/simple-registry/pkg/registry"

	u "github.com/google/uuid"
)

// blobsUploadsPostMount mounts blob from other repository.
// ACL must be already checked.
func blobsUploadsPostMount(
	cfg config.Config,
	repo string,
	from string,
	mount string,
	w netHttp.ResponseWriter,
) {
	f, _, err := cfg.Data.BlobsGet(from, mount)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Alternatively, if a registry does not support cross-repository
			// mounting or is unable to mount the requested blob, it SHOULD
			// return a 202. This indicates that the upload session has begun
			// and that the client MAY proceed with the upload.
			//
			// https://github.com/opencontainers/distribution-spec/blob/v1.1.1/spec.md#mounting-a-blob-from-another-repository
			blobsUploadsPostThenPut(cfg, repo, w)
			return
		}

		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}
	defer f.Close()

	uuid, err := cfg.Data.BlobsUploadCreate(repo)
	if err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	if err := cfg.Data.BlobsUploadWrite(repo, uuid, f, -1); err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	if err := cfg.Data.BlobsUploadCommit(repo, uuid, mount); err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/v2/%s/blobs/%s", repo, mount)
	w.Header().Set("Location", location)
	w.Header().Set("Docker-Content-Digest", mount)
	w.WriteHeader(netHttp.StatusCreated)
}

// blobsUploadsPostSingle uploads blob in a single POST.
// ACL must be already checked.
func blobsUploadsPostSingle(
	cfg config.Config,
	repo string,
	digest string,
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	uuid, err := cfg.Data.BlobsUploadCreate(repo)
	if err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}
	if err := cfg.Data.BlobsUploadWrite(repo, uuid, r.Body, 0); err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}
	if err := cfg.Data.BlobsUploadCommit(repo, uuid, digest); err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/v2/%s/blobs/%s", repo, digest)
	w.Header().Set("Location", location)
	w.WriteHeader(netHttp.StatusCreated)
}

// blobsUploadsPostThenPut creates upload blob session.
// ACL must be already checked.
func blobsUploadsPostThenPut(
	cfg config.Config,
	repo string,
	w netHttp.ResponseWriter,
) {
	uuid, err := cfg.Data.BlobsUploadCreate(repo)
	if err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/v2/%s/blobs/uploads/%s", repo, uuid)

	w.Header().Set("Location", location)
	w.Header().Set("Docker-Upload-UUID", uuid)
	w.Header().Set("Range", "0-0")
	w.WriteHeader(netHttp.StatusAccepted)
}

// BlobsUploadsPost handles the POST request for uploading blobs.
//
// There are three modes:
//   - POST then PUT.
//   - Single POST (requires url query param "digest").
//   - Mount blob from other repository.
//
// # Route pattern:
//
//	"POST /v2/{name}/blobs/uploads"
//
// # Path params:
//   - {name}		must be a valid repository name.
//
// # Url query params:
//   - "mount" optional. Requires "from" query param.
//   - "from" optional. Requires "mount" query param.
//   - "digest" optional. Must be a valid digest.
//
// # HTTP status codes:
//   - 201 Created - Blob uploaded successfully.
//   - 202 Accepted - Blob upload accepted but not yet complete.
//   - 400 Bad Request
//   - 401 Unauthorized
//   - 403 Forbidden
//   - 404 Not Found - Blob from other repository not found.
//   - 500 Internal Server Error
func (m *ServeMux) BlobsUploadsPost(
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	username, err := m.cfg.Rbac.GetUsernameFromHttpRequest(r)
	if err, ok := err.(*http.HttpError); ok {
		w.WriteHeader(err.Status)
		return
	}

	// "repo" must be a valid repository name.
	repo := r.PathValue("name")
	if !regexp.MustCompile(registry.RegExpName).MatchString(repo) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// Check if the user can push to the repository.
	if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, netHttp.MethodPost) {
		if username == rbac.AnonymousUsername {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		}
	}

	// Case 1. Mount blob from other repository.
	mount := r.URL.Query().Get("mount")
	from := r.URL.Query().Get("from")
	if mount != "" && from != "" {
		// Check if the user can pull the other repository.
		if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, netHttp.MethodPost) {
			w.WriteHeader(netHttp.StatusForbidden)
			return
		}

		blobsUploadsPostMount(
			m.cfg,
			repo,
			from,
			mount,
			w,
		)
		return
	}

	// Case 2. Single POST request to upload a blob.
	digest := r.URL.Query().Get("digest")
	if _, _, err := d.Parse(digest); err == nil {
		blobsUploadsPostSingle(
			m.cfg,
			repo,
			digest,
			w,
			r,
		)
		return
	}

	// Case 3. POST request to create an upload session.
	blobsUploadsPostThenPut(
		m.cfg,
		repo,
		w,
	)
}

func (m *ServeMux) BlobsUploadsGet(
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	username, err := m.cfg.Rbac.GetUsernameFromHttpRequest(r)
	if err, ok := err.(*http.HttpError); ok {
		w.WriteHeader(err.Status)
		return
	}

	// "repo" must be a valid repository name.
	repo := r.PathValue("name")
	if !regexp.MustCompile(registry.RegExpName).MatchString(repo) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// "uuid" must be a valid UUID.
	uuid := r.PathValue("uuid")
	if u.Validate(uuid) != nil {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// Check if the user can push to the repository.
	if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, netHttp.MethodPost) {
		if username == rbac.AnonymousUsername {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		}
	}

	size, err := m.cfg.Data.BlobsUploadSize(repo, uuid)
	if err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/v2/%s/blobs/uploads/%s", repo, uuid)

	w.Header().Set("Location", location)
	w.Header().Set("Range", fmt.Sprintf("0-%d", size-1))
	w.Header().Set("Docker-Upload-UUID", uuid)
	w.WriteHeader(netHttp.StatusNoContent)
}

// BlobsUploadsPatch handles PATCH requests for blob uploads.
// Supports range uploads.
//
// Route pattern:
//
//	"PATCH /v2/{name}/blobs/uploads/{uuid}"
//
// # Path params:
//   - {name}		must be a valid repository name.
//   - {uuid}		must be a valid [UUID].
//
// # HTTP status codes:
//   - 202 Accepted
//   - 400 Bad Request
//   - 401 Unauthorized
//   - 403 Forbidden
//   - 404 Not Found
//   - 416 Requested range not satisfiable
//   - 500 Internal Server Error
//
// [UUID]: https://www.rfc-editor.org/rfc/rfc4122
func (m *ServeMux) BlobsUploadsPatch(
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	// Validate request Content-Type header.
	if r.Header.Get("Content-Type") != "application/octet-stream" {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	username, err := m.cfg.Rbac.GetUsernameFromHttpRequest(r)
	if err, ok := err.(*http.HttpError); ok {
		w.WriteHeader(err.Status)
		return
	}

	// "repo" must be a valid repository name.
	repo := r.PathValue("name")
	if !regexp.MustCompile(registry.RegExpName).MatchString(repo) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// "uuid" must be a valid UUID.
	uuid := r.PathValue("uuid")
	if u.Validate(uuid) != nil {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// Check if the user can push to the repository.
	if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, netHttp.MethodPatch) {
		if username == rbac.AnonymousUsername {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		}
	}

	size, err := m.cfg.Data.BlobsUploadSize(repo, uuid)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(netHttp.StatusNotFound)
			return
		}

		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	var start int64 = -1
	if rngStart, _, err := http.ParseRequestContentRange(r); err == nil {
		start = rngStart
	}

	// Check if the range is valid.
	if start >= 0 && start != size {
		w.WriteHeader(netHttp.StatusRequestedRangeNotSatisfiable)
		return
	}

	if err := m.cfg.Data.BlobsUploadWrite(repo, uuid, r.Body, start); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(netHttp.StatusNotFound)
			return
		}

		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	// Update the size of the blob.
	size, err = m.cfg.Data.BlobsUploadSize(repo, uuid)
	if err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/v2/%s/blobs/uploads/%s", repo, uuid)

	w.Header().Set("Location", location)
	w.Header().Set("Range", fmt.Sprintf("0-%d", size-1))
	w.Header().Set("Docker-Upload-UUID", uuid)
	w.WriteHeader(netHttp.StatusAccepted)
}

// BlobsUploadsPut handles the PUT request for blobs uploads.
//
// # Route pattern:
//
//	"PUT /v2/{name}/blobs/uploads/{uuid}?digest={digest}"
//
// # Path params:
//   - {name}		must be a valid repository name.
//   - {uuid}		must be a valid [UUID].
//
// # Url query params:
//   - "digest" must be a valid digest.
//
// # HTTP status codes:
//   - 201 Created
//   - 400 Bad Request
//   - 401 Unauthorized
//   - 403 Forbidden
//   - 404 Not Found
//   - 500 Internal Server Error
//
// [UUID]: https://www.rfc-editor.org/rfc/rfc4122
func (m *ServeMux) BlobsUploadsPut(
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	username, err := m.cfg.Rbac.GetUsernameFromHttpRequest(r)
	if err, ok := err.(*http.HttpError); ok {
		w.WriteHeader(err.Status)
		return
	}

	// "repo" must be a valid repository name.
	repo := r.PathValue("name")
	if !regexp.MustCompile(registry.RegExpName).MatchString(repo) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// "uuid" must be a valid UUID.
	uuid := r.PathValue("uuid")
	if u.Validate(uuid) != nil {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// "digest" must be a valid digest.
	digest := r.URL.Query().Get("digest")
	if _, _, err := d.Parse(digest); err != nil {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// Check if the user can push to the repository.
	if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, netHttp.MethodPut) {
		if username == rbac.AnonymousUsername {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		}
	}

	if r.Header.Get("Content-Type") == "application/octet-stream" && r.Header.Get("Content-Length") != "" {
		// Optionally, PUT can upload the last blob chunk data.

		var start int64 = -1
		if rngStart, _, err := http.ParseRequestContentRange(r); err == nil {
			start = rngStart
		}

		if err := m.cfg.Data.BlobsUploadWrite(repo, uuid, r.Body, start); err != nil {
			// Docker expects 404 when the blob does not exist
			if errors.Is(err, fs.ErrNotExist) {
				w.WriteHeader(netHttp.StatusNotFound)
				return
			}

			w.WriteHeader(netHttp.StatusInternalServerError)
			return
		}

		// Then commit the upload.
	}

	if err := m.cfg.Data.BlobsUploadCommit(repo, uuid, digest); err != nil {
		// Docker expects 404 when the blob does not exist
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(netHttp.StatusNotFound)
			return
		}

		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/v2/%s/blobs/%s", repo, digest)

	w.Header().Set("Location", location)
	w.Header().Set("Docker-Content-Digest", digest)
	w.WriteHeader(netHttp.StatusCreated)
}

// BlobsUploadsDelete delete a blob upload in progress.
//
// # Route pattern:
//
//	"DELETE /v2/{name}/blobs/uploads/{uuid}"
//
// # Path params:
//   - {name}		must be a valid repository name.
//   - {uuid}		must be a valid [UUID].
//
// # HTTP status codes:
//   - 204 No Content
//   - 400 Bad Request
//   - 401 Unauthorized
//   - 403 Forbidden
//   - 404 Not Found
//   - 500 Internal Server Error
//
// [UUID]: https://www.rfc-editor.org/rfc/rfc4122
func (m *ServeMux) BlobsUploadsDelete(
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	username, err := m.cfg.Rbac.GetUsernameFromHttpRequest(r)
	if err, ok := err.(*http.HttpError); ok {
		w.WriteHeader(err.Status)
		return
	}

	// "repo" must be a valid repository name.
	repo := r.PathValue("name")
	if !regexp.MustCompile(registry.RegExpName).MatchString(repo) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// "uuid" must be a valid UUID.
	uuid := r.PathValue("uuid")
	if u.Validate(uuid) != nil {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// Check if the user can delete blobs from the repository.
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

	if err := m.cfg.Data.BlobsUploadCancel(repo, uuid); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(netHttp.StatusNotFound)
			return
		}
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	w.WriteHeader(netHttp.StatusNoContent)
}
