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
	"errors"
	"strings"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/data/filesystem"
	"github.com/jlsalvador/simple-registry/internal/data/proxy"
	"github.com/jlsalvador/simple-registry/pkg/digest"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func TestWrappers_NilNext(t *testing.T) {
	s := &proxy.ProxyDataStorage{}

	if _, err := s.BlobsUploadCreate("r"); !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("BlobsUploadCreate: expected ErrDataStorageNotInitialized, got %v", err)
	}
	if err := s.BlobsUploadCancel("r", "u"); !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("BlobsUploadCancel: expected ErrDataStorageNotInitialized, got %v", err)
	}
	if err := s.BlobsUploadWrite("r", "u", strings.NewReader(""), 0); !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("BlobsUploadWrite: expected ErrDataStorageNotInitialized, got %v", err)
	}
	if err := s.BlobsUploadCommit("r", "u", "d"); !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("BlobsUploadCommit: expected ErrDataStorageNotInitialized, got %v", err)
	}
	if _, err := s.BlobsUploadSize("r", "u"); !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("BlobsUploadSize: expected ErrDataStorageNotInitialized, got %v", err)
	}
	if err := s.BlobsDelete("r", "d"); !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("BlobsDelete: expected ErrDataStorageNotInitialized, got %v", err)
	}
	if _, err := s.BlobsList(); !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("BlobsList: expected ErrDataStorageNotInitialized, got %v", err)
	}
	if _, err := s.BlobLastAccess("d"); !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("BlobLastAccess: expected ErrDataStorageNotInitialized, got %v", err)
	}
	if err := s.ManifestDelete("r", "ref"); !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("ManifestDelete: expected ErrDataStorageNotInitialized, got %v", err)
	}
	if _, err := s.ManifestsList("r"); !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("ManifestsList: expected ErrDataStorageNotInitialized, got %v", err)
	}
	if _, err := s.ManifestLastAccess("d"); !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("ManifestLastAccess: expected ErrDataStorageNotInitialized, got %v", err)
	}
	if _, err := s.RepositoriesList(); !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("RepositoriesList: expected ErrDataStorageNotInitialized, got %v", err)
	}
	if _, err := s.ManifestPut("r", "ref", strings.NewReader("")); !errors.Is(err, proxy.ErrDataStorageNotInitialized) {
		t.Errorf("ManifestPut: expected ErrDataStorageNotInitialized, got %v", err)
	}
}

func TestWrappers_BlobDelegation(t *testing.T) {
	blob := []byte("hello world")
	hasher, _ := digest.NewHasher("sha256")
	hasher.Write(blob)
	dgst := "sha256:" + hasher.GetHashAsString()
	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	s := proxy.NewProxyDataStorage(storage, nil)

	uuid, err := s.BlobsUploadCreate("r")
	if err != nil || !registry.RegExprUUID.MatchString(uuid) {
		t.Errorf("BlobsUploadCreate: %v %v", uuid, err)
	}
	if err := s.BlobsUploadCancel("r", uuid); err != nil {
		t.Errorf("BlobsUploadCancel: %v", err)
	}

	uuid, err = s.BlobsUploadCreate("r")
	if err != nil || !registry.RegExprUUID.MatchString(uuid) {
		t.Errorf("BlobsUploadCreate: %v %v", uuid, err)
	}
	if err := s.BlobsUploadWrite("r", uuid, bytes.NewReader(blob), 0); err != nil {
		t.Errorf("BlobsUploadWrite: %v", err)
	}
	if size, err := s.BlobsUploadSize("r", uuid); err != nil || size != int64(len(blob)) {
		t.Errorf("BlobsUploadSize: %v %v", size, err)
	}
	if err := s.BlobsUploadCommit("r", uuid, dgst); err != nil {
		t.Errorf("BlobsUploadCommit: %v", err)
	}
	if _, err := s.BlobsList(); err != nil {
		t.Errorf("BlobsList: %v", err)
	}
	if _, err := s.BlobLastAccess(dgst); err != nil {
		t.Errorf("BlobLastAccess: %v", err)
	}
	if err := s.BlobsDelete("r", dgst); err != nil {
		t.Errorf("BlobsDelete: %v", err)
	}
}

func TestWrappers_ManifestAndRepoDelegation(t *testing.T) {
	storage := filesystem.NewFilesystemDataStorage(t.TempDir())
	s := proxy.NewProxyDataStorage(storage, nil)

	dgst, err := s.ManifestPut("r", "ref", strings.NewReader(""))
	if err != nil || !registry.RegExprDigest.MatchString(dgst) {
		t.Errorf("ManifestPut: %v %v", dgst, err)
	}
	if err := s.ManifestDelete("r", "ref"); err != nil {
		t.Errorf("ManifestDelete: %v", err)
	}
	if _, err := s.ManifestsList("r"); err != nil {
		t.Errorf("ManifestsList: %v", err)
	}
	if _, err := s.ManifestLastAccess(dgst); err != nil {
		t.Errorf("ManifestLastAccess: %v", err)
	}
	if repos, err := s.RepositoriesList(); err != nil || len(repos) != 1 {
		t.Errorf("RepositoriesList: %v %v", repos, err)
	}
}
