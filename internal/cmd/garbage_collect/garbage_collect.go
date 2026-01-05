package garbagecollect

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"time"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/internal/data"
	"github.com/jlsalvador/simple-registry/internal/data/proxy"
	"github.com/jlsalvador/simple-registry/pkg/mapset"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

// withoutProxy will return the underlying [proxy.ProxyDataStorage.Next] if it
// is a [proxy.ProxyDataStorage], otherwise it will return the same
// [data.DataStorage].
//
// The reason for this function is to avoid mirroring upstream while we are
// marking manifests and blobs.
func withoutProxy(data data.DataStorage) data.DataStorage {
	if d, ok := data.(*proxy.ProxyDataStorage); ok {
		return d.Next
	}
	return data
}

func markManifestByReferrers(
	cfg config.Config,
	repo string,
	digest string,
	seenManifests mapset.MapSet,
	seenBlobs mapset.MapSet,
) error {
	referrers, err := withoutProxy(cfg.Data).ReferrersGet(repo, digest)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if referrers == nil {
		return nil
	}
	for ref := range referrers {
		if err := markManifest(cfg, repo, ref, seenManifests, seenBlobs); err != nil {
			return err
		}
	}
	return nil
}

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

	r, _, _, err := withoutProxy(cfg.Data).ManifestGet(repo, digest)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// Manifest file not found, maybe because belongs to a proxy.
			return nil
		}
		return err
	}
	defer r.Close()

	payload, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	r.Close()

	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return err
	}

	_, hasSignatures := raw["signatures"]
	_, hasHistory := raw["history"]
	if hasSignatures || hasHistory {
		raw["mediaType"] = registry.MediaTypeDockerImageManifestV1
	}

	mediaType, _ := raw["mediaType"].(string)

	switch mediaType {

	case registry.MediaTypeOCIImageManifest,
		registry.MediaTypeDockerImageManifest:
		if err := markImageManifest(payload, seenBlobs); err != nil {
			return err
		}

	case registry.MediaTypeOCIImageIndex,
		registry.MediaTypeDockerManifestList:
		if err := markIndexManifest(payload, cfg, repo, seenManifests, seenBlobs); err != nil {
			return err
		}

	case registry.MediaTypeDockerImageManifestV1:
		if err := markDockerV1Manifest(payload, seenBlobs); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unsupported media type: %s", mediaType)
	}

	if err := markManifestByReferrers(cfg, repo, digest, seenManifests, seenBlobs); err != nil {
		return err
	}

	return nil
}

func markImageManifest(payload []byte, seenBlobs mapset.MapSet) error {
	var manifest registry.ImageManifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return err
	}

	if manifest.Config.Digest != "" {
		seenBlobs.Add(manifest.Config.Digest)
	}

	for _, layer := range manifest.Layers {
		seenBlobs.Add(layer.Digest)
	}
	return nil
}

func markIndexManifest(payload []byte, cfg config.Config, repo string, seenManifests mapset.MapSet, seenBlobs mapset.MapSet) error {
	var index registry.ImageIndexManifest
	if err := json.Unmarshal(payload, &index); err != nil {
		return err
	}

	for _, m := range index.Manifests {
		if err := markManifest(cfg, repo, m.Digest, seenManifests, seenBlobs); err != nil {
			return err
		}
	}
	return nil
}

func markDockerV1Manifest(payload []byte, seenBlobs mapset.MapSet) error {
	var manifest registry.DockerManifestV1
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return err
	}

	for _, layer := range manifest.FSLayers {
		if layer.BlobSum != "" {
			seenBlobs.Add(layer.BlobSum)
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

	repos, err := withoutProxy(cfg.Data).RepositoriesList()
	if err != nil {
		return nil, err
	}
	for _, repo := range repos {

		if !deleteUntagged {
			// Because we are not delete untagged manifests, include all the
			// manifests from this repo.

			manifests, err := withoutProxy(cfg.Data).ManifestsList(repo)
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
		tags, err := withoutProxy(cfg.Data).TagsList(repo)
		if err != nil {
			return nil, err
		}
		for _, tag := range tags {
			r, _, digest, err := withoutProxy(cfg.Data).ManifestGet(repo, tag)
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
	deletedSlice := []string{}

	repos, err := withoutProxy(cfg.Data).RepositoriesList()
	if err != nil {
		return nil, err
	}
	for _, repo := range repos {
		digests, err := withoutProxy(cfg.Data).ManifestsList(repo)
		if err != nil {
			return nil, err
		}

		for digest := range digests {
			manifestLastAccess, err := withoutProxy(cfg.Data).ManifestLastAccess(digest)
			if err != nil {
				return nil, err
			}

			if time.Since(manifestLastAccess) > lastAccess && !seenManifests.Contains(digest) {
				deletedSlice = append(deletedSlice, digest)
				if !dryRun {
					if err := withoutProxy(cfg.Data).ManifestDelete(repo, digest); err != nil && !errors.Is(err, fs.ErrNotExist) {
						return nil, err
					}
				}
			}
		}
	}

	return func(yield func(string) bool) {
		for _, digest := range deletedSlice {
			if !yield(digest) {
				return
			}
		}
	}, nil
}

func sweepBlobs(
	cfg config.Config,
	seenBlobs mapset.MapSet,
	seenManifests mapset.MapSet,
	dryRun bool,
	lastAccess time.Duration,
) (deleted iter.Seq[string], err error) {
	deletedSlice := []string{}

	blobs, err := withoutProxy(cfg.Data).BlobsList()
	if err != nil {
		return nil, err
	}

	for blob := range blobs {
		blobLastAccess, err := withoutProxy(cfg.Data).BlobLastAccess(blob)
		if err != nil {
			return nil, err
		}

		if time.Since(blobLastAccess) > lastAccess && !seenBlobs.Contains(blob) && !seenManifests.Contains(blob) {
			deletedSlice = append(deletedSlice, blob)
			if !dryRun {
				if err := withoutProxy(cfg.Data).BlobsDelete("", blob); err != nil {
					return nil, err
				}
			}
		}
	}

	return func(yield func(string) bool) {
		for _, digest := range deletedSlice {
			if !yield(digest) {
				return
			}
		}
	}, nil
}

// GarbageCollect deletes unreferrenced blobs (includes manifests blobs).
func GarbageCollect(
	cfg config.Config,
	dryRun bool,
	lastAccess time.Duration,
	deleteUntagged bool,
) (
	deletedBlobs iter.Seq[string],
	deletedManifests iter.Seq[string],
	markedBlobs mapset.MapSet,
	markedManifests mapset.MapSet,
	err error,
) {
	// Collect all the root manifests from all the repositories.
	roots, err := collectRootManifests(cfg, deleteUntagged)
	if err != nil {
		return
	}

	// Mark all manifests and blobs that are referenced by the roots.
	markedManifests = mapset.NewMapSet()
	markedBlobs = mapset.NewMapSet()
	for repo, digests := range roots {
		for _, d := range digests {
			if err = markManifest(cfg, repo, d, markedManifests, markedBlobs); err != nil {
				return
			}
		}
	}

	// Walk through all the manifests in the data store, and removes any that
	// is not referenced and is older than the last access time.
	deletedManifests, err = sweepManifests(cfg, markedManifests, dryRun, lastAccess)
	if err != nil {
		return
	}

	// Walk through all the blobs in the data store, and removes any that is not
	// referenced and is older than the last access time.
	deletedBlobs, err = sweepBlobs(cfg, markedBlobs, markedManifests, dryRun, lastAccess)
	if err != nil {
		return
	}

	return deletedBlobs, deletedManifests, markedBlobs, markedManifests, nil
}
