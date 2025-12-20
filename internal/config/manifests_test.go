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

package config_test

import (
	"strings"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/pkg/yamlscheme"
)

func TestParseYAML_FullCoverage(t *testing.T) {

	t.Run("parse all supported kinds", func(t *testing.T) {
		t.Parallel()

		data := `
---
apiVersion: ` + config.ApiVersion + `
kind: Token
metadata:
  name: token1
spec:
  value: abc
  username: admin
  expiresAt: 2025-01-01T00:00:00Z

---
apiVersion: ` + config.ApiVersion + `
kind: User
metadata:
  name: admin
spec:
  passwordHash: hash
  groups: [admins]

---
apiVersion: ` + config.ApiVersion + `
kind: Role
metadata:
  name: admins
spec:
  resources: ["*"]
  verbs: ["*"]

---
apiVersion: ` + config.ApiVersion + `
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

		tokens, users, roles, bindings := config.GetTokensUsersRolesRoleBindingsFromManifests(m)

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

	t.Run("invalid YAML", func(t *testing.T) {
		t.Parallel()

		data := `
apiVersion: ` + config.ApiVersion + `
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
apiVersion: ` + config.ApiVersion + `
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

		_, _, _, _ = config.GetTokensUsersRolesRoleBindingsFromManifests(m)
	})
}
