package garbagecollect

import (
	"encoding/json"
	"fmt"
	"iter"
	"time"

	"github.com/jlsalvador/simple-registry/internal/config"
	pkgIter "github.com/jlsalvador/simple-registry/pkg/iter"
	"github.com/jlsalvador/simple-registry/pkg/mapset"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

// markManifest marks a manifest as seen and recursively marks all manifests and
// blobs it references.
func markManifest(
	cfg config.Config,
	repo string,
	digest string,
	seenManifests mapset.MapSet,
	seenBlobs mapset.MapSet,
) error {
	if seenManifests.Contains(digest) {
		return nil
	}
	seenManifests.Add(digest)

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
			seenBlobs.Add(layer.Digest)
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

	referrers, err := cfg.Data.ReferrersGet(repo, digest)
	if err != nil {
		return err
	}
	for ref := range referrers {
		if err := markManifest(cfg, repo, ref, seenManifests, seenBlobs); err != nil {
			return err
		}
	}

	return nil
}

// collectRootManifests returns a map of repository names to their root manifest
// digests.
//
// If `deleteUntagged` is true, only root manifests that have at least one tag
// are included.
func collectRootManifests(
	cfg config.Config,
	deleteUntagged bool,
) (map[string][]string, error) {
	roots := map[string][]string{}

	repos, err := cfg.Data.RepositoriesList()
	if err != nil {
		return nil, err
	}
	for _, repo := range repos {

		if !deleteUntagged {
			// Because we are not delete untagged manifests, include all the
			// manifests from this repo.

			manifests, err := cfg.Data.ManifestsList(repo)
			if err != nil {
				return nil, err
			}
			for digest := range manifests {
				roots[repo] = append(roots[repo], digest)
			}
			continue
		}

		// We are going to delete untagged manifests, so only include the
		// manifests that have tags.
		tags, err := cfg.Data.TagsList(repo)
		if err != nil {
			return nil, err
		}
		for _, tag := range tags {
			r, _, digest, err := cfg.Data.ManifestGet(repo, tag)
			if err != nil {
				return nil, err
			}
			r.Close()

			roots[repo] = append(roots[repo], digest)
		}

	}

	return roots, nil
}

func sweepManifests(
	cfg config.Config,
	seenManifests mapset.MapSet,
	dryRun bool,
	lastAccess time.Duration,
) (deleted iter.Seq[string], err error) {
	repos, err := cfg.Data.RepositoriesList()
	if err != nil {
		return nil, err
	}
	for _, repo := range repos {
		digests, err := cfg.Data.ManifestsList(repo)
		if err != nil {
			return nil, err
		}

		for digest := range digests {
			blobLastAccess, err := cfg.Data.BlobLastAccess(digest)
			if err != nil {
				return nil, err
			}

			if time.Since(blobLastAccess) > lastAccess && !seenManifests.Contains(digest) {
				//TODO: add eligible manifest to iterator `deleted`
				if !dryRun {
					if err := cfg.Data.ManifestDelete(repo, digest); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	return nil, nil
}

func sweepBlobs(
	cfg config.Config,
	seenBlobs mapset.MapSet,
	seenManifests mapset.MapSet,
	dryRun bool,
	lastAccess time.Duration,
) (deleted iter.Seq[string], err error) {
	blobs, err := cfg.Data.BlobsList()
	if err != nil {
		return nil, err
	}

	for blob := range blobs {
		blobLastAccess, err := cfg.Data.BlobLastAccess(blob)
		if err != nil {
			return nil, err
		}

		if time.Since(blobLastAccess) > lastAccess && !seenBlobs.Contains(blob) && !seenManifests.Contains(blob) {
			//TODO: add eligible blob to iterator `deleted`
			if !dryRun {
				if err := cfg.Data.BlobsDelete("", blob); err != nil {
					return nil, err
				}
			}
		}
	}

	return nil, nil
}

// GarbageCollect deletes unreferrenced blobs (includes manifests blobs).
func GarbageCollect(
	cfg config.Config,
	dryRun bool,
	lastAccess time.Duration,
	deleteUntagged bool,
) (deleted iter.Seq[string], err error) {
	// Collect all the root manifests from all the repositories.
	roots, err := collectRootManifests(cfg, deleteUntagged)
	if err != nil {
		return nil, err
	}

	// Mark all manifests and blobs that are referenced by the roots.
	seenManifests := mapset.NewMapSet()
	seenBlobs := mapset.NewMapSet()
	for repo, digests := range roots {
		for _, d := range digests {
			if err := markManifest(cfg, repo, d, seenManifests, seenBlobs); err != nil {
				return nil, err
			}
		}
	}

	// Walk through all the manifests in the data store, and removes any that
	// is not referenced and is older than the last access time.
	deletedManifests, err := sweepManifests(cfg, seenManifests, dryRun, lastAccess)
	if err != nil {
		return nil, err
	}

	// Walk through all the blobs in the data store, and removes any that is not
	// referenced and is older than the last access time.
	deletedBlobs, err := sweepBlobs(cfg, seenBlobs, seenManifests, dryRun, lastAccess)
	if err != nil {
		return nil, err
	}

	return pkgIter.Concat(deletedManifests, deletedBlobs), nil
}
