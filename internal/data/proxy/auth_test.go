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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/data/proxy"
)

func TestParseBearerChallenge_NotBearer(t *testing.T) {
	_, err := proxy.ParseBearerChallenge("Basic realm=\"test\"")
	if err == nil {
		t.Fatal("expected error for non-bearer challenge")
	}
}

func TestParseBearerChallenge_MissingRealm(t *testing.T) {
	_, err := proxy.ParseBearerChallenge(`Bearer service="registry"`)
	if err == nil {
		t.Fatal("expected error when realm is missing")
	}
}

func TestParseBearerChallenge_MalformedKV(t *testing.T) {
	// "noequalssign" has no '=' so it will be skipped; realm is also absent.
	_, err := proxy.ParseBearerChallenge(`Bearer noequalssign`)
	if err == nil {
		t.Fatal("expected error for malformed key-value pair with no realm")
	}
}

func TestParseBearerChallenge_Success(t *testing.T) {
	ch, err := proxy.ParseBearerChallenge(`Bearer realm="https://auth.example.com",service="reg",scope="pull"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch.Realm != "https://auth.example.com" {
		t.Errorf("wrong realm: %s", ch.Realm)
	}
	if ch.Service != "reg" {
		t.Errorf("wrong service: %s", ch.Service)
	}
	if ch.Scope != "pull" {
		t.Errorf("wrong scope: %s", ch.Scope)
	}
}

func TestParseBearerChallenge_RealmOnly(t *testing.T) {
	ch, err := proxy.ParseBearerChallenge(`Bearer realm="https://auth.example.com"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch.Realm != "https://auth.example.com" {
		t.Errorf("wrong realm: %s", ch.Realm)
	}
	if ch.Service != "" || ch.Scope != "" {
		t.Error("expected empty service and scope")
	}
}

func TestFetchBearerToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "mytoken"})
	}))
	defer srv.Close()

	p := &proxy.Proxy{}
	ch := &proxy.BearerChallenge{Realm: srv.URL, Service: "reg", Scope: "pull"}
	tok, err := proxy.FetchBearerToken(p, ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "mytoken" {
		t.Errorf("wrong token: %s", tok)
	}
}

func TestFetchBearerToken_UsesAccessTokenFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "accesstok"})
	}))
	defer srv.Close()

	p := &proxy.Proxy{}
	ch := &proxy.BearerChallenge{Realm: srv.URL}
	tok, err := proxy.FetchBearerToken(p, ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "accesstok" {
		t.Errorf("wrong token: %s", tok)
	}
}

func TestFetchBearerToken_WithCredentials(t *testing.T) {
	var gotUser, gotPass string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, _ = r.BasicAuth()
		_ = json.NewEncoder(w).Encode(map[string]string{"token": "tok"})
	}))
	defer srv.Close()

	p := &proxy.Proxy{Username: "user", Password: "pass"}
	ch := &proxy.BearerChallenge{Realm: srv.URL}
	if _, err := proxy.FetchBearerToken(p, ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotUser != "user" || gotPass != "pass" {
		t.Errorf("expected basic auth user/pass, got %q/%q", gotUser, gotPass)
	}
}

func TestFetchBearerToken_NonOKStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	p := &proxy.Proxy{}
	ch := &proxy.BearerChallenge{Realm: srv.URL}
	_, err := proxy.FetchBearerToken(p, ch)
	if err == nil {
		t.Fatal("expected error for non-200 token response")
	}
}

func TestFetchBearerToken_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not-json"))
	}))
	defer srv.Close()

	p := &proxy.Proxy{}
	ch := &proxy.BearerChallenge{Realm: srv.URL}
	_, err := proxy.FetchBearerToken(p, ch)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestFetchBearerToken_BadRealm(t *testing.T) {
	p := &proxy.Proxy{}
	ch := &proxy.BearerChallenge{Realm: "://bad url"}
	_, err := proxy.FetchBearerToken(p, ch)
	if err == nil {
		t.Fatal("expected error for bad realm URL")
	}
}
