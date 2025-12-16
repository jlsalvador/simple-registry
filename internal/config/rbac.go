package config

import (
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/jlsalvador/simple-registry/internal/rbac"
	"github.com/jlsalvador/simple-registry/internal/rbac/yaml"
	"golang.org/x/crypto/bcrypt"
)

func GetRBACEngineStatic(adminName, adminPwd string) rbac.Engine {
	return rbac.Engine{
		Users: []rbac.User{
			{
				// Administrator.
				Name: adminName,
				PasswordHash: func() string {
					pwd, _ := bcrypt.GenerateFromPassword([]byte(adminPwd), bcrypt.DefaultCost)
					return string(pwd)
				}(),
				Groups: []string{"admins", "public"},
			},
			// {
			// 	// Anonymous.
			// 	Name:   "",
			// 	Groups: []string{"public"},
			// },
		},
		Roles: []rbac.Role{
			// Write
			{
				Name:      "write",
				Resources: []string{"*"},
				Verbs: []string{
					http.MethodHead,
					http.MethodGet,
					http.MethodPost,
					http.MethodPut,
					http.MethodPatch,
					http.MethodDelete,
				},
			},
			// Read-Only
			{
				Name:      "readonly",
				Resources: []string{"*"},
				Verbs: []string{
					http.MethodHead,
					http.MethodGet,
				},
			},
		},
		RoleBindings: []rbac.RoleBinding{
			// {
			// 	Name: "public",
			// 	Subjects: []rbac.Subject{
			// 		{
			// 			Kind: "Group",
			// 			Name: "public",
			// 		},
			// 	},
			// 	RoleName: "readonly",
			// 	Scopes:   []string{"^$", "^library\/.+$"},
			// },
			{
				Name: "admins",
				Subjects: []rbac.Subject{
					{
						Kind: "Group",
						Name: "admins",
					},
				},
				RoleName: "write",
				Scopes:   []string{"^.*$"},
			},
		},
	}
}

func LoadRBACFromYamlDir(dirName string) rbac.Engine {
	entries, err := os.ReadDir(dirName)
	if err != nil {
		panic(err)
	}

	tokens := []rbac.Token{}
	users := []rbac.User{}
	roles := []rbac.Role{}
	roleBindings := []rbac.RoleBinding{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if slices.Contains([]string{".yml", ".yaml"}, ext) {
			d, err := os.ReadFile(filepath.Join(dirName, name))
			if err != nil {
				panic(err)
			}

			if t, u, r, rb, err := yaml.ParseYAML(d); err == nil {
				tokens = append(tokens, t...)
				users = append(users, u...)
				roles = append(roles, r...)
				roleBindings = append(roleBindings, rb...)
			}
		}
	}

	return rbac.Engine{
		Tokens:       tokens,
		Users:        users,
		Roles:        roles,
		RoleBindings: roleBindings,
	}
}
