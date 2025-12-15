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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jlsalvador/simple-registry/pkg/digest"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

const manifestAlgo = "sha256"

// indexReferrer verifies if the manifest has a subject, if it so, create the refferers.
func (s *FilesystemDataStorage) indexReferrer(repo, referrerDigest string, manifestBytes []byte) error {
	var manifest registry.Manifest

	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil // Ignore invalid OCI 1.1.
	}

	if manifest.Subject == nil {
		return nil
	}

	subjectDigest := manifest.Subject.Digest

	algo, hash, err := digest.Parse(subjectDigest)
	if err != nil {
		return err
	}

	linkPath := filepath.Join(
		s.base, "repositories", repo, "_manifests",
		"referrers", algo, hash, referrerDigest,
		"link",
	)

	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		return fmt.Errorf("cannot create referrer directory %s: %w", filepath.Dir(linkPath), err)
	}

	if err := os.WriteFile(linkPath, []byte(subjectDigest), 0o644); err != nil {
		return fmt.Errorf("cannot write file referrer %s: %w", linkPath, err)
	}

	return nil
}

// ManifestPut stores a manifest identified by "reference" (either a tag or a digest)
// into the repository.
func (s *FilesystemDataStorage) ManifestPut(repo, reference string, r io.Reader) (dgst string, err error) {
	if !regexp.MustCompile("^" + registry.RegExpName + "$").MatchString(repo) {
		return "", ErrRepoInvalid
	}

	hasher, err := digest.NewHasher(manifestAlgo)
	if err != nil {
		return "", err
	}

	data := bytes.NewBuffer([]byte{})
	m := io.MultiWriter(hasher, data)
	if _, err := io.Copy(m, r); err != nil {
		return "", err
	}

	hash := hasher.GetHashAsString()
	dgst = manifestAlgo + ":" + hash

	// Store manifest blob.
	uuid, err := s.BlobsUploadCreate(repo)
	if err != nil {
		return "", err
	}
	blob := bytes.NewReader(data.Bytes())
	if err = s.BlobsUploadWrite(repo, uuid, blob, -1); err != nil {
		return "", err
	}
	if err := s.BlobsUploadCommit(repo, uuid, dgst); err != nil {
		return "", err
	}

	// Create revision link
	revisionLink := filepath.Join(
		s.base, "repositories", repo, "_manifests",
		"revisions", manifestAlgo, hash, "link",
	)
	if err := os.MkdirAll(filepath.Dir(revisionLink), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(revisionLink, []byte(dgst), 0o644); err != nil {
		return "", err
	}

	// If reference is a tag, update tag link
	if regexp.MustCompile("^" + registry.RegExpTag + "$").MatchString(reference) {
		tagLink := filepath.Join(
			s.base, "repositories", repo, "_manifests",
			"tags", reference, "current", "link",
		)
		if err := os.MkdirAll(filepath.Dir(tagLink), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(tagLink, []byte(dgst), 0o644); err != nil {
			return "", err
		}
	}

	// Index the manifest referrer.
	s.indexReferrer(repo, dgst, data.Bytes())

	return dgst, nil
}

// ManifestGet retrieves a manifest using either a tag or a digest.
func (s *FilesystemDataStorage) ManifestGet(repo, reference string) (r io.ReadCloser, size int64, err error) {
	if !regexp.MustCompile("^" + registry.RegExpName + "$").MatchString(repo) {
		return nil, -1, ErrRepoInvalid
	}

	algo, hash, err := digest.Parse(reference)
	if err == nil {
		// If reference is a digest, use it directly.
	} else if regexp.MustCompile("^" + registry.RegExpTag + "$").MatchString(reference) {
		// If reference is a tag, resolve tag to digest.
		tagLink := filepath.Join(
			s.base, "repositories", repo, "_manifests",
			"tags", reference, "current", "link",
		)
		b, err := os.ReadFile(tagLink)
		if err != nil {
			return nil, -1, err
		}
		algo, hash, err = digest.Parse(string(b))
		if err != nil {
			return nil, -1, err
		}
	}

	// Open the actual blob manifest.
	blobPath := filepath.Join(s.base, "blobs", algo, hash[0:2], hash)
	f, err := os.Open(blobPath)
	if err != nil {
		return nil, -1, fmt.Errorf("cannot open blob %s: %w", blobPath, err)
	}

	// Get the size of the blob.
	stat, err := f.Stat()
	if err != nil {
		return nil, -1, err
	}
	size = stat.Size()

	return f, size, nil
}

func (s *FilesystemDataStorage) ManifestDelete(repo, reference string) error {
	if !regexp.MustCompile("^" + registry.RegExpName + "$").MatchString(repo) {
		return ErrRepoInvalid
	}

	// Case 1: reference is a tag.
	if regexp.MustCompile("^" + registry.RegExpTag + "$").MatchString(reference) {
		tagDir := filepath.Join(
			s.base, "repositories", repo, "_manifests",
			"tags", reference,
		)

		// Docker returns 404 if tag does not exist
		if _, err := os.Stat(tagDir); err != nil {
			return err
		}

		return os.RemoveAll(tagDir)
	}

	// Case 2: reference must be a digest.
	algo, hash, err := digest.Parse(reference)
	if err != nil {
		return err
	}

	revisionDir := filepath.Join(
		s.base, "repositories", repo, "_manifests",
		"revisions", algo, hash,
	)

	// Docker returns 404 if revision does not exist
	if _, err := os.Stat(revisionDir); err != nil {
		return err
	}

	// Remove revision link
	if err := os.RemoveAll(revisionDir); err != nil {
		return err
	}

	//TODO: Garbage collection of unused revisions and tags.

	return nil
}
