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

package yaml_test

import (
	"errors"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/rbac"
	"github.com/jlsalvador/simple-registry/pkg/rbac/yaml"
)

func TestParseYAML_FullCoverage(t *testing.T) {

	t.Run("parse all supported kinds", func(t *testing.T) {
		t.Parallel()

		data := `
---
apiVersion: v1
kind: User
metadata:
  name: admin
spec:
  passwordHash: hash
  groups: [admins]

---
apiVersion: v1
kind: Role
metadata:
  name: admins
spec:
  resources: ["*"]
  verbs: ["*"]

---
apiVersion: v1
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

---
apiVersion: v1
kind: Token
metadata:
  name: token1
spec:
  value: abc
  username: admin
  expiresAt: 2025-01-01T00:00:00Z
`

		tokens, users, roles, bindings, err := yaml.ParseYAML([]byte(data))
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

	t.Run("invalid YAML", func(t *testing.T) {
		t.Parallel()

		data := `
apiVersion: v1
kind: User
metadata:
  name: test
spec:
  groups: [`
		_, _, _, _, err := yaml.ParseYAML([]byte(data))
		if !errors.Is(err, yaml.ErrWhileParsing) {
			t.Fatal("expected yaml.ErrWhileParsing")
		}
	})

	t.Run("invalid role verbs", func(t *testing.T) {
		t.Parallel()

		data := `
apiVersion: v1
kind: Role
metadata:
  name: bad-role
spec:
  resources: ["*"]
  verbs: ["explode"]
`
		_, _, _, _, err := yaml.ParseYAML([]byte(data))
		if !errors.Is(err, rbac.ErrInvalidVerb) {
			t.Fatal("expected rbac.ErrActionInvalid")
		}
	})
}
