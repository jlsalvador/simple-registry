// Copyright 2026 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package filesystem_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jlsalvador/simple-registry/internal/data"
	"github.com/jlsalvador/simple-registry/internal/data/filesystem"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func createTestManifest(subject *registry.DescriptorManifest) []byte {
	manifest := registry.ImageManifest{
		SchemaVersion: 2,
		MediaType:     "application/vnd.oci.image.manifest.v1+json",
		Config: registry.DescriptorManifest{
			MediaType: "application/vnd.oci.image.config.v1+json",
			Digest:    "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			Size:      1234,
		},
		Layers: []registry.DescriptorManifest{
			{
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				Digest:    "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				Size:      5678,
			},
		},
		Subject: subject,
	}

	data, _ := json.Marshal(manifest)
	return data
}

func TestManifestPut(t *testing.T) {
	tests := []struct {
		name      string
		repo      string
		reference string
		manifest  []byte
		wantErr   bool
		checkFunc func(t *testing.T, storage *filesystem.FilesystemDataStorage, baseDir, repo, reference, digest string)
	}{
		{
			name:      "put manifest with tag",
			repo:      "myrepo",
			reference: "latest",
			manifest:  createTestManifest(nil),
			wantErr:   false,
			checkFunc: checkManifestWithTag,
		},
		{
			name:      "put manifest with digest",
			repo:      "myrepo",
			reference: "sha256:fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321",
			manifest:  createTestManifest(nil),
			wantErr:   false,
			checkFunc: checkManifestWithDigest,
		},
		{
			name:      "put manifest with subject (referrer)",
			repo:      "myrepo",
			reference: "latest",
			manifest: createTestManifest(&registry.DescriptorManifest{
				MediaType: "application/vnd.oci.image.manifest.v1+json",
				Digest:    "sha256:subject1234567890subject1234567890subject1234567890subject12345678",
				Size:      999,
			}),
			wantErr:   false,
			checkFunc: checkManifestWithSubject,
		},
		{
			name:      "invalid repository name",
			repo:      "invalid/repo/name!@#",
			reference: "latest",
			manifest:  createTestManifest(nil),
			wantErr:   true,
		},
		{
			name:      "empty manifest",
			repo:      "myrepo",
			reference: "latest",
			manifest:  []byte{},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			storage := filesystem.NewFilesystemDataStorage(baseDir)

			r := bytes.NewReader(tt.manifest)
			digest, err := storage.ManifestPut(tt.repo, tt.reference, r)

			if (err != nil) != tt.wantErr {
				t.Errorf("ManifestPut() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			validateDigest(t, digest)

			if tt.checkFunc != nil {
				tt.checkFunc(t, storage, baseDir, tt.repo, tt.reference, digest)
			}
		})
	}
}

func validateDigest(t *testing.T, digest string) {
	t.Helper()
	if digest == "" {
		t.Error("ManifestPut() returned empty digest")
		return
	}

	if !strings.HasPrefix(digest, "sha256:") {
		t.Errorf("ManifestPut() digest = %s, should start with sha256:", digest)
	}
}

func checkManifestWithTag(t *testing.T, storage *filesystem.FilesystemDataStorage, baseDir, repo, reference, digest string) {
	t.Helper()
	checkTagLink(t, baseDir, repo, reference, digest)
	checkRevisionLink(t, baseDir, repo, digest)
	checkBlobExists(t, baseDir, digest)
}

func checkManifestWithDigest(t *testing.T, storage *filesystem.FilesystemDataStorage, baseDir, repo, reference, digest string) {
	t.Helper()
	checkRevisionLinkExists(t, baseDir, repo, digest)
	ensureTagLinkNotCreated(t, baseDir, repo, reference)
}

func checkManifestWithSubject(t *testing.T, storage *filesystem.FilesystemDataStorage, baseDir, repo, reference, digest string) {
	t.Helper()
	subjectDigest := "sha256:subject1234567890subject1234567890subject1234567890subject12345678"
	checkReferrerLink(t, baseDir, repo, digest, subjectDigest)
}

func checkTagLink(t *testing.T, baseDir, repo, reference, digest string) {
	t.Helper()
	tagLink := filepath.Join(baseDir, "repositories", repo, "_manifests", "tags", reference, "current", "link")
	linkData, err := os.ReadFile(tagLink)
	if err != nil {
		t.Errorf("tag link not created: %v", err)
		return
	}
	if string(linkData) != digest {
		t.Errorf("tag link content = %s, want %s", string(linkData), digest)
	}
}

func checkRevisionLink(t *testing.T, baseDir, repo, digest string) {
	t.Helper()
	algo, hash, _ := splitDigest(digest)
	revisionLink := filepath.Join(baseDir, "repositories", repo, "_manifests", "revisions", algo, hash, "link")
	revData, err := os.ReadFile(revisionLink)
	if err != nil {
		t.Errorf("revision link not created: %v", err)
		return
	}
	if string(revData) != digest {
		t.Errorf("revision link content = %s, want %s", string(revData), digest)
	}
}

func checkBlobExists(t *testing.T, baseDir, digest string) {
	t.Helper()
	algo, hash, _ := splitDigest(digest)
	blobPath := filepath.Join(baseDir, "blobs", algo, hash[0:2], hash)
	if _, err := os.Stat(blobPath); err != nil {
		t.Errorf("blob not created: %v", err)
	}
}

func checkRevisionLinkExists(t *testing.T, baseDir, repo, digest string) {
	t.Helper()
	algo, hash, _ := splitDigest(digest)
	revisionLink := filepath.Join(baseDir, "repositories", repo, "_manifests", "revisions", algo, hash, "link")
	if _, err := os.Stat(revisionLink); err != nil {
		t.Errorf("revision link not created: %v", err)
	}
}

func ensureTagLinkNotCreated(t *testing.T, baseDir, repo, reference string) {
	t.Helper()
	tagLink := filepath.Join(baseDir, "repositories", repo, "_manifests", "tags", reference, "current", "link")
	if _, err := os.Stat(tagLink); err == nil {
		t.Error("tag link should not be created for digest reference")
	}
}

func checkReferrerLink(t *testing.T, baseDir, repo, digest, subjectDigest string) {
	t.Helper()
	algo, hash, _ := splitDigest(subjectDigest)
	referrerDir := filepath.Join(baseDir, "repositories", repo, "_manifests", "referrers", algo, hash, digest)
	linkPath := filepath.Join(referrerDir, "link")

	linkData, err := os.ReadFile(linkPath)
	if err != nil {
		t.Errorf("referrer link not created: %v", err)
		return
	}
	if string(linkData) != subjectDigest {
		t.Errorf("referrer link content = %s, want %s", string(linkData), subjectDigest)
	}
}

func TestManifestGet(t *testing.T) {
	tests := []struct {
		name      string
		repo      string
		reference string
		setupFunc func(storage *filesystem.FilesystemDataStorage, baseDir string) (expectedDigest string, expectedContent []byte)
		wantErr   bool
	}{
		{
			name:      "get manifest by tag",
			repo:      "myrepo",
			reference: "v1.0.0",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) (string, []byte) {
				manifest := createTestManifest(nil)
				r := bytes.NewReader(manifest)
				digest, _ := storage.ManifestPut("myrepo", "v1.0.0", r)
				return digest, manifest
			},
			wantErr: false,
		},
		{
			name:      "get manifest by digest",
			repo:      "myrepo",
			reference: "", // Will be set by setupFunc
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) (string, []byte) {
				manifest := createTestManifest(nil)
				r := bytes.NewReader(manifest)
				digest, _ := storage.ManifestPut("myrepo", "latest", r)
				return digest, manifest
			},
			wantErr: false,
		},
		{
			name:      "get non-existent manifest by tag",
			repo:      "myrepo",
			reference: "nonexistent",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) (string, []byte) {
				return "", nil
			},
			wantErr: true,
		},
		{
			name:      "get non-existent manifest by digest",
			repo:      "myrepo",
			reference: "sha256:nonexistent1234567890nonexistent1234567890nonexistent1234567890abcd",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) (string, []byte) {
				return "", nil
			},
			wantErr: true,
		},
		{
			name:      "invalid repository name",
			repo:      "invalid/repo!@#",
			reference: "latest",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) (string, []byte) {
				return "", nil
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			storage := filesystem.NewFilesystemDataStorage(baseDir)

			expectedDigest, expectedContent := tt.setupFunc(storage, baseDir)

			// Use the returned digest if reference should be a digest
			reference := tt.reference
			if reference == "" && expectedDigest != "" {
				reference = expectedDigest
			}

			r, size, digest, err := storage.ManifestGet(tt.repo, reference)

			if (err != nil) != tt.wantErr {
				t.Errorf("ManifestGet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			defer r.Close()

			if digest != expectedDigest {
				t.Errorf("ManifestGet() digest = %s, want %s", digest, expectedDigest)
			}

			content, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("failed to read manifest content: %v", err)
			}

			if !bytes.Equal(content, expectedContent) {
				t.Errorf("ManifestGet() content mismatch")
			}

			if size != int64(len(expectedContent)) {
				t.Errorf("ManifestGet() size = %d, want %d", size, len(expectedContent))
			}
		})
	}
}

func TestManifestDelete(t *testing.T) {
	tests := []struct {
		name      string
		repo      string
		reference string
		setupFunc func(storage *filesystem.FilesystemDataStorage, baseDir string) string
		checkFunc func(t *testing.T, baseDir, repo string)
		wantErr   bool
	}{
		{
			name:      "delete manifest by tag",
			repo:      "myrepo",
			reference: "v1.0.0",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) string {
				manifest := createTestManifest(nil)
				r := bytes.NewReader(manifest)
				storage.ManifestPut("myrepo", "v1.0.0", r)
				return ""
			},
			checkFunc: func(t *testing.T, baseDir, repo string) {
				tagDir := filepath.Join(baseDir, "repositories", repo, "_manifests", "tags", "v1.0.0")
				if _, err := os.Stat(tagDir); !os.IsNotExist(err) {
					t.Error("tag directory should be deleted")
				}
			},
			wantErr: false,
		},
		{
			name:      "delete manifest by digest",
			repo:      "myrepo",
			reference: "", // Will be set by setupFunc
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) string {
				manifest := createTestManifest(nil)
				r := bytes.NewReader(manifest)
				digest, _ := storage.ManifestPut("myrepo", "latest", r)
				return digest
			},
			checkFunc: func(t *testing.T, baseDir, repo string) {
				// Revision should be deleted
				// Note: We can't check exact path without knowing the digest
			},
			wantErr: false,
		},
		{
			name:      "delete manifest with referrer",
			repo:      "myrepo",
			reference: "", // Will be set by setupFunc
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) string {
				// First create a subject manifest
				subjectManifest := createTestManifest(nil)
				r := bytes.NewReader(subjectManifest)
				subjectDigest, _ := storage.ManifestPut("myrepo", "subject", r)

				// Then create a referrer manifest pointing to the subject
				referrerManifest := createTestManifest(&registry.DescriptorManifest{
					MediaType: "application/vnd.oci.image.manifest.v1+json",
					Digest:    subjectDigest,
					Size:      int64(len(subjectManifest)),
				})
				r2 := bytes.NewReader(referrerManifest)
				referrerDigest, _ := storage.ManifestPut("myrepo", "referrer", r2)
				return referrerDigest
			},
			wantErr: false,
		},
		{
			name:      "delete non-existent tag",
			repo:      "myrepo",
			reference: "nonexistent",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) string {
				return ""
			},
			wantErr: true,
		},
		{
			name:      "delete non-existent digest",
			repo:      "myrepo",
			reference: "sha256:nonexistent1234567890nonexistent1234567890nonexistent1234567890abcd",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) string {
				return ""
			},
			wantErr: true,
		},
		{
			name:      "invalid repository name",
			repo:      "invalid/repo!@#",
			reference: "latest",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) string {
				return ""
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			storage := filesystem.NewFilesystemDataStorage(baseDir)

			digest := tt.setupFunc(storage, baseDir)

			// Use the returned digest if reference should be a digest
			reference := tt.reference
			if reference == "" && digest != "" {
				reference = digest
			}

			err := storage.ManifestDelete(tt.repo, reference)

			if (err != nil) != tt.wantErr {
				t.Errorf("ManifestDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, baseDir, tt.repo)
			}
		})
	}
}

func TestManifestsList(t *testing.T) {
	tests := []struct {
		name          string
		repo          string
		setupFunc     func(storage *filesystem.FilesystemDataStorage)
		expectedCount int
		wantErr       bool
	}{
		{
			name: "list manifests with revisions only",
			repo: "myrepo",
			setupFunc: func(storage *filesystem.FilesystemDataStorage) {
				m1 := createTestManifest(nil)
				storage.ManifestPut("myrepo", "v1", bytes.NewReader(m1))
				m2 := createTestManifest(nil)
				storage.ManifestPut("myrepo", "v2", bytes.NewReader(m2))
			},
			expectedCount: 2,
			wantErr:       false,
		},
		{
			name: "list manifests with referrers",
			repo: "myrepo",
			setupFunc: func(storage *filesystem.FilesystemDataStorage) {
				// Create subject
				subject := createTestManifest(nil)
				subjectDigest, _ := storage.ManifestPut("myrepo", "subject", bytes.NewReader(subject))

				// Create referrer
				referrer := createTestManifest(&registry.DescriptorManifest{
					MediaType: "application/vnd.oci.image.manifest.v1+json",
					Digest:    subjectDigest,
					Size:      int64(len(subject)),
				})
				storage.ManifestPut("myrepo", "referrer", bytes.NewReader(referrer))
			},
			expectedCount: 2, // subject + referrer
			wantErr:       false,
		},
		{
			name: "list empty repository",
			repo: "emptyrepo",
			setupFunc: func(storage *filesystem.FilesystemDataStorage) {
				// Don't create anything
			},
			expectedCount: 0,
			wantErr:       false,
		},
		{
			name: "list manifests with duplicate digest",
			repo: "myrepo",
			setupFunc: func(storage *filesystem.FilesystemDataStorage) {
				m := createTestManifest(nil)
				storage.ManifestPut("myrepo", "tag1", bytes.NewReader(m))
				storage.ManifestPut("myrepo", "tag2", bytes.NewReader(m))
			},
			expectedCount: 1, // Same manifest, different tags
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			storage := filesystem.NewFilesystemDataStorage(baseDir)

			tt.setupFunc(storage)

			digests, err := storage.ManifestsList(tt.repo)

			if (err != nil) != tt.wantErr {
				t.Errorf("ManifestsList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			count := 0
			seenDigests := make(map[string]bool)
			for digest := range digests {
				count++
				if seenDigests[digest] {
					t.Errorf("ManifestsList() returned duplicate digest: %s", digest)
				}
				seenDigests[digest] = true

				if !strings.Contains(digest, ":") {
					t.Errorf("ManifestsList() invalid digest format: %s", digest)
				}
			}
		})
	}
}

func TestManifestLastAccess(t *testing.T) {
	baseDir := t.TempDir()
	storage := filesystem.NewFilesystemDataStorage(baseDir)

	manifest := createTestManifest(nil)
	r := bytes.NewReader(manifest)
	digest, err := storage.ManifestPut("myrepo", "latest", r)
	if err != nil {
		t.Fatalf("ManifestPut() failed: %v", err)
	}

	// Small delay to ensure time difference
	time.Sleep(10 * time.Millisecond)

	lastAccess, err := storage.ManifestLastAccess(digest)
	if err != nil {
		t.Errorf("ManifestLastAccess() error = %v", err)
		return
	}

	// Check that the returned time is reasonable (within the last minute)
	now := time.Now()
	diff := now.Sub(lastAccess)
	if diff < 0 || diff > time.Minute {
		t.Errorf("ManifestLastAccess() returned unexpected time: %v (diff from now: %v)", lastAccess, diff)
	}
}

func TestManifestIntegration(t *testing.T) {
	baseDir := t.TempDir()
	storage := filesystem.NewFilesystemDataStorage(baseDir)

	// 1. Put a manifest with multiple tags
	manifest := createTestManifest(nil)

	digest1, err := storage.ManifestPut("integration-repo", "v1.0.0", bytes.NewReader(manifest))
	if err != nil {
		t.Fatalf("ManifestPut(v1.0.0) failed: %v", err)
	}

	digest2, err := storage.ManifestPut("integration-repo", "latest", bytes.NewReader(manifest))
	if err != nil {
		t.Fatalf("ManifestPut(latest) failed: %v", err)
	}

	// Same manifest should produce same digest
	if digest1 != digest2 {
		t.Errorf("same manifest produced different digests: %s vs %s", digest1, digest2)
	}

	// 2. Get manifest by tag
	r1, size1, retDigest1, err := storage.ManifestGet("integration-repo", "v1.0.0")
	if err != nil {
		t.Fatalf("ManifestGet(v1.0.0) failed: %v", err)
	}
	r1.Close()

	if retDigest1 != digest1 {
		t.Errorf("ManifestGet(v1.0.0) digest = %s, want %s", retDigest1, digest1)
	}

	// 3. Get manifest by digest
	r2, size2, _, err := storage.ManifestGet("integration-repo", digest1)
	if err != nil {
		t.Fatalf("ManifestGet(digest) failed: %v", err)
	}
	r2.Close()

	if size1 != size2 {
		t.Errorf("sizes differ: %d vs %d", size1, size2)
	}

	// 4. List manifests
	digests, err := storage.ManifestsList("integration-repo")
	if err != nil {
		t.Fatalf("ManifestsList() failed: %v", err)
	}

	count := 0
	for range digests {
		count++
	}

	if count != 1 {
		t.Errorf("ManifestsList() returned %d manifests, want 1", count)
	}

	// 5. Delete by tag
	if err := storage.ManifestDelete("integration-repo", "v1.0.0"); err != nil {
		t.Fatalf("ManifestDelete(v1.0.0) failed: %v", err)
	}

	// Tag should be gone
	_, _, _, err = storage.ManifestGet("integration-repo", "v1.0.0")
	if err == nil {
		t.Error("ManifestGet(v1.0.0) should fail after delete")
	}

	// But digest should still work (because latest tag still references it)
	r3, _, _, err := storage.ManifestGet("integration-repo", digest1)
	if err != nil {
		t.Errorf("ManifestGet(digest) should still work: %v", err)
	} else {
		r3.Close()
	}

	// 6. Delete by digest
	if err := storage.ManifestDelete("integration-repo", digest1); err != nil {
		t.Fatalf("ManifestDelete(digest) failed: %v", err)
	}
}

func TestManifestReferrerIntegration(t *testing.T) {
	baseDir := t.TempDir()
	storage := filesystem.NewFilesystemDataStorage(baseDir)

	// Create subject manifest
	subjectManifest := createTestManifest(nil)
	subjectDigest, err := storage.ManifestPut("repo", "subject", bytes.NewReader(subjectManifest))
	if err != nil {
		t.Fatalf("failed to create subject manifest: %v", err)
	}

	// Create referrer manifest
	referrerManifest := createTestManifest(&registry.DescriptorManifest{
		MediaType: "application/vnd.oci.image.manifest.v1+json",
		Digest:    subjectDigest,
		Size:      int64(len(subjectManifest)),
	})
	referrerDigest, err := storage.ManifestPut("repo", "referrer", bytes.NewReader(referrerManifest))
	if err != nil {
		t.Fatalf("failed to create referrer manifest: %v", err)
	}

	// Verify referrer directory exists
	algo, hash, _ := splitDigest(subjectDigest)
	referrerDir := filepath.Join(baseDir, "repositories", "repo", "_manifests", "referrers", algo, hash, referrerDigest)
	if _, err := os.Stat(referrerDir); err != nil {
		t.Errorf("referrer directory not created: %v", err)
	}

	// List manifests should include both
	digests, _ := storage.ManifestsList("repo")
	count := 0
	for range digests {
		count++
	}
	if count != 2 {
		t.Errorf("expected 2 manifests, got %d", count)
	}

	// Delete referrer
	if err := storage.ManifestDelete("repo", referrerDigest); err != nil {
		t.Fatalf("failed to delete referrer: %v", err)
	}

	// Verify referrer directory is cleaned up
	if _, err := os.Stat(referrerDir); !os.IsNotExist(err) {
		t.Error("referrer directory should be deleted")
	}
}

func TestInvalidManifestJSON(t *testing.T) {
	baseDir := t.TempDir()
	storage := filesystem.NewFilesystemDataStorage(baseDir)

	// Put invalid JSON (should still succeed as we store it as-is)
	invalidJSON := []byte("{invalid json")
	digest, err := storage.ManifestPut("repo", "invalid", bytes.NewReader(invalidJSON))
	if err != nil {
		t.Errorf("ManifestPut() with invalid JSON should succeed: %v", err)
	}

	// Should be able to retrieve it
	r, _, retDigest, err := storage.ManifestGet("repo", "invalid")
	if err != nil {
		t.Errorf("ManifestGet() should succeed: %v", err)
	}
	if r != nil {
		r.Close()
	}

	if retDigest != digest {
		t.Errorf("digest mismatch: got %s, want %s", retDigest, digest)
	}
}

// Helper function to split digest into algorithm and hash
func splitDigest(digest string) (algo, hash string, err error) {
	parts := strings.SplitN(digest, ":", 2)
	if len(parts) != 2 {
		return "", "", data.ErrDigestInvalid
	}
	return parts[0], parts[1], nil
}
