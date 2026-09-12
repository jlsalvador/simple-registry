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

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/data/proxy"
)

func writeProxyManifest(t *testing.T, dir string) {
	t.Helper()

	proxyYaml := `
apiVersion: ` + apiVersion + `
kind: PullThroughCache
metadata:
  name: docker-io
spec:
  upstream:
    url: https://registry-1.docker.io
    timeout: 60s
    ttl: 30d
  scopes:
  - ^library/.+$
  - ^jlsalvador/opencode(:.+)?$
`
	if err := os.WriteFile(filepath.Join(dir, "proxies.yaml"), []byte(proxyYaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Regression test: proxies from cfgdir must be wired even when dataDir
// comes from the -datadir flag (no Configuration manifest).
func TestNewProxiesWiredWithFlagDataDir(t *testing.T) {
	newCfgDir := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		writeProxyManifest(t, dir)
		return dir
	}

	t.Run("datadir flag + cfgdir proxies (serve order)", func(t *testing.T) {
		t.Parallel()

		cfgDir := newCfgDir(t)
		dataDir := t.TempDir()

		cfg, err := New(
			WithAdminName("admin"),
			WithAdminPwd([]byte("secret")),
			WithDataDir(dataDir),
			WithCfgDirs([]string{cfgDir}),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ps, ok := cfg.Data.(*proxy.ProxyDataStorage)
		if !ok {
			t.Fatalf("expected *proxy.ProxyDataStorage, got %T (proxies ignored)", cfg.Data)
		}
		if len(ps.Proxies) != 1 {
			t.Fatalf("expected 1 proxy, got %d", len(ps.Proxies))
		}
		if ps.MatchProxy("jlsalvador/opencode") == nil {
			t.Fatal("expected proxy match for jlsalvador/opencode")
		}
		if ps.MatchProxy("library/nginx") == nil {
			t.Fatal("expected proxy match for library/nginx")
		}
		if ps.MatchProxy("other/repo") != nil {
			t.Fatal("expected no proxy match for other/repo")
		}
	})

	t.Run("cfgdir proxies + datadir flag (reverse order)", func(t *testing.T) {
		t.Parallel()

		cfgDir := newCfgDir(t)
		dataDir := t.TempDir()

		cfg, err := New(
			WithAdminName("admin"),
			WithAdminPwd([]byte("secret")),
			WithCfgDirs([]string{cfgDir}),
			WithDataDir(dataDir),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		ps, ok := cfg.Data.(*proxy.ProxyDataStorage)
		if !ok {
			t.Fatalf("expected *proxy.ProxyDataStorage, got %T (proxies ignored)", cfg.Data)
		}
		if len(ps.Proxies) != 1 {
			t.Fatalf("expected 1 proxy, got %d", len(ps.Proxies))
		}
	})

	t.Run("no proxies keeps plain filesystem", func(t *testing.T) {
		t.Parallel()

		emptyDir := t.TempDir()
		dataDir := t.TempDir()

		cfg, err := New(
			WithAdminName("admin"),
			WithAdminPwd([]byte("secret")),
			WithDataDir(dataDir),
			WithCfgDirs([]string{emptyDir}),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, ok := cfg.Data.(*proxy.ProxyDataStorage); ok {
			t.Fatal("expected plain filesystem storage without proxies")
		}
	})
}
