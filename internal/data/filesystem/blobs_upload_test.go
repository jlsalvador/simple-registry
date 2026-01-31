// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/data"
	"github.com/jlsalvador/simple-registry/internal/data/filesystem"
	"github.com/jlsalvador/simple-registry/pkg/digest"
)

func TestBlobsUploadCreate(t *testing.T) {
	tmpdir := t.TempDir()
	fs := filesystem.NewFilesystemDataStorage(tmpdir)

	// Invalid repo
	if _, err := fs.BlobsUploadCreate("THIS IS INVALID"); err == nil {
		t.Fatal("expected error for invalid repo")
	}

	uuid, err := fs.BlobsUploadCreate("repo")
	if err != nil {
		t.Fatal(err)
	}

	// Should exist
	p := filepath.Join(tmpdir, "repositories/repo/_uploads", uuid)
	if _, err := os.Stat(p); err != nil {
		t.Fatal("upload folder not created")
	}
}

func TestBlobsUploadCancel(t *testing.T) {
	tmpdir := t.TempDir()
	fs := filesystem.NewFilesystemDataStorage(tmpdir)

	// invalid repo
	if err := fs.BlobsUploadCancel("INVALID REPO", "abc"); err == nil {
		t.Fatal("expected repo invalid error")
	}

	// invalid uuid
	if err := fs.BlobsUploadCancel("repo", "not-uuid"); err == nil {
		t.Fatal("expected uuid invalid error")
	}

	// valid cancel
	id, _ := fs.BlobsUploadCreate("repo")
	if err := fs.BlobsUploadCancel("repo", id); err != nil {
		t.Fatal(err)
	}
}

func TestBlobsUploadWriteAndCommit(t *testing.T) {
	tmpdir := t.TempDir()
	fs := filesystem.NewFilesystemDataStorage(tmpdir)

	uploadID, err := fs.BlobsUploadCreate("repo")
	if err != nil {
		t.Fatal(err)
	}

	// invalid repo
	if err := fs.BlobsUploadWrite("INVALID REPO", uploadID, bytes.NewBufferString("data"), -1); err == nil {
		t.Fatal("expected error for invalid repo")
	}

	// invalid uuid
	if err := fs.BlobsUploadWrite("repo", "not-uuid", bytes.NewBufferString("data"), -1); err == nil {
		t.Fatal("expected uuid error")
	}

	// write start < 0
	if err := fs.BlobsUploadWrite("repo", uploadID, bytes.NewBufferString("hello"), -1); err != nil {
		t.Fatal(err)
	}

	// write start >= 0 (overwrite start)
	if err := fs.BlobsUploadWrite("repo", uploadID, bytes.NewBufferString("X"), 0); err != nil {
		t.Fatal(err)
	}

	// check written content
	dataPath := filepath.Join(tmpdir, "repositories/repo/_uploads", uploadID, "data")
	b, _ := os.ReadFile(dataPath)
	if string(b) != "Xello" {
		t.Fatalf("unexpected upload content: %s", string(b))
	}

	// ----------------------------
	// Test Commit
	// ----------------------------
	hasher, _ := digest.NewHasher("sha256")
	hasher.Write([]byte("Xello"))
	d := "sha256:" + hasher.GetHashAsString()

	// invalid repo
	if err := fs.BlobsUploadCommit("INVALID", uploadID, d); err == nil {
		t.Fatal("expected repo error")
	}

	// invalid uuid
	if err := fs.BlobsUploadCommit("repo", "invalid", d); err == nil {
		t.Fatal("expected uuid error")
	}

	// invalid digest
	if err := fs.BlobsUploadCommit("repo", uploadID, "xxx"); err == nil {
		t.Fatal("expected digest parse error")
	}

	// wrong digest
	hasher, _ = digest.NewHasher("sha256")
	hasher.Write([]byte("another"))
	wrong := "sha256:" + hasher.GetHashAsString()
	if err := fs.BlobsUploadCommit("repo", uploadID, wrong); !errors.Is(err, data.ErrDigestMismatch) {
		t.Fatal("expected digest mismatch")
	}

	// short digest hash
	if err := fs.BlobsUploadCommit("repo", uploadID, "sha256:a"); !errors.Is(err, data.ErrHashShort) {
		t.Fatal("expected hash short")
	}

	// correct digest commit
	if err := fs.BlobsUploadCommit("repo", uploadID, d); err != nil {
		t.Fatal(err)
	}

	// blob should exist
	a, h, _ := digest.Parse(d)
	blobPath := filepath.Join(tmpdir, "blobs", a, h[:2], h)
	if _, err := os.Stat(blobPath); err != nil {
		t.Fatal("blob not committed")
	}

	// link should exist
	link := filepath.Join(tmpdir, "repositories/repo/_layers", a, h, "link")
	if _, err := os.Stat(link); err != nil {
		t.Fatal("link not written")
	}
}

func TestBlobsUploadSize(t *testing.T) {
	tmpdir := t.TempDir()
	fs := filesystem.NewFilesystemDataStorage(tmpdir)

	repo := "repo"
	uuid, err := fs.BlobsUploadCreate(repo)
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("hello world")
	if err := fs.BlobsUploadWrite(repo, uuid, bytes.NewReader(data), 0); err != nil {
		t.Fatal(err)
	}

	if size, err := fs.BlobsUploadSize(repo, uuid); err != nil {
		t.Fatal(err)
	} else if size != int64(len(data)) {
		t.Fatalf("wrong size: %d", size)
	}
}
