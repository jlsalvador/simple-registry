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
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func TestGarbageCollect(t *testing.T) {
	tmpdir := t.TempDir()

	cfg, err := config.New("test", "test", "", tmpdir)
	if err != nil {
		t.Fatal(err)
	}

	const repo = "testing/repo"
	const algo = "sha256"
	blobs := []struct {
		Data   []byte
		Size   int64
		Digest string
	}{
		{Data: []byte("first blob")},
		{Data: []byte("second blob")},
		{Data: []byte("third blob")},
	}
	for i := range blobs {
		t.Run(fmt.Sprintf("write blob #%d", i), func(t *testing.T) {
			blobs[i].Size = int64(len(blobs[i].Data))

			// Calculate blob digest.
			d, err := digest.NewHasher(algo)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := d.Write(blobs[i].Data); err != nil {
				t.Fatal(err)
			}
			blobs[i].Digest = algo + ":" + d.GetHashAsString()

			// Write blob.
			uuid, err := cfg.Data.BlobsUploadCreate(repo)
			if err != nil {
				t.Fatal(err)
			}
			if err := cfg.Data.BlobsUploadWrite(repo, uuid, bytes.NewReader(blobs[i].Data), -1); err != nil {
				t.Fatal(err)
			}
			if err := cfg.Data.BlobsUploadCommit(repo, uuid, blobs[i].Digest); err != nil {
				t.Fatal(err)
			}
		})
	}

	const reference = "latest"
	type testManifest struct {
		payload registry.ImageIndexManifest
		json    []byte
		size    int64
		digest  string
	}
	testManifests := []testManifest{}
	for i := range blobs {
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
			imageManifestJson, err := json.Marshal(imageManifest)
			if err != nil {
				t.Fatal(err)
			}
			d, err := digest.NewHasher(algo)
			if err != nil {
				t.Fatal(err)
			}
			_, err = d.Write(imageManifestJson)
			if err != nil {
				t.Fatal(err)
			}
			imageManifestDigest := algo + ":" + d.GetHashAsString()
			if _, err := cfg.Data.ManifestPut(repo, imageManifestDigest, bytes.NewReader(imageManifestJson)); err != nil {
				t.Fatal(err)
			}

			testManifest := testManifest{}
			testManifest.payload = registry.NewImageIndexManifest()
			testManifest.payload.Manifests = []registry.DescriptorManifest{
				{
					MediaType: registry.MediaTypeOCIImageManifest,
					Digest:    imageManifestDigest,
					Size:      int64(len(imageManifestJson)),
				},
			}
			if testManifest.json, err = json.Marshal(testManifest.payload); err != nil {
				t.Fatal(err)
			}
			testManifest.size = int64(len(testManifest.json))
			d, err = digest.NewHasher(algo)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := d.Write(testManifest.json); err != nil {
				t.Fatal(err)
			}
			testManifest.digest = algo + ":" + d.GetHashAsString()

			if _, err = cfg.Data.ManifestPut(repo, reference, bytes.NewReader(testManifest.json)); err != nil {
				t.Fatal(err)
			}

			testManifests = append(testManifests, testManifest)
		})
	}

	t.Run("check latest manifest", func(t *testing.T) {
		manifestDataStream, manifestSize, manifestDigest, err := cfg.Data.ManifestGet(repo, reference)
		manifestData, err := io.ReadAll(manifestDataStream)
		if err != nil {
			t.Fatal(err)
		}

		last := len(testManifests) - 1
		if !bytes.Equal(testManifests[last].json, manifestData) {
			t.Errorf("manifest data mismatch: expected %v, got %v", testManifests[last].json, manifestData)
		}
		if testManifests[last].size != manifestSize {
			t.Errorf("manifest size mismatch: expected %d, got %d", testManifests[last].size, manifestSize)
		}
		if testManifests[last].digest != manifestDigest {
			t.Errorf("manifest digest mismatch: expected %s, got %s", testManifests[last].digest, manifestDigest)
		}
	})

	manifestDigests, err := cfg.Data.ManifestsList(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(slices.Collect(manifestDigests)) != 2*len(blobs) {
		t.Fatalf("expected %d manifests, got %d", 2*len(blobs), len(slices.Collect(manifestDigests)))
	}

	if _, err := garbagecollect.GarbageCollect(*cfg, false, 1, true); err != nil {
		t.Fatal(err)
	}

	manifestDigests, err = cfg.Data.ManifestsList(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(slices.Collect(manifestDigests)) != 2 {
		t.Errorf("expected %d manifests, got %d", 2, len(slices.Collect(manifestDigests)))
	}

	blobDigests, err := cfg.Data.BlobsList()
	if err != nil {
		t.Fatal(err)
	}
	if len(slices.Collect(blobDigests)) != 3 {
		t.Errorf("expected %d blobs, got %d", 3, len(slices.Collect(blobDigests)))
	}
}
