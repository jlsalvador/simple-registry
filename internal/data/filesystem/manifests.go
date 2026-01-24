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
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"time"

	"github.com/jlsalvador/simple-registry/internal/data"
	pkgDigest "github.com/jlsalvador/simple-registry/pkg/digest"
	"github.com/jlsalvador/simple-registry/pkg/mapset"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

const manifestAlgo = "sha256"

// indexReferrer verifies if the manifest has a subject, if it so, create the refferers.
func (s *FilesystemDataStorage) indexReferrer(repo, referrerDigest string, manifestBytes []byte) error {
	var manifest registry.ImageManifest

	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil // Ignore invalid OCI 1.1.
	}

	if manifest.Subject == nil {
		return nil
	}

	subjectDigest := manifest.Subject.Digest

	algo, hash, err := pkgDigest.Parse(subjectDigest)
	if err != nil {
		return err
	}

	referrerDir := filepath.Join(
		s.base, "repositories", repo, "_manifests",
		"referrers", algo, hash, referrerDigest,
	)

	if err := os.MkdirAll(referrerDir, 0o755); err != nil {
		return fmt.Errorf("cannot create referrer directory %s: %w", referrerDir, err)
	}

	linkPath := filepath.Join(
		referrerDir, "link",
	)

	if err := os.WriteFile(linkPath, []byte(subjectDigest), 0o644); err != nil {
		return fmt.Errorf("cannot write referrer file %s: %w", linkPath, err)
	}

	return nil
}

// ManifestPut stores a manifest identified by "reference" (either a tag or a digest)
// into the repository.
func (s *FilesystemDataStorage) ManifestPut(repo, reference string, r io.Reader) (dgst string, err error) {
	if !registry.RegExprName.MatchString(repo) {
		return "", data.ErrRepoInvalid
	}

	hasher, err := pkgDigest.NewHasher(manifestAlgo)
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
	if registry.RegExprTag.MatchString(reference) {
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
func (s *FilesystemDataStorage) ManifestGet(repo, reference string) (
	r io.ReadCloser,
	size int64,
	digest string,
	err error,
) {
	if !registry.RegExprName.MatchString(repo) {
		return nil, -1, "", data.ErrRepoInvalid
	}

	algo, hash, err := pkgDigest.Parse(reference)
	if err == nil {
		// If reference is a digest, use it directly.
	} else if registry.RegExprTag.MatchString(reference) {
		// If reference is a tag, resolve tag to digest.
		tagLink := filepath.Join(
			s.base, "repositories", repo, "_manifests",
			"tags", reference, "current", "link",
		)
		b, err := os.ReadFile(tagLink)
		if err != nil {
			return nil, -1, "", err
		}
		algo, hash, err = pkgDigest.Parse(string(b))
		if err != nil {
			return nil, -1, "", err
		}
	}

	// Open the actual blob manifest.
	blobPath := filepath.Join(s.base, "blobs", algo, hash[0:2], hash)
	f, err := os.Open(blobPath)
	if err != nil {
		return nil, -1, "", fmt.Errorf("cannot open blob %s: %w", blobPath, err)
	}

	// Get the size of the blob.
	stat, err := f.Stat()
	if err != nil {
		return nil, -1, "", err
	}
	size = stat.Size()

	digest = algo + ":" + hash

	return f, size, digest, nil
}

func (s *FilesystemDataStorage) ManifestDelete(repo, reference string) error {
	if !registry.RegExprName.MatchString(repo) {
		return data.ErrRepoInvalid
	}

	// Case 1: reference is a tag.
	if registry.RegExprTag.MatchString(reference) {
		tagDir := filepath.Join(
			s.base, "repositories", repo, "_manifests",
			"tags", reference,
		)

		if _, err := os.Stat(tagDir); err != nil {
			return err
		}

		return os.RemoveAll(tagDir)
	}

	// Case 2: reference must be a digest.

	// Delete the referrers.
	r, _, _, err := s.ManifestGet(repo, reference)
	if err == nil {
		if data, err := io.ReadAll(r); err == nil {
			var m registry.ImageManifest
			if json.Unmarshal(data, &m) == nil && m.Subject != nil {
				// It is a referrer, so remove the referrer directory.
				subjAlgo, subjHash, _ := pkgDigest.Parse(m.Subject.Digest)
				refDir := filepath.Join(
					s.base, "repositories", repo, "_manifests",
					"referrers", subjAlgo, subjHash, reference,
				)
				os.RemoveAll(refDir)
			}
		}
		r.Close()
	}

	// Delete the revision.
	algo, hash, err := pkgDigest.Parse(reference)
	if err != nil {
		return errors.Join(data.ErrDigestInvalid, err)
	}

	revisionDir := filepath.Join(
		s.base, "repositories", repo, "_manifests",
		"revisions", algo, hash,
	)

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

func (s *FilesystemDataStorage) ManifestsList(repo string) (digests iter.Seq[string], err error) {
	repoPath := filepath.Join(
		s.base,
		"repositories",
		repo,
		"_manifests",
	)

	digestsSet := mapset.NewMapSet[string]()

	// 1. Revisions
	revisions := filepath.Join(repoPath, "revisions")
	algos, _ := os.ReadDir(revisions)
	for _, algo := range algos {
		if !algo.IsDir() {
			continue
		}

		dir := filepath.Join(revisions, algo.Name())
		entries, _ := os.ReadDir(dir)

		for _, e := range entries {
			if !e.IsDir() {
				continue
			}

			digest := algo.Name() + ":" + e.Name()
			digestsSet.Add(digest)
		}
	}

	// 2. Referrers
	referrers := filepath.Join(repoPath, "referrers")
	algos, _ = os.ReadDir(referrers)
	for _, algo := range algos {
		if !algo.IsDir() {
			continue
		}

		algoDir := filepath.Join(referrers, algo.Name())
		subjects, _ := os.ReadDir(algoDir)

		for _, subj := range subjects {
			if !subj.IsDir() {
				continue
			}

			subjDir := filepath.Join(algoDir, subj.Name())
			refs, _ := os.ReadDir(subjDir)

			for _, ref := range refs {
				if !ref.IsDir() {
					continue
				}

				digest := ref.Name() // filename has the algo prefix.
				digestsSet.Add(digest)
			}
		}
	}

	return func(yield func(string) bool) {
		for digest := range digestsSet {
			if !yield(digest) {
				return
			}
		}
	}, nil
}

func (s *FilesystemDataStorage) ManifestLastAccess(digest string) (lastAccess time.Time, err error) {
	// Our manifests are stored as blobs, so return its blob last access time.
	return s.BlobLastAccess(digest)
}
