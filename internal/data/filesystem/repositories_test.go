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

func TestRepositoriesList(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(storage *filesystem.FilesystemDataStorage, baseDir string)
		expectedRepos []string
		wantErr       bool
	}{
		{
			name: "list single repository",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				storage.ManifestPut("myrepo", "latest", bytes.NewReader(manifest))
			},
			expectedRepos: []string{"myrepo"},
			wantErr:       false,
		},
		{
			name: "list multiple repositories",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				storage.ManifestPut("repo1", "latest", bytes.NewReader(manifest))
				storage.ManifestPut("repo2", "latest", bytes.NewReader(manifest))
				storage.ManifestPut("repo3", "latest", bytes.NewReader(manifest))
			},
			expectedRepos: []string{"repo1", "repo2", "repo3"},
			wantErr:       false,
		},
		{
			name: "list nested repositories",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				storage.ManifestPut("org/project", "latest", bytes.NewReader(manifest))
				storage.ManifestPut("org/another", "latest", bytes.NewReader(manifest))
				storage.ManifestPut("company/team/app", "latest", bytes.NewReader(manifest))
			},
			expectedRepos: []string{"org/project", "org/another", "company/team/app"},
			wantErr:       false,
		},
		{
			name: "empty repositories directory",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				// Create repositories directory but don't add any repos
				os.MkdirAll(filepath.Join(baseDir, "repositories"), 0755)
			},
			expectedRepos: []string{},
			wantErr:       false,
		},
		{
			name: "repositories directory does not exist",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				// Don't create anything
			},
			expectedRepos: nil,
			wantErr:       true,
		},
		{
			name: "skip internal directories",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				storage.ManifestPut("validrepo", "latest", bytes.NewReader(manifest))

				// Create some internal directories that should be skipped
				reposDir := filepath.Join(baseDir, "repositories", "validrepo")
				os.MkdirAll(filepath.Join(reposDir, "_uploads", "temp"), 0755)
				os.MkdirAll(filepath.Join(reposDir, "_layers", "sha256"), 0755)
				os.MkdirAll(filepath.Join(reposDir, "_links"), 0755)
			},
			expectedRepos: []string{"validrepo"},
			wantErr:       false,
		},
		{
			name: "directories without _manifests are not repositories",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				storage.ManifestPut("realrepo", "latest", bytes.NewReader(manifest))

				// Create a directory that looks like a repo but has no _manifests
				fakerepoPath := filepath.Join(baseDir, "repositories", "fakerepo")
				os.MkdirAll(fakerepoPath, 0755)
				os.WriteFile(filepath.Join(fakerepoPath, "somefile.txt"), []byte("test"), 0644)
			},
			expectedRepos: []string{"realrepo"},
			wantErr:       false,
		},
		{
			name: "nested repos with parent without _manifests",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)

				// Create nested repos
				storage.ManifestPut("org/team/app1", "latest", bytes.NewReader(manifest))
				storage.ManifestPut("org/team/app2", "latest", bytes.NewReader(manifest))

				// Parent directories (org, org/team) should not be listed as repos
			},
			expectedRepos: []string{"org/team/app1", "org/team/app2"},
			wantErr:       false,
		},
		{
			name: "mixed valid and invalid repositories",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				storage.ManifestPut("valid1", "latest", bytes.NewReader(manifest))
				storage.ManifestPut("valid2", "latest", bytes.NewReader(manifest))

				// Create invalid repo (no _manifests)
				invalidPath := filepath.Join(baseDir, "repositories", "invalid")
				os.MkdirAll(invalidPath, 0755)
			},
			expectedRepos: []string{"valid1", "valid2"},
			wantErr:       false,
		},
		{
			name: "repository with special characters in path",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				manifest := createTestManifest(nil)
				storage.ManifestPut("my-org/my-project", "latest", bytes.NewReader(manifest))
				storage.ManifestPut("my.company/app", "latest", bytes.NewReader(manifest))
			},
			expectedRepos: []string{"my-org/my-project", "my.company/app"},
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			storage := filesystem.NewFilesystemDataStorage(baseDir)

			tt.setupFunc(storage, baseDir)

			repos, err := storage.RepositoriesList()

			if (err != nil) != tt.wantErr {
				t.Errorf("RepositoriesList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Sort both slices for comparison
			slices.Sort(repos)
			expectedSorted := make([]string, len(tt.expectedRepos))
			copy(expectedSorted, tt.expectedRepos)
			slices.Sort(expectedSorted)

			if len(repos) != len(expectedSorted) {
				t.Errorf("RepositoriesList() returned %d repos, want %d. Got: %v, Want: %v",
					len(repos), len(expectedSorted), repos, expectedSorted)
				return
			}

			for i, repo := range repos {
				if repo != expectedSorted[i] {
					t.Errorf("RepositoriesList()[%d] = %s, want %s", i, repo, expectedSorted[i])
				}
			}
		})
	}
}

func TestRepositoriesListSkipAll(t *testing.T) {
	baseDir := t.TempDir()
	storage := filesystem.NewFilesystemDataStorage(baseDir)

	// Create a repository
	manifest := createTestManifest(nil)
	storage.ManifestPut("myrepo", "latest", bytes.NewReader(manifest))

	// Create directories with reserved names at different levels
	reposDir := filepath.Join(baseDir, "repositories")
	os.MkdirAll(filepath.Join(reposDir, "_manifests"), 0755)
	os.MkdirAll(filepath.Join(reposDir, "_layers"), 0755)
	os.MkdirAll(filepath.Join(reposDir, "_uploads"), 0755)
	os.MkdirAll(filepath.Join(reposDir, "_links"), 0755)
	os.MkdirAll(filepath.Join(reposDir, "myrepo", "_uploads", "test"), 0755)

	repos, err := storage.RepositoriesList()
	if err != nil {
		t.Fatalf("RepositoriesList() error = %v", err)
	}

	// Should only find "myrepo", all _* directories should be skipped
	if len(repos) != 1 || repos[0] != "myrepo" {
		t.Errorf("expected [myrepo], got %v", repos)
	}
}
