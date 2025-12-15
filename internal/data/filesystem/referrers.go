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
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int64  `json:"size"`
	}

	manifests := []descriptor{}

	for _, e := range entries {
		referrerDigest := e.Name()

		referrerAlgo, referrerHash, err := digest.Parse(referrerDigest)
		if err != nil {
			continue
		}

		blob := filepath.Join(s.base, "blobs", referrerAlgo, referrerHash[:2], referrerHash)
		st, err := os.Stat(blob)
		if err != nil {
			continue
		}

		manifests = append(manifests, descriptor{
			MediaType: "application/vnd.oci.image.manifest.v1+json",
			Digest:    referrerDigest,
			Size:      st.Size(),
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
