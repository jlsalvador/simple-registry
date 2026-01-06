package garbagecollect_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"testing"

	garbagecollect "github.com/jlsalvador/simple-registry/internal/cmd/garbage_collect"
	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/pkg/digest"
	"github.com/jlsalvador/simple-registry/pkg/mapset"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func getDigest(t *testing.T, data []byte) string {
	t.Helper()

	const algo = "sha256"
	d, err := digest.NewHasher(algo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Write(data); err != nil {
		t.Fatal(err)
	}
	return algo + ":" + d.GetHashAsString()
}

type TestBlob struct {
	Data   []byte
	Size   int64
	Digest string
}
type TestIndex struct {
	Payload registry.ImageIndexManifest
	Json    []byte
	Size    int64
	Digest  string
}
type TestAttestation struct {
	Payload registry.ImageManifest
	Json    []byte
	Size    int64
	Digest  string
}
type TestImage struct {
	Payload registry.ImageManifest
	Json    []byte
	Size    int64
	Digest  string
}

func setupTestEnvironment(t *testing.T) (
	cfg *config.Config,
	repo string,
	blobs []TestBlob,
	indexes []TestIndex,
	attestations []TestAttestation,
	images []TestImage,
	err error,
) {
	t.Helper()

	repo = "testing/repo"

	tmpdir := t.TempDir()

	if cfg, err = config.New("test", "test", "", tmpdir); err != nil {
		t.Fatal(err)
	}

	for i := range 4 {
		t.Run(fmt.Sprintf("write blob #%d", i), func(t *testing.T) {
			blob := TestBlob{Data: fmt.Appendf(nil, "blob #%d", i)}
			blob.Size = int64(len(blob.Data))
			blob.Digest = getDigest(t, blob.Data)

			// Write blob.
			uuid, err := cfg.Data.BlobsUploadCreate(repo)
			if err != nil {
				t.Fatal(err)
			}
			if err := cfg.Data.BlobsUploadWrite(repo, uuid, bytes.NewReader(blob.Data), -1); err != nil {
				t.Fatal(err)
			}
			if err := cfg.Data.BlobsUploadCommit(repo, uuid, blob.Digest); err != nil {
				t.Fatal(err)
			}

			blobs = append(blobs, blob)
		})
	}

	const reference = "latest"

	// There are 4 blobs, but we only writing 3 manifests to test GC.
	for i := range 3 {
		t.Run(fmt.Sprintf("write manifest #%d", i), func(t *testing.T) {
			imageManifest := registry.ImageManifest{
				SchemaVersion: 2,
				MediaType:     registry.MediaTypeOCIImageManifest,
				Config: registry.DescriptorManifest{
					MediaType: registry.MediaTypeOCIImageConfig,
					Digest:    "",
					Size:      0,
				},
				Layers: []registry.DescriptorManifest{
					{
						MediaType: "application/vnd.oci.image.layer.v1.raw",
						Digest:    blobs[i].Digest,
						Size:      blobs[i].Size,
					},
				},
			}
			imageManifestJson, _ := json.Marshal(imageManifest)
			imageManifestDigest := getDigest(t, imageManifestJson)
			if _, err := cfg.Data.ManifestPut(repo, imageManifestDigest, bytes.NewReader(imageManifestJson)); err != nil {
				t.Fatal(err)
			}
			images = append(images, TestImage{
				Payload: imageManifest,
				Json:    imageManifestJson,
				Size:    int64(len(imageManifestJson)),
				Digest:  imageManifestDigest,
			})

			attestation := registry.ImageManifest{
				SchemaVersion: 2,
				MediaType:     registry.MediaTypeOCIImageManifest,
				Subject: &registry.DescriptorManifest{
					MediaType: registry.MediaTypeOCIImageManifest,
					Digest:    imageManifestDigest,
					Size:      int64(len(imageManifestJson)),
				},
				Config: registry.DescriptorManifest{
					MediaType: registry.MediaTypeOCIImageConfig,
					Digest:    blobs[i].Digest,
					Size:      blobs[i].Size,
				},
				Layers: nil,
			}

			attestationJSON, _ := json.Marshal(attestation)
			attestationDigest := getDigest(t, attestationJSON)

			if _, err := cfg.Data.ManifestPut(
				repo,
				attestationDigest,
				bytes.NewReader(attestationJSON),
			); err != nil {
				t.Fatal(err)
			}
			attestations = append(attestations, TestAttestation{
				Payload: attestation,
				Json:    attestationJSON,
				Size:    int64(len(attestationJSON)),
				Digest:  attestationDigest,
			})

			testManifest := TestIndex{}
			testManifest.Payload = registry.NewImageIndexManifest()
			testManifest.Payload.Manifests = []registry.DescriptorManifest{
				{
					MediaType: registry.MediaTypeOCIImageManifest,
					Digest:    imageManifestDigest,
					Size:      int64(len(imageManifestJson)),
				},
			}
			testManifest.Json, _ = json.Marshal(testManifest.Payload)
			testManifest.Size = int64(len(testManifest.Json))
			testManifest.Digest = getDigest(t, testManifest.Json)
			if _, err = cfg.Data.ManifestPut(repo, reference, bytes.NewReader(testManifest.Json)); err != nil {
				t.Fatal(err)
			}

			indexes = append(indexes, testManifest)
		})
	}

	t.Run("check latest manifest", func(t *testing.T) {
		manifestDataStream, manifestSize, manifestDigest, err := cfg.Data.ManifestGet(repo, reference)
		manifestData, err := io.ReadAll(manifestDataStream)
		if err != nil {
			t.Fatal(err)
		}

		last := len(indexes) - 1
		if !bytes.Equal(indexes[last].Json, manifestData) {
			t.Errorf("manifest data mismatch: expected %v, got %v", indexes[last].Json, manifestData)
		}
		if indexes[last].Size != manifestSize {
			t.Errorf("manifest size mismatch: expected %d, got %d", indexes[last].Size, manifestSize)
		}
		if indexes[last].Digest != manifestDigest {
			t.Errorf("manifest digest mismatch: expected %s, got %s", indexes[last].Digest, manifestDigest)
		}
	})

	return cfg, repo, blobs, indexes, attestations, images, nil
}

func validateHowMany(
	t *testing.T,
	cfg config.Config,
	repo string,
	wantNManifests int,
	wantNBlobs int,
) {
	t.Helper()

	manifestDigests, err := cfg.Data.ManifestsList(repo)
	if err != nil {
		t.Fatal(err)
	}

	currentManifestDigests := slices.Collect(manifestDigests)
	slices.Sort(currentManifestDigests)
	nCurrentManifestDigests := len(currentManifestDigests)

	if nCurrentManifestDigests != wantNManifests {
		t.Fatalf("expected %d manifests, got %d", wantNManifests, nCurrentManifestDigests)
	}

	blobDigests, err := cfg.Data.BlobsList()
	if err != nil {
		t.Fatal(err)
	}

	currentBlobDigests := slices.Collect(blobDigests)
	slices.Sort(currentBlobDigests)
	nCurrentBlobDigests := len(currentBlobDigests)

	if nCurrentBlobDigests != wantNBlobs {
		t.Errorf("expected %d blobs, got %d", wantNBlobs, nCurrentBlobDigests)
	}
}

func TestGarbageCollect(t *testing.T) {
	cfg, repo, blobs, indexes, attestations, images, err := setupTestEnvironment(t)
	if err != nil {
		t.Fatal(err)
	}

	validateHowMany(
		t, *cfg, repo,
		len(indexes)+len(attestations)+len(images),
		len(blobs)+len(indexes)+len(attestations)+len(images),
	)

	gotDeletedBlobs, gotDeletedManifests, _, _, err := garbagecollect.GarbageCollect(*cfg, false, 1, false)
	if err != nil {
		t.Fatal(err)
	}

	nDeletedManifests := len(gotDeletedManifests)
	wantDeletedManifests := 0 // None should be deleted as we are not deleting untagged manifests.
	if nDeletedManifests != wantDeletedManifests {
		t.Errorf("expected %d deleted manifests, got %d", wantDeletedManifests, nDeletedManifests)
	}

	validateHowMany(
		t, *cfg, repo,
		len(indexes)+len(attestations)+len(images),
		len(blobs)-1+len(indexes)+len(attestations)+len(images),
	)

	// Only the fourth blob should be deleted.
	wantDeletedBlobs := mapset.NewMapSet[string]().Add(blobs[3].Digest)

	if !wantDeletedBlobs.Equal(gotDeletedBlobs) {
		t.Errorf("expected %v, got %v", wantDeletedBlobs, gotDeletedBlobs)
	}
}

func TestGarbageCollectUntaggedManifests(t *testing.T) {
	cfg, repo, blobs, indexes, attestations, images, err := setupTestEnvironment(t)
	if err != nil {
		t.Fatal(err)
	}

	validateHowMany(
		t, *cfg, repo,
		len(indexes)+len(attestations)+len(images),
		len(blobs)+len(indexes)+len(attestations)+len(images),
	)

	gotDeletedBlobs, gotDeletedManifests, _, _, err := garbagecollect.GarbageCollect(*cfg, false, 1, true)
	if err != nil {
		t.Fatal(err)
	}

	nDeletedManifests := len(gotDeletedManifests)
	wantDeletedManifests := len(indexes) - 1 + len(attestations) - 1 + len(images) - 1 // Only the last manifest has a tag.
	if nDeletedManifests != wantDeletedManifests {
		t.Errorf("expected %d deleted manifests, got %d", wantDeletedManifests, nDeletedManifests)
	}

	validateHowMany(
		t, *cfg, repo,
		3, // There are one index, one image, and one attestation.
		4, // There are one blob, one index, one image, and one attestation.
	)

	// All blobs and manifests except the last manifests should be deleted.
	wantDeletedBlobs := mapset.NewMapSet[string]().Add(
		blobs[0].Digest,
		blobs[1].Digest,
		blobs[3].Digest,
		indexes[0].Digest,
		indexes[1].Digest,
		images[0].Digest,
		images[1].Digest,
		attestations[0].Digest,
		attestations[1].Digest,
	)

	if !wantDeletedBlobs.Equal(gotDeletedBlobs) {
		t.Errorf("expected %v, got %v", wantDeletedBlobs, gotDeletedBlobs)
	}
}
