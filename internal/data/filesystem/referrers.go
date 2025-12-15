package filesystem

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jlsalvador/simple-registry/pkg/digest"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func (s *FilesystemDataStorage) ReferrersGet(repo, dgst string) (r io.ReadCloser, size int64, err error) {
	algo, hash, err := digest.Parse(dgst)
	if err != nil {
		return nil, -1, err
	}

	refDir := filepath.Join(
		s.base, "repositories", repo, "_manifests",
		"referrers", algo, hash,
	)

	entries, err := os.ReadDir(refDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Directory does not exist.
			// Return an empty list instead of failing.
			entries = []os.DirEntry{}
		} else {
			return nil, -1, err
		}
	}

	type descriptor struct {
		MediaType   string `json:"mediaType"`
		Digest      string `json:"digest"`
		Size        int64  `json:"size"`
		Annotations map[string]string
	}

	manifests := []descriptor{}

	for _, e := range entries {
		referrerDigest := e.Name()
		fi, err := e.Info()
		if err != nil {
			continue
		}

		referrerAlgo, referrerHash, err := digest.Parse(referrerDigest)
		if err != nil {
			continue
		}

		blobName := filepath.Join(s.base, "blobs", referrerAlgo, referrerHash[:2], referrerHash)
		blob, err := os.OpenFile(blobName, os.O_RDONLY, 0o644)
		if err != nil {
			continue
		}
		defer blob.Close()

		manifest := &registry.Manifest{}
		if err := json.NewDecoder(blob).Decode(manifest); err != nil {
			continue
		}

		manifests = append(manifests, descriptor{
			MediaType:   manifest.MediaType,
			Digest:      referrerDigest,
			Size:        fi.Size(),
			Annotations: manifest.Annotations,
		})
	}

	index := map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.index.v1+json",
		"manifests":     manifests,
	}

	data, err := json.Marshal(index)
	if err != nil {
		return nil, -1, err
	}

	f := io.NopCloser(bytes.NewReader(data))

	return f, int64(len(data)), nil
}
