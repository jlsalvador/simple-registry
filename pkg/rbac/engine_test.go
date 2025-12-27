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

package rbac_test

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/rbac"
)

func baseEngine(t *testing.T) rbac.Engine {
	t.Helper()
	return rbac.Engine{
		Users: []rbac.User{
			{Name: "admin", Groups: []string{"admins"}},
		},
		Roles: []rbac.Role{
			{
				Name:      "admins",
				Resources: []string{"*"},
				Verbs: []string{
					http.MethodHead,
					http.MethodGet,
					http.MethodPost,
					http.MethodPut,
					http.MethodPatch,
				},
			},
		},
		RoleBindings: []rbac.RoleBinding{
			{
				Name:     "allow-admin",
				Subjects: []rbac.Subject{{Kind: "Group", Name: "admins"}},
				RoleName: "admins",
				Scopes:   []regexp.Regexp{*regexp.MustCompile("^.*$")},
			},
		},
	}
}

func TestIsAllowed_FullCoverage(t *testing.T) {

	t.Run("user does not exist", func(t *testing.T) {
		engine := baseEngine(t)

		if engine.IsAllowed("ghost", "blobs", "library/busybox", http.MethodGet) {
			t.Fatal("expected access denied")
		}
	})

	t.Run("role does not exist", func(t *testing.T) {
		engine := baseEngine(t)
		engine.RoleBindings = []rbac.RoleBinding{
			{
				Name:     "invalid-role",
				Subjects: []rbac.Subject{{Kind: "Group", Name: "admins"}},
				RoleName: "missing",
				Scopes:   []regexp.Regexp{*regexp.MustCompile("^.*$")},
			},
		}

		if engine.IsAllowed("admin", "blobs", "library/busybox", http.MethodGet) {
			t.Fatal("expected access denied")
		}
	})

	t.Run("subject mismatch", func(t *testing.T) {
		engine := baseEngine(t)
		engine.RoleBindings = []rbac.RoleBinding{
			{
				Name:     "wrong-subject",
				Subjects: []rbac.Subject{{Kind: "User", Name: "someone"}},
				RoleName: "admins",
				Scopes:   []regexp.Regexp{*regexp.MustCompile("^.*$")},
			},
		}

		if engine.IsAllowed("admin", "blobs", "library/busybox", http.MethodGet) {
			t.Fatal("expected access denied")
		}
	})

	t.Run("scope does not match repo", func(t *testing.T) {
		engine := baseEngine(t)
		engine.RoleBindings = []rbac.RoleBinding{
			{
				Name:     "scope-no-match",
				Subjects: []rbac.Subject{{Kind: "Group", Name: "admins"}},
				RoleName: "admins",
				Scopes:   []regexp.Regexp{*regexp.MustCompile("^private/.*$")},
			},
		}

		if engine.IsAllowed("admin", "blobs", "library/busybox", http.MethodGet) {
			t.Fatal("expected access denied")
		}
	})

	t.Run("resource not allowed", func(t *testing.T) {
		engine := baseEngine(t)
		engine.Roles = []rbac.Role{
			{
				Name:      "limited",
				Resources: []string{"manifests"},
				Verbs: []string{
					http.MethodHead,
					http.MethodGet,
				},
			},
		}
		engine.RoleBindings = []rbac.RoleBinding{
			{
				Name:     "resource-mismatch",
				Subjects: []rbac.Subject{{Kind: "Group", Name: "admins"}},
				RoleName: "limited",
				Scopes:   []regexp.Regexp{*regexp.MustCompile("^.*$")},
			},
		}

		if engine.IsAllowed("admin", "blobs", "library/busybox", http.MethodGet) {
			t.Fatal("expected access denied")
		}
	})

	t.Run("verb not allowed", func(t *testing.T) {
		engine := baseEngine(t)
		engine.Roles = []rbac.Role{
			{
				Name:      "limited",
				Resources: []string{"*"},
				Verbs: []string{
					http.MethodHead,
					http.MethodGet,
				},
			},
		}
		engine.RoleBindings = []rbac.RoleBinding{
			{
				Name:     "verb-mismatch",
				Subjects: []rbac.Subject{{Kind: "Group", Name: "admins"}},
				RoleName: "limited",
				Scopes:   []regexp.Regexp{*regexp.MustCompile("^.*$")},
			},
		}

		if engine.IsAllowed("admin", "blobs", "library/busybox", http.MethodPost) {
			t.Fatal("expected access denied")
		}
	})

	t.Run("allowed", func(t *testing.T) {
		engine := baseEngine(t)

		if !engine.IsAllowed("admin", "blobs", "library/busybox", http.MethodPost) {
			t.Fatal("expected access allowed")
		}
	})
}
