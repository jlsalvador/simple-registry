package filesystem

import (
	"bytes"
	"encoding/json"
	"io"
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
		return nil, -1, err
	}

	type descriptor struct {
		MediaType string `json:"mediaType"`
		Digest    string `json:"digest"`
		Size      int64  `json:"size"`
	}

	var manifests []descriptor

	for _, e := range entries {
		link := filepath.Join(refDir, e.Name(), "link")
		b, err := os.ReadFile(link)
		if err != nil {
			continue
		}

		dgst := string(b)
		algo, hash, _ := digest.Parse(dgst)

		blob := filepath.Join(s.base, "blobs", algo, hash[:2], hash)
		st, err := os.Stat(blob)
		if err != nil {
			continue
		}

		manifests = append(manifests, descriptor{
			MediaType: "application/vnd.oci.image.manifest.v1+json",
			Digest:    dgst,
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
