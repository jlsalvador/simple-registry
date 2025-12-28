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
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"syscall"
	"time"

	d "github.com/jlsalvador/simple-registry/pkg/digest"
)

func (s *FilesystemDataStorage) BlobsGet(repo, digest string) (r io.ReadCloser, size int64, err error) {
	algo, hash, err := d.Parse(digest)
	if err != nil {
		return nil, -1, err
	}

	if len(hash) < 2 {
		return nil, -1, ErrHashShort
	}

	if repo != "" {
		// Check repository link
		linkPath := filepath.Join(
			s.base,
			"repositories",
			repo,
			"_layers",
			algo,
			hash,
			"link",
		)

		linkData, err := os.ReadFile(linkPath)
		if err != nil {
			return nil, -1, fmt.Errorf("cannot read link %s: %w", linkPath, err)
		}

		// Verify link content matches requested digest
		if string(linkData) != digest {
			return nil, -1, fmt.Errorf("repository link mismatch: expected %s, got %s", digest, string(linkData))
		}
	}

	// Open the actual blob
	blobPath := filepath.Join(s.base, "blobs", algo, hash[0:2], hash)
	f, err := os.Open(blobPath)
	if err != nil {
		return nil, -1, fmt.Errorf("cannot open blob %s: %w", blobPath, err)
	}

	// Get the size of the blob
	stat, err := f.Stat()
	if err != nil {
		return nil, -1, err
	}
	size = stat.Size()

	return f, size, nil
}

func (s *FilesystemDataStorage) BlobsDelete(repo, digest string) error {
	algo, hash, err := d.Parse(digest)
	if err != nil {
		return err
	}

	if len(hash) < 2 {
		return ErrHashShort
	}

	// Link path
	linkPath := filepath.Join(
		s.base,
		"repositories",
		repo,
		"_layers",
		algo,
		hash,
		"link",
	)

	if err := os.RemoveAll(filepath.Dir(linkPath)); err != nil {
		return err
	}

	//TODO: Garbage collect unused blobs

	return nil
}

func (s *FilesystemDataStorage) BlobsList() (digests iter.Seq[string], err error) {
	blobsDir := filepath.Join(
		s.base,
		"blobs",
	)

	return func(yield func(string) bool) {
		algos, _ := os.ReadDir(blobsDir)
		for _, algo := range algos {
			if !algo.IsDir() {
				continue
			}

			dir := filepath.Join(blobsDir, algo.Name())
			hashPrefixes, _ := os.ReadDir(dir)

			for _, hashPrefix := range hashPrefixes {
				if !hashPrefix.IsDir() {
					continue
				}

				dir := filepath.Join(blobsDir, algo.Name(), hashPrefix.Name())
				dgsts, _ := os.ReadDir(dir)

				for _, e := range dgsts {
					if e.IsDir() {
						continue
					}

					digest := algo.Name() + ":" + e.Name()
					if !yield(digest) {
						return
					}
				}
			}
		}
	}, nil
}

func (s *FilesystemDataStorage) BlobLastAccess(digest string) (lastAccess time.Time, err error) {
	algo, hash, err := d.Parse(digest)
	if err != nil {
		return time.Now(), err
	}

	if len(hash) < 2 {
		return time.Now(), ErrDigestMismatch
	}

	blobPath := filepath.Join(s.base, "blobs", algo, hash[0:2], hash)

	fi, err := os.Stat(blobPath)
	if err != nil {
		return time.Now(), err
	}

	fis, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return time.Now(), errors.New("cannot fetch atime from filesystem")
	}

	return time.Unix(fis.Atim.Sec, fis.Atim.Nsec), nil
}
