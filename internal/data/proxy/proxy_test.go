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

package proxy_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/jlsalvador/simple-registry/internal/data"
	"github.com/jlsalvador/simple-registry/internal/data/filesystem"
	"github.com/jlsalvador/simple-registry/internal/data/proxy"
	"github.com/jlsalvador/simple-registry/pkg/digest"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func TestBlobsGet_NilNext(t *testing.T) {
	s := &proxy.ProxyDataStorage{}
	_, _, err := s.BlobsGet("repo", "sha256:abc")
	if !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("expected ErrDataStorageNotInitialized, got %v", err)
	}
}

func TestBlobsGet_LocalHit(t *testing.T) {
	repo := "repo"
	blob := []byte("hello world")
	hasher, _ := digest.NewHasher("sha256")
	hasher.Write(blob)
	digest := "sha256:" + hasher.GetHashAsString()
	storage := filesystem.NewFilesystemDataStorage(t.TempDir())

	// Create local blob.
	uuid, err := storage.BlobsUploadCreate(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.BlobsUploadWrite(repo, uuid, bytes.NewReader(blob), 0); err != nil {
		t.Fatal(err)
	}
	if err := storage.BlobsUploadCommit(repo, uuid, digest); err != nil {
		t.Fatal(err)
	}

	// Test getting blob from local storage through the proxy storage.
	s := proxy.NewProxyDataStorage(storage, nil)
	rc, size, err := s.BlobsGet("repo", digest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()
	if size != int64(len(blob)) {
		t.Errorf("expected size %d, got %d", len(blob), size)
	}
}

func TestBlobsGet_LocalError_NonNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid local link.
	linkPath := filepath.Join(tmpDir, "repositories", "repo", "_layers", "sha256", "abc", "link")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatalf("failed to create directory for link file: %v", err)
	}
	if err := os.WriteFile(linkPath, []byte("invalid"), 0o644); err != nil {
		t.Fatalf("failed to create link file: %v", err)
	}

	storage := filesystem.NewFilesystemDataStorage(tmpDir)
	s := proxy.NewProxyDataStorage(storage, nil)
	_, _, err := s.BlobsGet("repo", "sha256:abc")
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected storage failure error, got %v", err)
	}
}

func TestBlobsGet_NoProxy(t *testing.T) {
	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	// No proxies, default returns fs.ErrNotExist.
	s := proxy.NewProxyDataStorage(storage, nil)
	_, _, err := s.BlobsGet("repo", "sha256:abc")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}

func TestBlobsGet_UpstreamFetch_Success(t *testing.T) {
	blob := []byte("hello world")
	hasher, _ := digest.NewHasher("sha256")
	hasher.Write(blob)
	digest := "sha256:" + hasher.GetHashAsString()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(blob))
	}))
	defer srv.Close()

	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(storage, []proxy.Proxy{p})

	rc, _, err := s.BlobsGet("repo", digest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rc.Close()
}

func TestBlobsGet_UpstreamFetch_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(storage, []proxy.Proxy{p})

	_, _, err := s.BlobsGet("repo", "sha256:abc")
	if !errors.Is(err, proxy.ErrUpstreamError) {
		t.Errorf("expected ErrUpstreamError, got %v", err)
	}
}

func TestBlobsGet_UploadCreate_Fails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data"))
	}))
	defer srv.Close()
	tmpDir := t.TempDir()

	// Create an invalid "_uploads" file to simulate a broken upload session.
	f := filepath.Join(tmpDir, "repositories", "repo", "img", "_uploads")
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f, nil, 0o755); err != nil {
		t.Fatal(err)
	}

	storage := filesystem.NewFilesystemDataStorage(tmpDir)
	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(storage, []proxy.Proxy{p})

	_, _, err := s.BlobsGet("repo", "sha256:abc")
	if err == nil {
		t.Fatal("expected error from BlobsUploadCreate failure")
	}
}

type TestFailOnWriteStorage struct {
	data.DataStorage
}

func (f *TestFailOnWriteStorage) BlobsUploadWrite(repo string, uuid string, r io.Reader, start int64) error {
	return errors.New("simulated BlobsUploadWrite failure")
}

func TestBlobsGet_UploadWrite_Fails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data"))
	}))
	defer srv.Close()

	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	invalidStorage := &TestFailOnWriteStorage{storage}
	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(invalidStorage, []proxy.Proxy{p})

	_, _, err := s.BlobsGet("repo", "sha256:abc")
	if err == nil {
		t.Fatal("expected error from BlobsUploadWrite failure")
	}
}

func TestBlobsGet_UploadCommit_Fails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("data"))
	}))
	defer srv.Close()

	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(storage, []proxy.Proxy{p})

	_, _, err := s.BlobsGet("repo", "sha256:abc")
	if !errors.Is(err, data.ErrDigestMismatch) {
		t.Fatal("expected error ErrDigestMismatch from BlobsUploadCommit failure")
	}
}

func TestManifestGet_NilNext(t *testing.T) {
	s := &proxy.ProxyDataStorage{}
	_, _, _, err := s.ManifestGet("repo", "latest")
	if !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("expected ErrDataStorageNotInitialized, got %v", err)
	}
}

func TestManifestGet_LocalHit_NoProxy(t *testing.T) {
	manifest := []byte("{}")
	hasher, _ := digest.NewHasher("sha256")
	hasher.Write(manifest)
	digest := "sha256:" + hasher.GetHashAsString()
	repo := "repo"
	reference := "latest"

	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	if _, err := storage.ManifestPut(repo, reference, bytes.NewReader(manifest)); err != nil {
		t.Fatal(err)
	}

	s := proxy.NewProxyDataStorage(storage, nil)
	rc, _, dgst, err := s.ManifestGet(repo, reference)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()
	if dgst != digest {
		t.Errorf("wrong digest: %s", dgst)
	}
}

func TestManifestGet_LocalHit_ProxyButDigestMatches(t *testing.T) {
	manifest := []byte("{}")
	hasher, _ := digest.NewHasher("sha256")
	hasher.Write(manifest)
	digest := "sha256:" + hasher.GetHashAsString()
	repo := "repo"
	reference := "latest"

	headSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Docker-Content-Digest", digest)
		w.WriteHeader(http.StatusOK)
		w.Write(manifest)
	}))
	defer headSrv.Close()

	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	p := proxy.Proxy{Url: headSrv.URL, Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(storage, []proxy.Proxy{p})

	rc, _, dgst, err := s.ManifestGet(repo, reference)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()
	if dgst != digest {
		t.Errorf("wrong digest: %s", dgst)
	}
}

func TestManifestGet_LocalHit_ProxyDigestDiffers_FetchesUpstream(t *testing.T) {
	repo := "repo"
	tag := "latest"
	oldManifest := []byte(`{"schemaVersion":2,"annotations":{"test_version":"old"}}`)
	newManifest := []byte(`{"schemaVersion":2,"annotations":{"test_version":"new"}}`)
	hasher, _ := digest.NewHasher("sha256")
	hasher.Write(newManifest)
	newDigest := "sha256:" + hasher.GetHashAsString()

	// Upstream with new manifest.
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Docker-Content-Digest", newDigest)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(newManifest)
	}))
	defer svr.Close()

	storage := filesystem.NewFilesystemDataStorage(t.TempDir())

	// Write old manifest into local.
	if _, err := storage.ManifestPut(repo, tag, bytes.NewReader(oldManifest)); err != nil {
		t.Fatal(err)
	}

	p := proxy.Proxy{Url: svr.URL, Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(storage, []proxy.Proxy{p})

	rc, _, dgst, err := s.ManifestGet(repo, tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()
	if dgst != newDigest {
		t.Errorf("expected new digest, got %s", dgst)
	}
}

func TestManifestGet_LocalMiss_NoProxy(t *testing.T) {
	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	s := proxy.NewProxyDataStorage(storage, nil)
	_, _, _, err := s.ManifestGet("repo", "latest")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}

func TestManifestGet_LocalMiss_ProxyFetchesAndStores(t *testing.T) {
	repo := "repo"
	tag := "latest"
	manifest := []byte(`{"schemaVersion":2}`)
	hasher, _ := digest.NewHasher("sha256")
	hasher.Write(manifest)
	digest := "sha256:" + hasher.GetHashAsString()

	firstCall := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.Header().Set("Docker-Content-Digest", digest)
			return
		}
		if firstCall {
			firstCall = false
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(manifest)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(storage, []proxy.Proxy{p})

	// First call must be returned by the proxy.
	rc, _, _, err := s.ManifestGet(repo, tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rc.Close()

	// Second call must be returned by the local storage.
	rc, _, _, err = s.ManifestGet(repo, tag)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rc.Close()
}

func TestManifestGet_LocalMiss_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(storage, []proxy.Proxy{p})

	_, _, _, err := s.ManifestGet("repo", "latest")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestManifestGet_LocalError_NonNotExist(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid local link for manifest tag.
	linkPath := filepath.Join(tmpDir, "repositories", "repo", "_manifests", "tags", "latest", "current", "link")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatalf("failed to create directory for link file: %v", err)
	}
	if err := os.WriteFile(linkPath, []byte("invalid"), 0o644); err != nil {
		t.Fatalf("failed to create link file: %v", err)
	}

	storage := filesystem.NewFilesystemDataStorage(tmpDir)
	s := proxy.NewProxyDataStorage(storage, nil)
	_, _, _, err := s.ManifestGet("repo", "latest")
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected storage failure error, got %v", err)
	}
}

func TestManifestGet_ManifestPut_Fails(t *testing.T) {
	manifest := []byte(`{"schemaVersion":2}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(manifest)
	}))
	defer srv.Close()

	tmpDir := t.TempDir()

	// Create an invalid "_uploads" file to simulate a broken upload session.
	f := filepath.Join(tmpDir, "repositories", "repo", "_uploads")
	if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f, nil, 0o755); err != nil {
		t.Fatal(err)
	}

	storage := filesystem.NewFilesystemDataStorage(tmpDir)
	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(storage, []proxy.Proxy{p})

	_, _, _, err := s.ManifestGet("repo", "latest")
	if err == nil {
		t.Fatal("expected error from ManifestPut failure")
	}
}

func TestManifestGet_ByDigest_LocalHit(t *testing.T) {
	repo := "repo"
	manifest := []byte(`{"schemaVersion":2}`)
	hasher, _ := digest.NewHasher("sha256")
	hasher.Write(manifest)
	digest := "sha256:" + hasher.GetHashAsString()
	storage := filesystem.NewFilesystemDataStorage(t.TempDir())

	// Write local manifest for local hit by digest.
	if _, err := storage.ManifestPut(repo, digest, bytes.NewReader(manifest)); err != nil {
		t.Fatal(err)
	}

	// When reference is a digest, HEAD is skipped; use local directly.
	p := proxy.Proxy{Url: "http://unused", Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(storage, []proxy.Proxy{p})

	rc, _, dgst, err := s.ManifestGet(repo, digest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()
	if dgst != digest {
		t.Errorf("wrong digest: %s", dgst)
	}
}

func TestReferrersGet_NilNext(t *testing.T) {
	s := &proxy.ProxyDataStorage{}
	_, err := s.ReferrersGet("repo", "latest")
	if !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("expected ErrDataStorageNotInitialized, got %v", err)
	}
}

func TestReferrersGet_LocalHit(t *testing.T) {
	repo := "repo"
	referrerDigest := "sha256:subject1234567890subject1234567890subject1234567890subject12345678"
	storage := filesystem.NewFilesystemDataStorage(t.TempDir())

	manifest := registry.ImageManifest{
		SchemaVersion: 2,
		MediaType:     "application/vnd.oci.image.manifest.v1+json",
		Subject: &registry.DescriptorManifest{
			MediaType: "application/vnd.oci.image.manifest.v1+json",
			Digest:    referrerDigest,
			Size:      999,
		},
	}
	b, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}

	dgst, err := storage.ManifestPut(repo, "latest", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}

	s := proxy.NewProxyDataStorage(storage, nil)
	seq, err := s.ReferrersGet(repo, referrerDigest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got []string
	for d := range seq {
		got = append(got, d)
	}

	want := []string{dgst}
	if slices.Compare(want, got) != 0 {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestReferrersGet_LocalError_NonNotExist(t *testing.T) {
	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	s := proxy.NewProxyDataStorage(storage, nil)
	_, err := s.ReferrersGet("repo", "invalid")
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected storage error, got %v", err)
	}
}

func TestReferrersGet_NoProxy(t *testing.T) {
	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	s := proxy.NewProxyDataStorage(storage, nil)
	_, err := s.ReferrersGet("repo", "sha256:abc")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}

func TestReferrersGet_UpstreamFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"manifests":[{"digest":"sha256:aaa"}]}`))
	}))
	defer srv.Close()

	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(storage, []proxy.Proxy{p})

	seq, err := s.ReferrersGet("repo", "sha256:abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got []string
	for d := range seq {
		got = append(got, d)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 referrer, got %d", len(got))
	}
}

func TestReferrersGet_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(storage, []proxy.Proxy{p})

	_, err := s.ReferrersGet("repo", "latest")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTagsList_NilNext(t *testing.T) {
	s := &proxy.ProxyDataStorage{}
	_, err := s.TagsList("repo")
	if !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("expected ErrDataStorageNotInitialized, got %v", err)
	}
}

func TestTagsList_ProxySuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"name":"repo","tags":["v1","v2"]}`))
	}))
	defer srv.Close()

	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(storage, []proxy.Proxy{p})

	tags, err := s.TagsList("repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestTagsList_ProxyFail_LocalFallback(t *testing.T) {
	repo := "repo"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	storage := filesystem.NewFilesystemDataStorage(t.TempDir())

	// Create local manifests for local fallback.
	want := []string{"dev", "latest"}
	for _, tag := range want {
		if _, err := storage.ManifestPut(repo, tag, bytes.NewReader([]byte(`{"schemaVersion":2}`))); err != nil {
			t.Fatal(err)
		}
	}

	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second, Scopes: []string{".*"}}
	s := proxy.NewProxyDataStorage(storage, []proxy.Proxy{p})

	tags, err := s.TagsList(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != len(want) || slices.Compare(want, tags) != 0 {
		t.Errorf("expected %v, got %v", want, tags)
	}
}

func TestTagsList_NoProxy_LocalResult(t *testing.T) {
	repo := "repo"
	storage := filesystem.NewFilesystemDataStorage(t.TempDir())

	// Create local manifests for local fallback.
	want := []string{"dev", "latest"}
	for _, tag := range want {
		if _, err := storage.ManifestPut(repo, tag, bytes.NewReader([]byte(`{"schemaVersion":2}`))); err != nil {
			t.Fatal(err)
		}
	}

	s := proxy.NewProxyDataStorage(storage, nil)

	tags, err := s.TagsList(repo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != len(want) || slices.Compare(want, tags) != 0 {
		t.Errorf("expected %v, got %v", want, tags)
	}
}
