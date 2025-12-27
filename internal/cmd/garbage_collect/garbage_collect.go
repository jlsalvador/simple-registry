package garbagecollect

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func garbageCollect(cfg config.Config) error {
	refBlobs := []string{}

	repositories, err := cfg.Data.RepositoriesList()
	if err != nil {
		return err
	}

	for _, repo := range repositories {
		manifests, err := cfg.Data.ManifestsList(repo)
		if err != nil {
			return err
		}

		for digest := range manifests {
			r, _, _, err := cfg.Data.ManifestGet(repo, digest)
			if err != nil {
				return err
			}
			defer r.Close()

			manifest := &registry.ImageManifest{}
			if err := json.NewDecoder(r).Decode(manifest); err != nil {
				return err
			}

			for _, layer := range manifest.Layers {
				if !slices.Contains(refBlobs, layer.Digest) {
					refBlobs = append(refBlobs, layer.Digest)
				}
			}
		}
	}

	for _, refBlob := range refBlobs {
		fmt.Println(refBlob)
	}

	return nil
}
