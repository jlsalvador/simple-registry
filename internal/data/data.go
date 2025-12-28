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

package data

import (
	"io"
	"iter"
	"time"
)

type DataStorage interface {
	// BlobsGet retrieves a blob from the storage.
	//
	// `repo` is the name of the repository, and could be empty.
	BlobsGet(repo, digest string) (r io.ReadCloser, size int64, err error)
	BlobsDelete(repo, digest string) error
	BlobsList() (digests iter.Seq[string], err error)
	BlobLastAccess(digest string) (lastAccess time.Time, err error)

	BlobsUploadCreate(repo string) (uuid string, err error)
	BlobsUploadCancel(repo, uuid string) error
	BlobsUploadWrite(repo, uuid string, r io.Reader, start int64) error
	BlobsUploadCommit(repo, uuid, digest string) error
	BlobsUploadSize(repo, uuid string) (size int64, err error)

	ManifestPut(repo, reference string, r io.Reader) (digest string, err error)
	ManifestGet(repo, reference string) (r io.ReadCloser, size int64, digest string, err error)
	ManifestDelete(repo, reference string) error
	ManifestsList(repo string) (digests iter.Seq[string], err error)

	TagsList(repo string) ([]string, error)

	RepositoriesList() ([]string, error)

	// ReferrersGet returns an iter of referrer digests for the given manifest
	// digest.
	ReferrersGet(
		repo,
		manifestDigest string,
	) (digests iter.Seq[string], err error)
}
