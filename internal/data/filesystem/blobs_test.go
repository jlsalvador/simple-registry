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
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jlsalvador/simple-registry/internal/data/filesystem"
)

func createTestBlob(t *testing.T, baseDir, algo, hash, content string) {
	t.Helper()
	blobPath := filepath.Join(baseDir, "blobs", algo, hash[0:2], hash)
	if err := os.MkdirAll(filepath.Dir(blobPath), 0755); err != nil {
		t.Fatalf("failed to create blob dir: %v", err)
	}
	if err := os.WriteFile(blobPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write blob: %v", err)
	}
}

func createTestRepoLink(t *testing.T, baseDir, repo, algo, hash, digest string) {
	t.Helper()
	linkPath := filepath.Join(baseDir, "repositories", repo, "_layers", algo, hash, "link")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		t.Fatalf("failed to create link dir: %v", err)
	}
	if err := os.WriteFile(linkPath, []byte(digest), 0644); err != nil {
		t.Fatalf("failed to write link: %v", err)
	}
}

func TestBlobsGet(t *testing.T) {
	tests := []struct {
		name        string
		repo        string
		digest      string
		setupFunc   func(storage *filesystem.FilesystemDataStorage, baseDir string)
		wantContent string
		wantSize    int64
		wantErr     bool
	}{
		{
			name:   "get blob without repository",
			repo:   "",
			digest: "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				createTestBlob(t, baseDir, "sha256", "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", "test content")
			},
			wantContent: "test content",
			wantSize:    12,
			wantErr:     false,
		},
		{
			name:   "get blob with repository link",
			repo:   "myrepo",
			digest: "sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				hash := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
				digest := "sha256:" + hash
				createTestBlob(t, baseDir, "sha256", hash, "repository content")
				createTestRepoLink(t, baseDir, "myrepo", "sha256", hash, digest)
			},
			wantContent: "repository content",
			wantSize:    18,
			wantErr:     false,
		},
		{
			name:      "invalid digest format",
			repo:      "",
			digest:    "invalid",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {},
			wantErr:   true,
		},
		{
			name:      "hash too short",
			repo:      "",
			digest:    "sha256:a",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {},
			wantErr:   true,
		},
		{
			name:   "missing repository link",
			repo:   "myrepo",
			digest: "sha256:9876543210abcdef9876543210abcdef9876543210abcdef9876543210abcdef",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				createTestBlob(t, baseDir, "sha256", "9876543210abcdef9876543210abcdef9876543210abcdef9876543210abcdef", "test")
			},
			wantErr: true,
		},
		{
			name:   "link digest mismatch",
			repo:   "myrepo",
			digest: "sha256:fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				hash := "fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321"
				createTestBlob(t, baseDir, "sha256", hash, "test")
				createTestRepoLink(t, baseDir, "myrepo", "sha256", hash, "sha256:wrongdigest")
			},
			wantErr: true,
		},
		{
			name:      "blob does not exist",
			repo:      "",
			digest:    "sha256:nonexistent1234567890nonexistent1234567890nonexistent1234567890abcd",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			storage := filesystem.NewFilesystemDataStorage(baseDir)

			tt.setupFunc(storage, baseDir)

			r, size, err := storage.BlobsGet(tt.repo, tt.digest)
			if (err != nil) != tt.wantErr {
				t.Errorf("BlobsGet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			defer r.Close()

			if size != tt.wantSize {
				t.Errorf("BlobsGet() size = %v, want %v", size, tt.wantSize)
			}

			content, err := io.ReadAll(r)
			if err != nil {
				t.Fatalf("failed to read blob content: %v", err)
			}

			if string(content) != tt.wantContent {
				t.Errorf("BlobsGet() content = %v, want %v", string(content), tt.wantContent)
			}
		})
	}
}

func TestBlobsDelete(t *testing.T) {
	tests := []struct {
		name      string
		repo      string
		digest    string
		setupFunc func(storage *filesystem.FilesystemDataStorage, baseDir string)
		checkFunc func(t *testing.T, baseDir string)
		wantErr   bool
	}{
		{
			name:   "delete blob without repository",
			repo:   "",
			digest: "sha256:abc123def456abc123def456abc123def456abc123def456abc123def456abc1",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				createTestBlob(t, baseDir, "sha256", "abc123def456abc123def456abc123def456abc123def456abc123def456abc1", "test")
			},
			checkFunc: func(t *testing.T, baseDir string) {
				blobPath := filepath.Join(baseDir, "blobs", "sha256", "ab", "abc123def456abc123def456abc123def456abc123def456abc123def456abc1")
				if _, err := os.Stat(blobPath); !os.IsNotExist(err) {
					t.Error("blob should be deleted")
				}
			},
			wantErr: false,
		},
		{
			name:   "delete repository link only",
			repo:   "myrepo",
			digest: "sha256:def456abc123def456abc123def456abc123def456abc123def456abc123def4",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				hash := "def456abc123def456abc123def456abc123def456abc123def456abc123def4"
				digest := "sha256:" + hash
				createTestBlob(t, baseDir, "sha256", hash, "test")
				createTestRepoLink(t, baseDir, "myrepo", "sha256", hash, digest)
			},
			checkFunc: func(t *testing.T, baseDir string) {
				linkPath := filepath.Join(baseDir, "repositories", "myrepo", "_layers", "sha256", "def456abc123def456abc123def456abc123def456abc123def456abc123def4")
				if _, err := os.Stat(linkPath); !os.IsNotExist(err) {
					t.Error("repository link should be deleted")
				}
				// Blob should still exist
				blobPath := filepath.Join(baseDir, "blobs", "sha256", "de", "def456abc123def456abc123def456abc123def456abc123def456abc123def4")
				if _, err := os.Stat(blobPath); err != nil {
					t.Error("blob should still exist")
				}
			},
			wantErr: false,
		},
		{
			name:      "invalid digest format",
			repo:      "",
			digest:    "invalid",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {},
			wantErr:   true,
		},
		{
			name:      "hash too short",
			repo:      "",
			digest:    "sha256:a",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			storage := filesystem.NewFilesystemDataStorage(baseDir)

			tt.setupFunc(storage, baseDir)

			err := storage.BlobsDelete(tt.repo, tt.digest)
			if (err != nil) != tt.wantErr {
				t.Errorf("BlobsDelete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, baseDir)
			}
		})
	}
}

func TestBlobsList(t *testing.T) {
	baseDir := t.TempDir()
	storage := filesystem.NewFilesystemDataStorage(baseDir)

	// Create multiple blobs
	createTestBlob(t, baseDir, "sha256", "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", "blob1")
	createTestBlob(t, baseDir, "sha256", "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", "blob2")
	createTestBlob(t, baseDir, "sha512", "fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321", "blob3")

	digests, err := storage.BlobsList()
	if err != nil {
		t.Fatalf("BlobsList() error = %v", err)
	}

	found := make(map[string]bool)
	for digest := range digests {
		found[digest] = true
	}

	expectedDigests := []string{
		"sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"sha256:1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		"sha512:fedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321",
	}

	if len(found) != len(expectedDigests) {
		t.Errorf("BlobsList() returned %d digests, want %d", len(found), len(expectedDigests))
	}

	for _, expected := range expectedDigests {
		if !found[expected] {
			t.Errorf("BlobsList() missing digest %s", expected)
		}
	}
}

func TestBlobsListEmpty(t *testing.T) {
	baseDir := t.TempDir()
	storage := filesystem.NewFilesystemDataStorage(baseDir)

	// Create blobs directory but leave it empty
	os.MkdirAll(filepath.Join(baseDir, "blobs"), 0755)

	digests, err := storage.BlobsList()
	if err != nil {
		t.Fatalf("BlobsList() error = %v", err)
	}

	count := 0
	for range digests {
		count++
	}

	if count != 0 {
		t.Errorf("BlobsList() returned %d digests, want 0", count)
	}
}

func TestBlobLastAccess(t *testing.T) {
	tests := []struct {
		name      string
		digest    string
		setupFunc func(storage *filesystem.FilesystemDataStorage, baseDir string)
		wantErr   bool
	}{
		{
			name:   "valid blob",
			digest: "sha256:abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {
				createTestBlob(t, baseDir, "sha256", "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", "test")
			},
			wantErr: false,
		},
		{
			name:      "invalid digest format",
			digest:    "invalid",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {},
			wantErr:   true,
		},
		{
			name:      "hash too short",
			digest:    "sha256:a",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {},
			wantErr:   true,
		},
		{
			name:      "blob does not exist",
			digest:    "sha256:nonexistent1234567890nonexistent1234567890nonexistent1234567890abcd",
			setupFunc: func(storage *filesystem.FilesystemDataStorage, baseDir string) {},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			storage := filesystem.NewFilesystemDataStorage(baseDir)

			tt.setupFunc(storage, baseDir)

			lastAccess, err := storage.BlobLastAccess(tt.digest)
			if (err != nil) != tt.wantErr {
				t.Errorf("BlobLastAccess() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Check that the returned time is reasonable (within the last minute)
				now := time.Now()
				diff := now.Sub(lastAccess)
				if diff < 0 || diff > time.Minute {
					t.Errorf("BlobLastAccess() returned unexpected time: %v (diff from now: %v)", lastAccess, diff)
				}
			}
		})
	}
}

func TestBlobsListYieldBreak(t *testing.T) {
	baseDir := t.TempDir()
	storage := filesystem.NewFilesystemDataStorage(baseDir)

	// Create multiple blobs
	createTestBlob(t, baseDir, "sha256", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "blob1")
	createTestBlob(t, baseDir, "sha256", "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "blob2")
	createTestBlob(t, baseDir, "sha256", "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "blob3")

	digests, err := storage.BlobsList()
	if err != nil {
		t.Fatalf("BlobsList() error = %v", err)
	}

	// Only iterate once and break
	count := 0
	for range digests {
		count++
		if count >= 1 {
			break
		}
	}

	if count != 1 {
		t.Errorf("Expected to iterate only once, but iterated %d times", count)
	}
}
