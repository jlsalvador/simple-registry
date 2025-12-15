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

package filesystem

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	d "github.com/jlsalvador/simple-registry/pkg/digest"
	"github.com/jlsalvador/simple-registry/pkg/registry"

	u "github.com/google/uuid"
)

// BlobsUploadCreate creates a new blob upload session for the given
// repository, and returns the upload uuid.
func (s *FilesystemDataStorage) BlobsUploadCreate(repo string) (uuid string, err error) {
	if !regexp.MustCompile(registry.RegExpName).MatchString(repo) {
		return "", ErrRepoInvalid
	}

	uuid = u.NewString()
	uploadDir := filepath.Join(s.base, "repositories", repo, "_uploads", uuid)

	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return "", err
	}

	// Store metadata like "startedat".
	startedAt := time.Now().UTC().Format(time.RFC3339Nano)
	if err := os.WriteFile(filepath.Join(uploadDir, "startedat"), []byte(startedAt), 0o644); err != nil {
		return "", err
	}

	// Create empty data file (appendable).
	f, err := os.OpenFile(filepath.Join(uploadDir, "data"), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return "", err
	}
	f.Close()

	return uuid, nil
}

// BlobsUploadCancel cancels a blob upload in progress.
func (s *FilesystemDataStorage) BlobsUploadCancel(repo, uuid string) error {
	if !regexp.MustCompile(registry.RegExpName).MatchString(repo) {
		return ErrRepoInvalid
	}
	if u.Validate(uuid) != nil {
		return ErrUUIDInvalid
	}

	uploadDir := filepath.Join(s.base, "repositories", repo, "_uploads", uuid)
	return os.RemoveAll(uploadDir)
}

// BlobsUploadWrite writes data to an blob upload in progress.
//
// If "start" is less than 0, the data will be appended to the end of the
// temporal blob data file.
func (s *FilesystemDataStorage) BlobsUploadWrite(repo, uuid string, r io.Reader, start int64) error {
	if !regexp.MustCompile(registry.RegExpName).MatchString(repo) {
		return ErrRepoInvalid
	}
	if u.Validate(uuid) != nil {
		return ErrUUIDInvalid
	}

	uploadFile := filepath.Join(s.base, "repositories", repo, "_uploads", uuid, "data")

	f, err := os.OpenFile(uploadFile, os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if start < 0 {
		_, err = f.Seek(0, io.SeekEnd)
	} else {
		_, err = f.Seek(start, io.SeekStart)
	}
	if err != nil {
		return err
	}

	_, err = io.Copy(f, r)
	return err
}

func blobsUploadCommit(
	s *FilesystemDataStorage,
	repo,
	uuid,
	algo,
	hash string,
) error {
	if !regexp.MustCompile(registry.RegExpName).MatchString(repo) {
		return ErrRepoInvalid
	}
	if u.Validate(uuid) != nil {
		return ErrUUIDInvalid
	}

	uploadFile := filepath.Join(s.base, "repositories", repo, "_uploads", uuid, "data")
	f, err := os.OpenFile(uploadFile, os.O_RDONLY, 0o644)
	if err != nil {
		return err
	}

	// Calculate the digest of the uploaded file.
	hasher, err := d.NewHasher(algo)
	if err != nil {
		f.Close()
		return err
	}
	if _, err := io.Copy(hasher, f); err != nil {
		f.Close()
		return err
	}
	f.Close()

	// Check if the uploaded data matches the expected digest.
	if hasher.GetHashAsString() != hash {
		return ErrDigestMismatch
	}

	// Replace existing blob atomically.
	blobPath := filepath.Join(s.base, "blobs", algo, hash[0:2], hash)
	if err := os.MkdirAll(filepath.Dir(blobPath), 0o755); err != nil {
		return err
	}
	_ = os.Remove(blobPath) // ignore error
	if err := os.Rename(uploadFile, blobPath); err != nil {
		return err
	}

	// Delete temporal upload.
	return os.RemoveAll(filepath.Dir(uploadFile))
}

// BlobsUploadCommit commits an blob upload in progress as a layer.
//
// After check uploaded data hash, the temporal blob's data file will be moved
// to the final location and the repository layer's link will be created.
func (s *FilesystemDataStorage) BlobsUploadCommit(repo, uuid, digest string) error {
	algo, hash, err := d.Parse(digest)
	if err != nil {
		return err
	}
	if len(hash) < 2 {
		return ErrHashShort
	}

	if err := blobsUploadCommit(s, repo, uuid, algo, hash); err != nil {
		return err
	}

	// Write repository link:
	// repositories/<repo>/_layers/<algo>/<hex>/link
	linkPath := filepath.Join(
		s.base,
		"repositories",
		repo,
		"_layers",
		algo,
		hash,
		"link",
	)
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(linkPath, []byte(digest), 0o644)
}

func (s *FilesystemDataStorage) BlobsUploadSize(repo, uuid string) (size int64, err error) {
	if !regexp.MustCompile(registry.RegExpName).MatchString(repo) {
		return -1, ErrRepoInvalid
	}
	if u.Validate(uuid) != nil {
		return -1, ErrUUIDInvalid
	}

	uploadFile := filepath.Join(s.base, "repositories", repo, "_uploads", uuid, "data")
	fi, err := os.Stat(uploadFile)
	if err != nil {
		return -1, err
	}

	return fi.Size(), nil
}
