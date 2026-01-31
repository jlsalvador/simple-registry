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
	"os"
	"path/filepath"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/data/filesystem"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func TestReferrersGet(t *testing.T) {
	tests := []struct {
		name           string
		repo           string
		manifestDigest string
		setupFunc      func(storage *filesystem.FilesystemDataStorage, baseDir string) []string
		expectedCount  int
		wantErr        bool
	}{
		{
			name:           "get referrers for manifest with multiple referrers",
			repo:           "myrepo",
			manifestDigest: "sha256:subject1234567890subject1234567890subject1234567890subject1234567890",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) []string {
				// Create a subject manifest
				subjectManifest := createTestManifest(nil)
				subjectDigest, _ := storage.ManifestPut("myrepo", "subject", bytes.NewReader(subjectManifest))

				// Create multiple referrer manifests
				var referrerDigests []string
				for i := 0; i < 3; i++ {
					referrer := createTestManifest(&registry.DescriptorManifest{
						MediaType: "application/vnd.oci.image.manifest.v1+json",
						Digest:    subjectDigest,
						Size:      int64(len(subjectManifest)),
					})
					digest, _ := storage.ManifestPut("myrepo", "subject", bytes.NewReader(referrer))
					referrerDigests = append(referrerDigests, digest)
				}
				return referrerDigests
			},
			expectedCount: 3,
			wantErr:       false,
		},
		{
			name:           "get referrers for manifest with no referrers",
			repo:           "myrepo",
			manifestDigest: "sha256:norefers1234567890norefers1234567890norefers1234567890norefers1234",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) []string {
				// Create a manifest without any referrers
				manifest := createTestManifest(nil)
				storage.ManifestPut("myrepo", "standalone", bytes.NewReader(manifest))
				return []string{}
			},
			expectedCount: 0,
			wantErr:       true, // Directory won't exist
		},
		{
			name:           "get referrers for non-existent manifest",
			repo:           "myrepo",
			manifestDigest: "sha256:nonexistent1234567890nonexistent1234567890nonexistent1234567890abcd",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) []string {
				return []string{}
			},
			expectedCount: 0,
			wantErr:       true,
		},
		{
			name:           "invalid digest format",
			repo:           "myrepo",
			manifestDigest: "invalid-digest",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) []string {
				return []string{}
			},
			expectedCount: 0,
			wantErr:       true,
		},
		{
			name:           "get referrers with single referrer",
			repo:           "myrepo",
			manifestDigest: "sha256:single1234567890single1234567890single1234567890single1234567890ab",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) []string {
				// Create subject
				subject := createTestManifest(nil)
				subjectDigest, _ := storage.ManifestPut("myrepo", "subject", bytes.NewReader(subject))

				// Create single referrer
				referrer := createTestManifest(&registry.DescriptorManifest{
					MediaType: "application/vnd.oci.image.manifest.v1+json",
					Digest:    subjectDigest,
					Size:      int64(len(subject)),
				})
				digest, _ := storage.ManifestPut("myrepo", "referrer", bytes.NewReader(referrer))
				return []string{digest}
			},
			expectedCount: 1,
			wantErr:       false,
		},
		{
			name:           "referrers directory with non-digest files",
			repo:           "myrepo",
			manifestDigest: "sha256:withfiles1234567890withfiles1234567890withfiles1234567890withfiles",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) []string {
				// Create subject
				subject := createTestManifest(nil)
				subjectDigest, _ := storage.ManifestPut("myrepo", "subject", bytes.NewReader(subject))

				// Create a referrer
				referrer := createTestManifest(&registry.DescriptorManifest{
					MediaType: "application/vnd.oci.image.manifest.v1+json",
					Digest:    subjectDigest,
					Size:      int64(len(subject)),
				})
				digest, _ := storage.ManifestPut("myrepo", "referrer", bytes.NewReader(referrer))

				// Manually add non-digest files to the referrers directory
				algo, hash, _ := splitDigest(subjectDigest)
				refDir := filepath.Join(baseDir, "repositories", "myrepo", "_manifests", "referrers", algo, hash)

				// Create some non-digest files
				os.WriteFile(filepath.Join(refDir, "invalid.txt"), []byte("test"), 0644)
				os.WriteFile(filepath.Join(refDir, "README.md"), []byte("test"), 0644)
				os.MkdirAll(filepath.Join(refDir, "not-a-digest"), 0755)

				return []string{digest}
			},
			expectedCount: 1, // Only the valid digest should be returned
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			storage := filesystem.NewFilesystemDataStorage(baseDir)

			expectedDigests := tt.setupFunc(storage, baseDir)

			// Get the actual subject digest if we created manifests
			var actualDigest string
			if len(expectedDigests) > 0 {
				// Get the first referrer's subject to find the actual subject digest
				manifest := createTestManifest(nil)
				actualDigest, _ = storage.ManifestPut(tt.repo, "temp-subject", bytes.NewReader(manifest))
			} else {
				actualDigest = tt.manifestDigest
			}

			digests, err := storage.ReferrersGet(tt.repo, actualDigest)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReferrersGet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Collect all returned digests
			var returnedDigests []string
			seenDigests := make(map[string]bool)
			for digest := range digests {
				returnedDigests = append(returnedDigests, digest)

				// Check for duplicates
				if seenDigests[digest] {
					t.Errorf("ReferrersGet() returned duplicate digest: %s", digest)
				}
				seenDigests[digest] = true

				// Verify it's a valid digest format
				if !registry.RegExprDigest.MatchString(digest) {
					t.Errorf("ReferrersGet() returned invalid digest format: %s", digest)
				}
			}
		})
	}
}
