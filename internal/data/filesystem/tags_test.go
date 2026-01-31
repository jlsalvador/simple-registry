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
	"slices"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/data/filesystem"
)

func TestTagsList(t *testing.T) {
	tests := []struct {
		name         string
		repo         string
		setupFunc    func(storage *filesystem.FilesystemDataStorage, baseDir string)
		expectedTags []string
		wantErr      bool
	}{
		{
			name: "list single tag",
			repo: "myrepo",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				storage.ManifestPut("myrepo", "latest", bytes.NewReader(manifest))
			},
			expectedTags: []string{"latest"},
			wantErr:      false,
		},
		{
			name: "list multiple tags",
			repo: "myrepo",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				storage.ManifestPut("myrepo", "v1.0.0", bytes.NewReader(manifest))
				storage.ManifestPut("myrepo", "v1.0.1", bytes.NewReader(manifest))
				storage.ManifestPut("myrepo", "latest", bytes.NewReader(manifest))
				storage.ManifestPut("myrepo", "stable", bytes.NewReader(manifest))
			},
			expectedTags: []string{"v1.0.0", "v1.0.1", "latest", "stable"},
			wantErr:      false,
		},
		{
			name: "repository with no tags",
			repo: "emptyrepo",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				// Create repository structure but with no tags
				tagsDir := filepath.Join(baseDir, "repositories", "emptyrepo", "_manifests", "tags")
				os.MkdirAll(tagsDir, 0755)
			},
			expectedTags: nil,
			wantErr:      false,
		},
		{
			name: "repository does not exist",
			repo: "nonexistent",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				// Don't create anything
			},
			expectedTags: nil,
			wantErr:      true,
		},
		{
			name: "tags directory with files",
			repo: "myrepo",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				storage.ManifestPut("myrepo", "v1", bytes.NewReader(manifest))
				storage.ManifestPut("myrepo", "v2", bytes.NewReader(manifest))

				// Add some files in the tags directory (should be ignored)
				tagsDir := filepath.Join(baseDir, "repositories", "myrepo", "_manifests", "tags")
				os.WriteFile(filepath.Join(tagsDir, "README.md"), []byte("test"), 0644)
				os.WriteFile(filepath.Join(tagsDir, ".gitkeep"), []byte(""), 0644)
			},
			expectedTags: []string{"v1", "v2"},
			wantErr:      false,
		},
		{
			name: "nested repository tags",
			repo: "org/project",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				storage.ManifestPut("org/project", "v1.0.0", bytes.NewReader(manifest))
				storage.ManifestPut("org/project", "latest", bytes.NewReader(manifest))
			},
			expectedTags: []string{"v1.0.0", "latest"},
			wantErr:      false,
		},
		{
			name: "tags with special characters",
			repo: "myrepo",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				storage.ManifestPut("myrepo", "v1.0.0-alpha", bytes.NewReader(manifest))
				storage.ManifestPut("myrepo", "v2.0.0-beta.1", bytes.NewReader(manifest))
				storage.ManifestPut("myrepo", "release-2024", bytes.NewReader(manifest))
			},
			expectedTags: []string{"v1.0.0-alpha", "v2.0.0-beta.1", "release-2024"},
			wantErr:      false,
		},
		{
			name: "same manifest different tags",
			repo: "myrepo",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				// Same manifest, different tags
				storage.ManifestPut("myrepo", "v1", bytes.NewReader(manifest))
				storage.ManifestPut("myrepo", "v1.0", bytes.NewReader(manifest))
				storage.ManifestPut("myrepo", "v1.0.0", bytes.NewReader(manifest))
				storage.ManifestPut("myrepo", "latest", bytes.NewReader(manifest))
			},
			expectedTags: []string{"v1", "v1.0", "v1.0.0", "latest"},
			wantErr:      false,
		},
		{
			name: "numeric tags",
			repo: "myrepo",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				storage.ManifestPut("myrepo", "1", bytes.NewReader(manifest))
				storage.ManifestPut("myrepo", "2", bytes.NewReader(manifest))
				storage.ManifestPut("myrepo", "10", bytes.NewReader(manifest))
			},
			expectedTags: []string{"1", "2", "10"},
			wantErr:      false,
		},
		{
			name: "mixed case tags",
			repo: "myrepo",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				storage.ManifestPut("myrepo", "Latest", bytes.NewReader(manifest))
				storage.ManifestPut("myrepo", "STABLE", bytes.NewReader(manifest))
				storage.ManifestPut("myrepo", "Dev", bytes.NewReader(manifest))
			},
			expectedTags: []string{"Latest", "STABLE", "Dev"},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			storage := filesystem.NewFilesystemDataStorage(baseDir)

			tt.setupFunc(storage, baseDir)

			tags, err := storage.TagsList(tt.repo)

			if (err != nil) != tt.wantErr {
				t.Errorf("TagsList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Sort both slices for comparison
			if tags != nil {
				slices.Sort(tags)
			}
			expectedSorted := make([]string, len(tt.expectedTags))
			copy(expectedSorted, tt.expectedTags)
			slices.Sort(expectedSorted)

			// Handle nil vs empty slice
			if len(tags) == 0 && len(expectedSorted) == 0 {
				return
			}

			if len(tags) != len(expectedSorted) {
				t.Errorf("TagsList() returned %d tags, want %d. Got: %v, Want: %v",
					len(tags), len(expectedSorted), tags, expectedSorted)
				return
			}

			for i, tag := range tags {
				if tag != expectedSorted[i] {
					t.Errorf("TagsList()[%d] = %s, want %s", i, tag, expectedSorted[i])
				}
			}
		})
	}
}
