package garbagecollect

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func markManifest(
	cfg config.Config,
	repo string,
	digest string,
	seenManifests digestSet,
	seenBlobs digestSet,
) error {
	if seenManifests.has(digest) {
		return nil
	}
	seenManifests.add(digest)

	r, _, _, err := cfg.Data.ManifestGet(repo, digest)
	if err != nil {
		return err
	}
	defer r.Close()

	var raw map[string]any
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return err
	}

	mediaType, _ := raw["mediaType"].(string)

	switch mediaType {

	case registry.MediaTypeOCIImageManifest,
		registry.MediaTypeDockerImageManifest:

		data, err := json.Marshal(raw)
		if err != nil {
			return err
		}

		var manifest registry.ImageManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			return err
		}

		for _, layer := range manifest.Layers {
			seenBlobs.add(layer.Digest)
		}

	case registry.MediaTypeOCIImageIndex,
		registry.MediaTypeDockerManifestList:

		data, err := json.Marshal(raw)
		if err != nil {
			return err
		}

		var index registry.ImageIndexManifest
		if err := json.Unmarshal(data, &index); err != nil {
			return err
		}

		for _, m := range index.Manifests {
			if err := markManifest(cfg, repo, m.Digest, seenManifests, seenBlobs); err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("unsupported media type: %s", mediaType)
	}

	return nil
}

func collectReferencedBlobs(cfg config.Config) (
	seenBlobs digestSet,
	seenManifests digestSet,
	err error,
) {
	seenManifests = newDigestSet()
	seenBlobs = newDigestSet()

	repos, err := cfg.Data.RepositoriesList()
	if err != nil {
		return nil, nil, err
	}

	for _, repo := range repos {
		manifests, err := cfg.Data.ManifestsList(repo)
		if err != nil {
			return nil, nil, err
		}

		for digest := range manifests {
			if err := markManifest(cfg, repo, digest, seenManifests, seenBlobs); err != nil {
				return nil, nil, err
			}
		}
	}

	return seenBlobs, seenManifests, nil
}

func deleteUntaggedManifests(
	cfg config.Config,
	dryRun bool,
) error {
	repos, err := cfg.Data.RepositoriesList()
	if err != nil {
		return err
	}

	for _, repo := range repos {
		seenBlobs := newDigestSet()
		seenManifests := newDigestSet()

		tags, err := cfg.Data.TagsList(repo)
		if err != nil {
			return err
		}

		for _, tag := range tags {
			r, _, d, err := cfg.Data.ManifestGet(repo, tag)
			if err != nil {
				return err
			}
			r.Close()

			if err := markManifest(cfg, repo, d, seenManifests, seenBlobs); err != nil {
				return err
			}
		}

		digests, err := cfg.Data.ManifestsList(repo)
		if err != nil {
			return err
		}
		for d := range digests {
			if !seenManifests.has(d) {
				fmt.Printf("manifest eligible for deletion: %s\n", d)
				if !dryRun {
					if err := cfg.Data.ManifestDelete(repo, d); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func GarbageCollect(
	cfg config.Config,
	dryRun bool,
	lastAccess time.Duration,
	deleteUntagged bool,
) error {
	if deleteUntagged {
		if err := deleteUntaggedManifests(cfg, dryRun); err != nil {
			return err
		}
	}

	usedBlobs, usedManifests, err := collectReferencedBlobs(cfg)
	if err != nil {
		return err
	}

	blobs, err := cfg.Data.BlobsList()
	if err != nil {
		return err
	}

	for blob := range blobs {
		blobLastAccess, err := cfg.Data.BlobLastAccess(blob)
		if err != nil {
			return err
		}

		if time.Since(blobLastAccess) > lastAccess && !usedBlobs.has(blob) && !usedManifests.has(blob) {
			fmt.Printf("blob eligible for deletion: %s\n", blob)
			if !dryRun {
				if err := cfg.Data.BlobsDelete("", blob); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
