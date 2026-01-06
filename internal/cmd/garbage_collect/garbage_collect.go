package garbagecollect

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/internal/data"
	"github.com/jlsalvador/simple-registry/internal/data/proxy"
	"github.com/jlsalvador/simple-registry/pkg/mapset"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

type ManifestRef struct {
	Repo   string
	Digest string
}

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
	ds data.DataStorage,
	repo string,
	digest string,
	seenManifests mapset.MapSet[string],
	seenBlobs mapset.MapSet[string],
) error {
	referrers, err := ds.ReferrersGet(repo, digest)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if referrers == nil {
		return nil
	}
	for ref := range referrers {
		if err := markManifest(ds, repo, ref, seenManifests, seenBlobs); err != nil {
			return err
		}
	}
	return nil
}

// markManifest marks a manifest as seen and recursively marks all manifests and
// blobs it references.
func markManifest(
	ds data.DataStorage,
	repo string,
	digest string,
	seenManifests mapset.MapSet[string],
	seenBlobs mapset.MapSet[string],
) error {
	if seenManifests.Contains(digest) {
		return nil
	}
	seenManifests.Add(digest)

	r, _, _, err := ds.ManifestGet(repo, digest)
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
		if err := markIndexManifest(payload, ds, repo, seenManifests, seenBlobs); err != nil {
			return err
		}

	case registry.MediaTypeDockerImageManifestV1:
		if err := markDockerV1Manifest(payload, seenBlobs); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unsupported media type: %s", mediaType)
	}

	if err := markManifestByReferrers(ds, repo, digest, seenManifests, seenBlobs); err != nil {
		return err
	}

	return nil
}

func markImageManifest(payload []byte, seenBlobs mapset.MapSet[string]) error {
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

func markIndexManifest(
	payload []byte,
	ds data.DataStorage,
	repo string,
	seenManifests mapset.MapSet[string],
	seenBlobs mapset.MapSet[string],
) error {
	var index registry.ImageIndexManifest
	if err := json.Unmarshal(payload, &index); err != nil {
		return err
	}

	for _, m := range index.Manifests {
		if err := markManifest(ds, repo, m.Digest, seenManifests, seenBlobs); err != nil {
			return err
		}
	}
	return nil
}

func markDockerV1Manifest(payload []byte, seenBlobs mapset.MapSet[string]) error {
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
	ds data.DataStorage,
	deleteUntagged bool,
) (
	map[string][]string,
	error,
) {
	roots := map[string][]string{}

	repos, err := ds.RepositoriesList()
	if err != nil {
		return nil, err
	}
	for _, repo := range repos {

		if !deleteUntagged {
			// Because we are not delete untagged manifests, include all the
			// manifests from this repo.

			manifests, err := ds.ManifestsList(repo)
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
		tags, err := ds.TagsList(repo)
		if err != nil {
			return nil, err
		}
		for _, tag := range tags {
			r, _, digest, err := ds.ManifestGet(repo, tag)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					// Manifest file not found, maybe because belongs to a proxy.
					continue
				}
				return nil, err
			}
			r.Close()

			roots[repo] = append(roots[repo], digest)
		}

	}

	return roots, nil
}

func planSweepManifests(
	ds data.DataStorage,
	seenManifests mapset.MapSet[string],
	lastAccess time.Duration,
) (
	mapset.MapSet[ManifestRef],
	error,
) {
	toDelete := mapset.NewMapSet[ManifestRef]()

	repos, err := ds.RepositoriesList()
	if err != nil {
		return nil, err
	}

	for _, repo := range repos {
		digests, err := ds.ManifestsList(repo)
		if err != nil {
			return nil, err
		}

		for digest := range digests {
			manifestLastAccess, err := ds.ManifestLastAccess(digest)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					// Manifest file not found, maybe because belongs to a proxy.
					continue
				}
				return nil, err
			}

			if time.Since(manifestLastAccess) > lastAccess && !seenManifests.Contains(digest) {
				toDelete.Add(ManifestRef{repo, digest})
			}
		}
	}

	return toDelete, nil
}

func planSweepBlobs(
	ds data.DataStorage,
	seenBlobs mapset.MapSet[string],
	seenManifests mapset.MapSet[string],
	lastAccess time.Duration,
) (
	mapset.MapSet[string],
	error,
) {
	toDelete := mapset.NewMapSet[string]()

	blobs, err := ds.BlobsList()
	if err != nil {
		return nil, err
	}

	for blob := range blobs {
		blobLastAccess, err := ds.BlobLastAccess(blob)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				// Blob file not found, maybe because belongs to a proxy.
				continue
			}
			return nil, err
		}

		if time.Since(blobLastAccess) > lastAccess && !seenBlobs.Contains(blob) && !seenManifests.Contains(blob) {
			toDelete.Add(blob)
		}
	}

	return toDelete, nil
}

// GarbageCollect deletes unreferrenced blobs (includes manifests blobs).
func GarbageCollect(
	cfg config.Config,
	dryRun bool,
	lastAccess time.Duration,
	deleteUntagged bool,
) (
	mapset.MapSet[string],
	mapset.MapSet[ManifestRef],
	mapset.MapSet[string],
	mapset.MapSet[string],
	error,
) {
	ds := withoutProxy(cfg.Data)

	// Collect all the root manifests from all the repositories.
	roots, err := collectRootManifests(ds, deleteUntagged)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Mark all manifests and blobs that are referenced by the roots.
	markedManifests := mapset.NewMapSet[string]()
	markedBlobs := mapset.NewMapSet[string]()
	for repo, digests := range roots {
		for _, d := range digests {
			if err = markManifest(ds, repo, d, markedManifests, markedBlobs); err != nil {
				return nil, nil, nil, nil, err
			}
		}
	}

	// Walk through all the manifests in the data store, and removes any that
	// is not referenced and is older than the last access time.
	manifestsToDelete, err := planSweepManifests(ds, markedManifests, lastAccess)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Walk through all the blobs in the data store, and removes any that is not
	// referenced and is older than the last access time.
	blobsToDelete, err := planSweepBlobs(ds, markedBlobs, markedManifests, lastAccess)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if !dryRun {
		for m := range manifestsToDelete {
			if err = ds.ManifestDelete(m.Repo, m.Digest); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return nil, nil, nil, nil, err
			}
		}

		for blob := range blobsToDelete {
			if err = ds.BlobsDelete("", blob); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return nil, nil, nil, nil, err
			}
		}
	}

	return blobsToDelete, manifestsToDelete, markedBlobs, markedManifests, nil
}
