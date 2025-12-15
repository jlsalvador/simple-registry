package rbac_test

import (
	"testing"

	"github.com/jlsalvador/simple-registry/internal/rbac"
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
				Verbs:     []rbac.Action{rbac.ActionPull, rbac.ActionPush},
			},
		},
		RoleBindings: []rbac.RoleBinding{
			{
				Name:     "allow-admin",
				Subjects: []rbac.Subject{{Kind: "Group", Name: "admins"}},
				RoleName: "admins",
				Scopes:   []string{"^.*$"},
			},
		},
	}
}

func TestIsAllowed_FullCoverage(t *testing.T) {

	t.Run("user does not exist", func(t *testing.T) {
		engine := baseEngine(t)

		if engine.IsAllowed("ghost", "blobs", "library/busybox", rbac.ActionPull) {
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
				Scopes:   []string{"^.*$"},
			},
		}

		if engine.IsAllowed("admin", "blobs", "library/busybox", rbac.ActionPull) {
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
				Scopes:   []string{"^.*$"},
			},
		}

		if engine.IsAllowed("admin", "blobs", "library/busybox", rbac.ActionPull) {
			t.Fatal("expected access denied")
		}
	})

	t.Run("scope regex invalid", func(t *testing.T) {
		engine := baseEngine(t)
		engine.RoleBindings = []rbac.RoleBinding{
			{
				Name:     "invalid-regex",
				Subjects: []rbac.Subject{{Kind: "Group", Name: "admins"}},
				RoleName: "admins",
				Scopes:   []string{"[invalid"},
			},
		}

		if engine.IsAllowed("admin", "blobs", "library/busybox", rbac.ActionPull) {
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
				Scopes:   []string{"^private/.*$"},
			},
		}

		if engine.IsAllowed("admin", "blobs", "library/busybox", rbac.ActionPull) {
			t.Fatal("expected access denied")
		}
	})

	t.Run("resource not allowed", func(t *testing.T) {
		engine := baseEngine(t)
		engine.Roles = []rbac.Role{
			{
				Name:      "limited",
				Resources: []string{"manifests"},
				Verbs:     []rbac.Action{rbac.ActionPull},
			},
		}
		engine.RoleBindings = []rbac.RoleBinding{
			{
				Name:     "resource-mismatch",
				Subjects: []rbac.Subject{{Kind: "Group", Name: "admins"}},
				RoleName: "limited",
				Scopes:   []string{"^.*$"},
			},
		}

		if engine.IsAllowed("admin", "blobs", "library/busybox", rbac.ActionPull) {
			t.Fatal("expected access denied")
		}
	})

	t.Run("verb not allowed", func(t *testing.T) {
		engine := baseEngine(t)
		engine.Roles = []rbac.Role{
			{
				Name:      "limited",
				Resources: []string{"*"},
				Verbs:     []rbac.Action{rbac.ActionPull},
			},
		}
		engine.RoleBindings = []rbac.RoleBinding{
			{
				Name:     "verb-mismatch",
				Subjects: []rbac.Subject{{Kind: "Group", Name: "admins"}},
				RoleName: "limited",
				Scopes:   []string{"^.*$"},
			},
		}

		if engine.IsAllowed("admin", "blobs", "library/busybox", rbac.ActionPush) {
			t.Fatal("expected access denied")
		}
	})

	t.Run("allowed", func(t *testing.T) {
		engine := baseEngine(t)

		if !engine.IsAllowed("admin", "blobs", "library/busybox", rbac.ActionPush) {
			t.Fatal("expected access allowed")
		}
	})
}
