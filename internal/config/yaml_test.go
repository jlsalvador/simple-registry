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

package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/config"
)

func TestNewFromYamlDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Valid YAML file.
	validYaml := `
apiVersion: ` + config.ApiVersion + `
kind: User
metadata:
  name: admin
spec:
  groups: [admins]
`
	err := os.WriteFile(filepath.Join(tmpDir, "valid.yaml"), []byte(validYaml), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Ignore non-yaml file.
	err = os.WriteFile(filepath.Join(tmpDir, "ignore.txt"), []byte("not yaml"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// Ignore subdirectories (even if they end in .yaml).
	err = os.Mkdir(filepath.Join(tmpDir, "subdir.yaml"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid directory parsing", func(t *testing.T) {
		cfg, err := config.NewFromYamlDir([]string{tmpDir}, t.TempDir())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Rbac.Users) != 1 {
			t.Fatalf("expected 1 user parsed from yaml, got %d", len(cfg.Rbac.Users))
		}
	})

	t.Run("invalid yaml decoding", func(t *testing.T) {
		badDir := t.TempDir()
		os.WriteFile(filepath.Join(badDir, "bad.yaml"), []byte("invalid: [yaml format"), 0644)

		_, err := config.NewFromYamlDir([]string{badDir}, t.TempDir())
		if err == nil {
			t.Fatal("expected error decoding malformed yaml")
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		// os.ReadDir fails silently in parseYamlDir, returning nil, nil.
		cfg, err := config.NewFromYamlDir([]string{"/path/does/not/exist"}, t.TempDir())
		if err != nil {
			t.Fatalf("did not expect error, got %v", err)
		}
		if cfg == nil {
			t.Fatal("expected valid config even if dir is missing")
		}
	})

	t.Run("error propagating from GetTokensUsersRolesRoleBindingsFromManifests", func(t *testing.T) {
		badRbacDir := t.TempDir()
		badYaml := `
apiVersion: ` + config.ApiVersion + `
kind: Role
metadata:
  name: bad
spec:
  verbs: ["invalid-verb"]
`
		os.WriteFile(filepath.Join(badRbacDir, "bad-role.yaml"), []byte(badYaml), 0644)
		_, err := config.NewFromYamlDir([]string{badRbacDir}, t.TempDir())
		if err == nil {
			t.Fatal("expected error from parsing verbs")
		}
	})

	t.Run("error propagating from GetProxiesFromManifests", func(t *testing.T) {
		badProxyDir := t.TempDir()
		badYaml := `
apiVersion: ` + config.ApiVersion + `
kind: PullThroughCache
metadata:
  name: proxy
spec:
  upstream:
    passwordFile: /path/that/does/not/exist/pwd.txt
`
		os.WriteFile(filepath.Join(badProxyDir, "bad-proxy.yaml"), []byte(badYaml), 0644)
		_, err := config.NewFromYamlDir([]string{badProxyDir}, t.TempDir())
		if err == nil {
			t.Fatal("expected error from reading invalid password file")
		}
	})
}
