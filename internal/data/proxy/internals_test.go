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
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jlsalvador/simple-registry/internal/data/proxy"
)

func TestMatchProxy_NoMatch(t *testing.T) {
	s := &proxy.ProxyDataStorage{
		Proxies: []proxy.Proxy{{Url: "https://registry.example.com", Scopes: []string{"^other/.*"}}},
	}
	if p := s.MatchProxy("myrepo/myimage"); p != nil {
		t.Errorf("expected nil, got %+v", p)
	}
}

func TestMatchProxy_Match(t *testing.T) {
	s := &proxy.ProxyDataStorage{
		Proxies: []proxy.Proxy{{Url: "https://registry.example.com", Scopes: []string{"^myrepo/.*"}}},
	}
	if p := s.MatchProxy("myrepo/myimage"); p == nil {
		t.Error("expected a proxy match")
	}
}

func TestNewUpstreamRequest_WithAuth(t *testing.T) {
	p := &proxy.Proxy{Username: "u", Password: "p"}
	req, err := proxy.NewUpstreamRequest(p, http.MethodGet, "https://example.com", []string{"application/json"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	u, pw, ok := req.BasicAuth()
	if !ok || u != "u" || pw != "p" {
		t.Error("expected basic auth")
	}
	if req.Header.Get("Accept") == "" {
		t.Error("expected Accept header")
	}
}

func TestNewUpstreamRequest_NoAuthNoAccept(t *testing.T) {
	p := &proxy.Proxy{}
	req, err := proxy.NewUpstreamRequest(p, http.MethodGet, "https://example.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Header.Get("Accept") != "" {
		t.Error("did not expect Accept header")
	}
}

func TestDoUpstreamRequest_Unauthenticated_ThenBearer(t *testing.T) {
	const token = "bearer-token-xyz"

	// Auth server mock that returns a valid token..
	auth := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"token": token})
	}))
	defer auth.Close()

	// Server mock that requires a valid token.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Requires authorization.
		if r.Header.Get("Authorization") == "" {
			w.Header().Set("WWW-Authenticate",
				`Bearer realm="`+auth.URL+`",service="registry",scope="pull"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Requires a valid token.
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	p := &proxy.Proxy{Timeout: 5 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/test", nil)
	resp, err := proxy.DoUpstreamRequest(p, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDoUpstreamRequest_401_BadChallenge(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", "NotBearer")
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := &proxy.Proxy{Timeout: 5 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := proxy.DoUpstreamRequest(p, req)
	if err == nil {
		t.Fatal("expected error for bad WWW-Authenticate")
	}
}

func TestDoUpstreamRequest_401_TokenFetchFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Point to an unreachable token server.
		w.Header().Set("WWW-Authenticate", `Bearer realm="http://127.0.0.1:1"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := &proxy.Proxy{Timeout: 5 * time.Second}
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	_, err := proxy.DoUpstreamRequest(p, req)
	if err == nil {
		t.Fatal("expected error when token fetch fails")
	}
}

func TestFetchManifestFromUpstream_Success(t *testing.T) {
	body := `{"schemaVersion":2}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/manifests/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	rc, size, err := proxy.FetchManifestFromUpstream(p, "repo/img", "latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()
	if size != int64(len(body)) {
		t.Errorf("size mismatch: got %d, want %d", size, len(body))
	}
}

func TestFetchManifestFromUpstream_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	_, _, err := proxy.FetchManifestFromUpstream(p, "repo/img", "latest")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}

func TestFetchManifestFromUpstream_Unauthorized(t *testing.T) {
	// A 401 without a proper Bearer challenge propagates as ErrNotExist
	// (Docker Hub behavior). We need a proper challenge so authentication
	// can succeed; if it still fails we expect ErrUpstreamError.
	// Here we simulate the "always 401" case (no auth server).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", "NotBearer")
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	_, _, err := proxy.FetchManifestFromUpstream(p, "repo/img", "latest")
	// The first 401 triggers auth, which fails with error (not ErrNotExist).
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestFetchManifestFromUpstream_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	_, _, err := proxy.FetchManifestFromUpstream(p, "repo/img", "latest")
	if !errors.Is(err, proxy.ErrUpstreamError) {
		t.Errorf("expected ErrUpstreamError, got %v", err)
	}
}

func TestFetchBlobFromUpstream_Success(t *testing.T) {
	content := []byte("blob-data")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	rc, _, err := proxy.FetchBlobFromUpstream(p, "repo/img", "sha256:deadbeef")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rc.Close()
}

func TestFetchBlobFromUpstream_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	_, _, err := proxy.FetchBlobFromUpstream(p, "repo/img", "sha256:deadbeef")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}

func TestFetchBlobFromUpstream_Unauthorized_BecauseMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("WWW-Authenticate", "NotBearer")
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	_, _, err := proxy.FetchBlobFromUpstream(p, "repo/img", "sha256:deadbeef")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestFetchBlobFromUpstream_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	_, _, err := proxy.FetchBlobFromUpstream(p, "repo/img", "sha256:deadbeef")
	if !errors.Is(err, proxy.ErrUpstreamError) {
		t.Errorf("expected ErrUpstreamError, got %v", err)
	}
}

func TestFetchTagsFromUpstream_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name": "repo/img",
			"tags": []string{"latest", "v1.0"},
		})
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	tags, err := proxy.FetchTagsFromUpstream(p, "repo/img")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}

func TestFetchTagsFromUpstream_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	_, err := proxy.FetchTagsFromUpstream(p, "repo/img")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}

func TestFetchTagsFromUpstream_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	_, err := proxy.FetchTagsFromUpstream(p, "repo/img")
	if !errors.Is(err, proxy.ErrUpstreamError) {
		t.Errorf("expected ErrUpstreamError, got %v", err)
	}
}

func TestFetchTagsFromUpstream_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	_, err := proxy.FetchTagsFromUpstream(p, "repo/img")
	if err == nil {
		t.Fatal("expected JSON decode error")
	}
}

func TestFetchReferrersFromUpstream_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Minimal OCI image index with two referrer manifests.
		payload := `{
			"schemaVersion": 2,
			"mediaType": "application/vnd.oci.image.index.v1+json",
			"manifests": [
				{"digest":"sha256:aaa"},
				{"digest":"sha256:bbb"}
			]
		}`
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	seq, err := proxy.FetchReferrersFromUpstream(p, "repo/img", "sha256:deadbeef")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var digests []string
	for d := range seq {
		digests = append(digests, d)
	}
	if len(digests) != 2 {
		t.Errorf("expected 2 referrer digests, got %d", len(digests))
	}
}

func TestFetchReferrersFromUpstream_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	_, err := proxy.FetchReferrersFromUpstream(p, "repo/img", "sha256:deadbeef")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("expected ErrNotExist, got %v", err)
	}
}

func TestFetchReferrersFromUpstream_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	_, err := proxy.FetchReferrersFromUpstream(p, "repo/img", "sha256:deadbeef")
	if !errors.Is(err, proxy.ErrUpstreamError) {
		t.Errorf("expected ErrUpstreamError, got %v", err)
	}
}

func TestFetchReferrersFromUpstream_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	_, err := proxy.FetchReferrersFromUpstream(p, "repo/img", "sha256:deadbeef")
	if err == nil {
		t.Fatal("expected JSON decode error")
	}
}

func TestFetchReferrersFromUpstream_EarlyYieldBreak(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload := `{
			"schemaVersion": 2,
			"manifests": [
				{"digest":"sha256:aaa"},
				{"digest":"sha256:bbb"},
				{"digest":"sha256:ccc"}
			]
		}`
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	seq, err := proxy.FetchReferrersFromUpstream(p, "repo/img", "sha256:deadbeef")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Break after the first item, exercises the `return` branch in the iterator.
	count := 0
	for range seq {
		count++
		break
	}
	if count != 1 {
		t.Errorf("expected 1 item before break, got %d", count)
	}
}

func TestFetchReferrersFromUpstream_WithCredentials(t *testing.T) {
	var gotUser string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, _, _ = r.BasicAuth()
		_, _ = w.Write([]byte(`{"manifests":[]}`))
	}))
	defer srv.Close()

	p := proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second, Username: "alice", Password: "secret"}
	_, err := proxy.FetchReferrersFromUpstream(p, "repo/img", "sha256:deadbeef")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotUser != "alice" {
		t.Errorf("expected basic auth user 'alice', got %q", gotUser)
	}
}

func TestFetchManifestDigestHEAD_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Docker-Content-Digest", "sha256:cafebabe")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	dgst, err := proxy.FetchManifestDigestHEAD(p, "repo/img", "latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dgst != "sha256:cafebabe" {
		t.Errorf("wrong digest: %s", dgst)
	}
}

func TestFetchManifestDigestHEAD_NonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	_, err := proxy.FetchManifestDigestHEAD(p, "repo/img", "latest")
	if !errors.Is(err, proxy.ErrUpstreamError) {
		t.Errorf("expected ErrUpstreamError, got %v", err)
	}
}

func TestFetchManifestDigestHEAD_MissingDigestHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Intentionally no Docker-Content-Digest header.
	}))
	defer srv.Close()

	p := &proxy.Proxy{Url: srv.URL, Timeout: 5 * time.Second}
	_, err := proxy.FetchManifestDigestHEAD(p, "repo/img", "latest")
	if err == nil {
		t.Fatal("expected error for missing Docker-Content-Digest header")
	}
}
