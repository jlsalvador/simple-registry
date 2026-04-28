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

package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/rbac"
	"github.com/jlsalvador/simple-registry/pkg/yamlscheme"
)

func TestParseYAML_Valid(t *testing.T) {

	t.Run("parse all supported kinds", func(t *testing.T) {
		t.Parallel()

		data := `
---
apiVersion: ` + apiVersion + `
kind: Token
metadata:
  name: token1
spec:
  value: abc
  username: admin
  expiresAt: 2025-01-01T00:00:00Z

---
apiVersion: ` + apiVersion + `
kind: User
metadata:
  name: admin
spec:
  passwordHash: hash
  groups: [admins]

---
apiVersion: ` + apiVersion + `
kind: Role
metadata:
  name: admins
spec:
  resources: ["*"]
  verbs: ["*"]

---
apiVersion: ` + apiVersion + `
kind: RoleBinding
metadata:
  name: admins-binding
spec:
  subjects:
    - kind: Group
      name: admins
  roleRef:
    name: admins
  scopes: ["^.*$"]
`
		m, err := yamlscheme.DecodeAll(strings.NewReader(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		tokens, users, roles, bindings, err := getTokensUsersRolesRoleBindingsFromManifests(m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tokens) != 1 || tokens[0].Username != "admin" {
			t.Fatalf("token not parsed correctly")
		}

		if len(users) != 1 || users[0].Name != "admin" {
			t.Fatalf("user not parsed")
		}

		if len(roles) != 1 || roles[0].Name != "admins" {
			t.Fatalf("role not parsed")
		}

		if len(bindings) != 1 || bindings[0].RoleName != "admins" {
			t.Fatalf("rolebinding not parsed")
		}
	})
}

func TestParseYAML_Invalid(t *testing.T) {
	t.Run("invalid YAML", func(t *testing.T) {
		t.Parallel()

		data := `
apiVersion: ` + apiVersion + `
kind: User
metadata:
  name: test
spec:
  groups: [`

		_, err := yamlscheme.DecodeAll(strings.NewReader(data))
		if err == nil {
			t.Fatal("expected err")
		}
	})

	t.Run("invalid role verbs", func(t *testing.T) {
		t.Parallel()

		data := `
apiVersion: ` + apiVersion + `
kind: Role
metadata:
  name: bad-role
spec:
  resources: ["*"]
  verbs: ["explode"]
`
		m, err := yamlscheme.DecodeAll(strings.NewReader(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, _, _, _, err = getTokensUsersRolesRoleBindingsFromManifests(m)
		if !errors.Is(err, rbac.ErrInvalidVerb) {
			t.Fatalf("expected rbac.ErrInvalidVerb error: %v", err)
		}
	})

	t.Run("invalid regexp", func(t *testing.T) {
		t.Parallel()

		data := `
apiVersion: ` + apiVersion + `
kind: RoleBinding
metadata:
  name: admins-binding
spec:
  subjects:
    - kind: Group
      name: admins
  roleRef:
    name: admins
  scopes: ["[invalid"]
`
		m, err := yamlscheme.DecodeAll(strings.NewReader(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, _, _, _, err = getTokensUsersRolesRoleBindingsFromManifests(m)
		if err == nil {
			t.Fatalf("expected regexp error")
		}
	})
}

func TestGetProxiesFromManifests(t *testing.T) {
	t.Run("parse valid proxy with string password", func(t *testing.T) {
		data := `
apiVersion: ` + apiVersion + `
kind: PullThroughCache
metadata:
  name: cache
spec:
  upstream:
    url: https://registry.example.com
    username: user1
    password: secretpassword
  scopes: ["library/.*"]
`
		m, err := yamlscheme.DecodeAll(strings.NewReader(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		proxies, err := getProxiesFromManifests(m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(proxies) != 1 {
			t.Fatalf("expected 1 proxy, got %d", len(proxies))
		}
		if proxies[0].Password != "secretpassword" {
			t.Fatalf("expected 'secretpassword', got %v", proxies[0].Password)
		}
	})

	t.Run("parse valid proxy with password file", func(t *testing.T) {
		tmpDir := t.TempDir()
		pwdFile := filepath.Join(tmpDir, "pwd.txt")
		os.WriteFile(pwdFile, []byte("filepassword"), 0o644)

		data := `
apiVersion: ` + apiVersion + `
kind: PullThroughCache
metadata:
  name: cache
spec:
  upstream:
    passwordFile: ` + pwdFile + `
`
		m, err := yamlscheme.DecodeAll(strings.NewReader(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		proxies, err := getProxiesFromManifests(m)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(proxies) != 1 {
			t.Fatalf("expected 1 proxy")
		}
		if proxies[0].Password != "filepassword" {
			t.Fatalf("expected 'filepassword', got %v", proxies[0].Password)
		}
	})

	t.Run("parse proxy with invalid password file", func(t *testing.T) {
		data := `
apiVersion: ` + apiVersion + `
kind: PullThroughCache
metadata:
  name: cache
spec:
  upstream:
    passwordFile: /does/not/exist.txt
`
		m, err := yamlscheme.DecodeAll(strings.NewReader(data))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, err = getProxiesFromManifests(m)
		if err == nil {
			t.Fatal("expected error reading non-existent password file")
		}
	})
}
