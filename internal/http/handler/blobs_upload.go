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
	"net/http"
	"regexp"

	"github.com/jlsalvador/simple-registry/internal/config"
	httpInternal "github.com/jlsalvador/simple-registry/internal/http"
	"github.com/jlsalvador/simple-registry/internal/rbac"
	d "github.com/jlsalvador/simple-registry/pkg/digest"
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
	w http.ResponseWriter,
) {
	f, _, err := cfg.Data.BlobsGet(from, mount)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	uuid, err := cfg.Data.BlobsUploadCreate(repo)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := cfg.Data.BlobsUploadWrite(repo, uuid, f, -1); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if err := cfg.Data.BlobsUploadCommit(repo, uuid, mount); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/v2/%s/blobs/%s", repo, mount)
	w.Header().Set("Location", location)
	w.Header().Set("Docker-Content-Digest", mount)
	w.WriteHeader(http.StatusCreated)
}

// blobsUploadsPostSingle uploads blob in a single POST.
// ACL must be already checked.
func blobsUploadsPostSingle(
	cfg config.Config,
	repo string,
	digest string,
	w http.ResponseWriter,
	r *http.Request,
) {
	uuid, err := cfg.Data.BlobsUploadCreate(repo)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := cfg.Data.BlobsUploadWrite(repo, uuid, r.Body, 0); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := cfg.Data.BlobsUploadCommit(repo, uuid, digest); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/v2/%s/blobs/%s", repo, digest)
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

// blobsUploadsPostThenPut creates upload blob session.
// ACL must be already checked.
func blobsUploadsPostThenPut(
	cfg config.Config,
	repo string,
	w http.ResponseWriter,
) {
	uuid, err := cfg.Data.BlobsUploadCreate(repo)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/v2/%s/blobs/uploads/%s", repo, uuid)

	w.Header().Set("Location", location)
	w.Header().Set("Docker-Upload-UUID", uuid)
	w.Header().Set("Range", "0-0")
	w.WriteHeader(http.StatusAccepted)
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

	// Check if the user can push to repository.
	if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, rbac.ActionPush) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// Case 1. Mount blob from other repository.
	mount := r.URL.Query().Get("mount")
	from := r.URL.Query().Get("from")
	if mount != "" && from != "" {
		// Check if the user can pull the other repository.
		if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, rbac.ActionPush) {
			w.WriteHeader(http.StatusForbidden)
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

	// Check if the user can push to the repository.
	if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, rbac.ActionPush) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// "uuid" must be a valid UUID.
	uuid := r.PathValue("uuid")
	if u.Validate(uuid) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	size, err := m.cfg.Data.BlobsUploadSize(repo, uuid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/v2/%s/blobs/uploads/%s", repo, uuid)

	w.Header().Set("Location", location)
	w.Header().Set("Range", fmt.Sprintf("0-%d", size-1))
	w.Header().Set("Docker-Upload-UUID", uuid)
	w.WriteHeader(http.StatusNoContent)
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
	w http.ResponseWriter,
	r *http.Request,
) {
	// Validate request
	if r.Header.Get("Content-Type") != "application/octet-stream" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var start int64 = -1

	if rngStart, _, err := httpInternal.ParseRequestContentRange(r); err == nil {
		start = rngStart
	}

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

	// Check if the user can push to the repository.
	if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, rbac.ActionPush) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// "uuid" must be a valid UUID.
	uuid := r.PathValue("uuid")
	if u.Validate(uuid) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	size, err := m.cfg.Data.BlobsUploadSize(repo, uuid)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Check if the range is valid.
	if start >= 0 && start != size {
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	if err := m.cfg.Data.BlobsUploadWrite(repo, uuid, r.Body, start); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Update the size of the blob.
	size, err = m.cfg.Data.BlobsUploadSize(repo, uuid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/v2/%s/blobs/uploads/%s", repo, uuid)

	w.Header().Set("Location", location)
	w.Header().Set("Range", fmt.Sprintf("0-%d", size-1))
	w.Header().Set("Docker-Upload-UUID", uuid)
	w.WriteHeader(http.StatusAccepted)
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

	// "uuid" must be a valid UUID.
	uuid := r.PathValue("uuid")
	if u.Validate(uuid) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// "digest" must be a valid digest.
	digest := r.URL.Query().Get("digest")
	if _, _, err := d.Parse(digest); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if the user can push to the repository.
	if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, rbac.ActionPush) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if r.Header.Get("Content-Type") == "application/octet-stream" && r.Header.Get("Content-Length") != "" {
		// Optionally, PUT can upload the last blob chunk data.

		var start int64 = -1
		if rngStart, _, err := httpInternal.ParseRequestContentRange(r); err == nil {
			start = rngStart
		}

		if err := m.cfg.Data.BlobsUploadWrite(repo, uuid, r.Body, start); err != nil {
			// Docker expects 404 when the blob does not exist
			if errors.Is(err, fs.ErrNotExist) {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Then commit the upload.
	}

	if err := m.cfg.Data.BlobsUploadCommit(repo, uuid, digest); err != nil {
		// Docker expects 404 when the blob does not exist
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	location := fmt.Sprintf("/v2/%s/blobs/%s", repo, digest)

	w.Header().Set("Location", location)
	w.Header().Set("Docker-Content-Digest", digest)
	w.WriteHeader(http.StatusCreated)
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

	// "uuid" must be a valid UUID.
	uuid := r.PathValue("uuid")
	if u.Validate(uuid) != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if the user can push blobs into the repository.
	if !m.cfg.Rbac.IsAllowed(username, "blobs", repo, rbac.ActionPush) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if err := m.cfg.Data.BlobsUploadCancel(repo, uuid); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
