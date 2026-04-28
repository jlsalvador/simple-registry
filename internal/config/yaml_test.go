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
)

func TestNewWithCfgDir(t *testing.T) {
	tmpDir := t.TempDir()

	cfgYaml := `
apiVersion: ` + apiVersion + `
kind: Configuration
metadata:
  name: test
spec:
  dataDir: ` + tmpDir + `
  web:
    addr: 127.0.0.1:5000
    tokenSecret: super-token-secret
    tokenTimeout: 30
    ui: true
`

	// Valid YAML file.
	validYaml := `
apiVersion: ` + apiVersion + `
kind: User
metadata:
  name: admin
spec:
  passwordHash: $2a$10$GsxTxNCV6Tv9lm9em287xOdRzE7VlbhI0EVRSvZFOq/cCSU6eJuWK # simple-registry
  groups: [admins]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "valid.yaml"), []byte(validYaml), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatal(err)
	}

	// Ignore non-yaml file.
	if err := os.WriteFile(filepath.Join(tmpDir, "ignore.txt"), []byte("not yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Ignore subdirectories (even if they end in .yaml).
	if err := os.Mkdir(filepath.Join(tmpDir, "subdir.yaml"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Run("valid directory parsing", func(t *testing.T) {
		cfg, err := New(WithCfgDirs([]string{tmpDir}))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Rbac.Users) != 1 {
			t.Fatalf("expected 1 user parsed from yaml, got %d", len(cfg.Rbac.Users))
		}
	})

	t.Run("invalid yaml decoding", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected a panic when decoding malformed yaml")
			}
		}()

		badDir := t.TempDir()
		os.WriteFile(filepath.Join(badDir, "bad.yaml"), []byte("invalid: [yaml format"), 0o644)

		_, err := New(WithCfgDirs([]string{badDir}))
		if err == nil {
			t.Fatal("expected error decoding malformed yaml")
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected a panic when config directory does not exist")
			}
		}()

		_, err := New(WithCfgDirs([]string{"/path/does/not/exist"}))
		if err == nil {
			t.Fatal("expected error if cfgdir is missing")
		}
	})

	t.Run("error propagating from GetTokensUsersRolesRoleBindingsFromManifests", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected a panic parsing invalid yaml")
			}
		}()

		badRbacDir := t.TempDir()
		badYaml := `
apiVersion: ` + apiVersion + `
kind: Role
metadata:
  name: bad
spec:
  verbs: ["invalid-verb"]
`
		os.WriteFile(filepath.Join(badRbacDir, "bad-role.yaml"), []byte(badYaml), 0o644)
		_, err := New(WithCfgDirs([]string{badRbacDir}))
		if err == nil {
			t.Fatal("expected error from parsing verbs")
		}
	})

	t.Run("error propagating from GetProxiesFromManifests", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected a panic parsing invalid yaml passwordFile value")
			}
		}()

		badProxyDir := t.TempDir()
		badYaml := `
apiVersion: ` + apiVersion + `
kind: PullThroughCache
metadata:
  name: proxy
spec:
  upstream:
    passwordFile: /path/that/does/not/exist/pwd.txt
`
		os.WriteFile(filepath.Join(badProxyDir, "bad-proxy.yaml"), []byte(badYaml), 0o644)
		_, err := New(WithCfgDirs([]string{badProxyDir}))
		if err == nil {
			t.Fatal("expected error from reading invalid password file")
		}
	})
}

func TestNewWithCfgDirCertAndKeyFiles(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	if err := os.WriteFile(certFile, []byte("cert-content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, []byte("key-content"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfgYaml := `
apiVersion: ` + apiVersion + `
kind: Configuration
metadata:
  name: test
spec:
  dataDir: ` + tmpDir + `
  web:
    addr: 127.0.0.1:5001
    tokenSecret: super-token-secret
    tokenTimeout: 30
    ui: true
    certfile: ` + certFile + `
    keyfile: ` + keyFile + `
`
	if err := os.WriteFile(filepath.Join(tmpDir, "config.yaml"), []byte(cfgYaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := New(WithCfgDirs([]string{tmpDir}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Web.CertFile != certFile {
		t.Fatalf("expected cert file %s, got %s", certFile, cfg.Web.CertFile)
	}
	if cfg.Web.KeyFile != keyFile {
		t.Fatalf("expected key file %s, got %s", keyFile, cfg.Web.KeyFile)
	}
	if cfg.Web.Addr != "127.0.0.1:5001" {
		t.Fatalf("expected http addr 127.0.0.1:5001, got %s", cfg.Web.Addr)
	}
}

func TestNewWithUnreadableYamlFile(t *testing.T) {
	tmpDir := t.TempDir()
	badFile := filepath.Join(tmpDir, "bad.yaml")
	if err := os.WriteFile(badFile, []byte("apiVersion: invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(badFile); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(tmpDir, "missing.yaml"), badFile); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic due missing datadir")
		}
	}()

	New(WithCfgDirs([]string{tmpDir}))
}

func TestParseYamlDirValidManifest(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "user.yaml")
	content := `apiVersion: ` + apiVersion + `
kind: User
metadata:
  name: admin
spec:
  passwordHash: hash
  groups: [admins]
`
	if err := os.WriteFile(yamlFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	manifests, err := parseYamlDir(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(manifests) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(manifests))
	}
}

func TestParseYamlDirUnreadableYamlFile(t *testing.T) {
	tmpDir := t.TempDir()
	badLink := filepath.Join(tmpDir, "bad.yaml")
	if err := os.Symlink(filepath.Join(tmpDir, "missing.yaml"), badLink); err != nil {
		t.Fatal(err)
	}

	manifests, err := parseYamlDir(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(manifests) != 0 {
		t.Fatalf("expected 0 manifests for unreadable yaml file, got %d", len(manifests))
	}
}

func TestParseYamlDirMissingDirectory(t *testing.T) {
	manifests, err := parseYamlDir(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(manifests) != 0 {
		t.Fatalf("expected 0 manifests for missing directory, got %d", len(manifests))
	}
}

func TestParseYamlDirAbsPathError(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "cwd")
	data := filepath.Join(root, "data")
	if err := os.Mkdir(cwd, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(data, 0755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(data, "user.yaml")
	if err := os.WriteFile(filePath, []byte(`apiVersion: `+apiVersion+`
kind: User
metadata:
  name: admin
spec:
  passwordHash: hash
  groups: [admins]
`), 0o644); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	if err := os.Chdir(cwd); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(cwd); err != nil {
		t.Fatal(err)
	}

	_, err = parseYamlDir(filepath.Join("..", "data"))
	if err == nil {
		t.Fatal("expected error from filepath.Abs when cwd is unavailable")
	}
}
