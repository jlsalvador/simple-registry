package garbagecollect

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func getReferrencedBlobs(cfg config.Config) ([]string, error) {
	refBlobs := []string{}

	//FIXME: needs to load children manifests too!

	repositories, err := cfg.Data.RepositoriesList()
	if err != nil {
		return nil, err
	}

	for _, repo := range repositories {
		manifests, err := cfg.Data.ManifestsList(repo)
		if err != nil {
			return nil, err
		}

		for digest := range manifests {
			r, _, _, err := cfg.Data.ManifestGet(repo, digest)
			if err != nil {
				return nil, err
			}
			defer r.Close()

			header := &struct {
				MediaType string `json:"mediaType"`
			}{}
			if err := json.NewDecoder(r).Decode(header); err != nil {
				return nil, err
			}

			// Re-read (simplest way).
			r2, _, _, err := cfg.Data.ManifestGet(repo, digest)
			if err != nil {
				return nil, err
			}
			defer r2.Close()

			switch header.MediaType {
			case registry.MediaTypeOCIImageManifest,
				registry.MediaTypeDockerImageManifest:

				manifest := &registry.ImageManifest{}
				if err := json.NewDecoder(r2).Decode(manifest); err != nil {
					return nil, err
				}

				refBlobs = append(refBlobs, manifest.Config.Digest)

				for _, layer := range manifest.Layers {
					if !slices.Contains(refBlobs, layer.Digest) {
						refBlobs = append(refBlobs, layer.Digest)
					}
				}

			case registry.MediaTypeOCIImageIndex,
				registry.MediaTypeDockerManifestList:

				manifest := &registry.ImageIndexManifest{}
				if err := json.NewDecoder(r2).Decode(manifest); err != nil {
					return nil, err
				}

				refBlobs = append(refBlobs, digest)

				for _, m := range manifest.Manifests {
					if !slices.Contains(refBlobs, m.Digest) {
						refBlobs = append(refBlobs, m.Digest)
					}
				}

			default:
				return nil, fmt.Errorf("unsupported media type: %s", header.MediaType)
			}

		}
	}

	return refBlobs, nil
}

func garbageCollect(cfg config.Config) error {
	refBlobs, err := getReferrencedBlobs(cfg)
	if err != nil {
		return err
	}

	blobs, err := cfg.Data.BlobsList()
	if err != nil {
		return err
	}
	for blob := range blobs {
		if !slices.Contains(refBlobs, blob) {
			fmt.Printf("Removing unused blob: %s\n", blob)
		}
	}

	return nil
}
