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

type genericManifest struct {
	MediaType    string  `json:"mediaType"`
	ArtifactType *string `json:"artifactType,omitempty"`
	Config       *struct {
		MediaType string `json:"mediaType"`
	} `json:"config,omitempty"`
	Annotations map[string]string `json:"annotations"`
}

func isSkipableManifest(artifactType string, blobManifest genericManifest) bool {
	if artifactType == "" {
		return false
	}

	if blobManifest.ArtifactType != nil {
		// Modern artifact
		if *blobManifest.ArtifactType != artifactType {
			return true
		}
	} else {
		// Legacy artifact
		if blobManifest.MediaType != "application/vnd.oci.image.manifest.v1+json" {
			return true
		}

		if blobManifest.Config == nil || blobManifest.Config.MediaType != artifactType {
			return true
		}
	}

	return false
}

func (s *FilesystemDataStorage) ReferrersGet(repo, dgst, artifactType string) (r io.ReadCloser, size int64, err error) {
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

	index := registry.NewImageIndexManifest()

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

		// Blob manifest with legacy support.
		blobManifest := genericManifest{}

		if err := json.NewDecoder(blob).Decode(&blobManifest); err != nil {
			continue
		}

		if isSkipableManifest(artifactType, blobManifest) {
			continue
		}

		index.Manifests = append(index.Manifests, registry.DescriptorManifest{
			MediaType:   blobManifest.MediaType,
			Digest:      referrerDigest,
			Size:        fi.Size(),
			Annotations: blobManifest.Annotations,
		})
	}

	data, err := json.Marshal(index)
	if err != nil {
		return nil, -1, err
	}

	f := io.NopCloser(bytes.NewReader(data))

	return f, int64(len(data)), nil
}
