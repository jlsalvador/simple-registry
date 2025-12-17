package config

import (
	"net/http"
	"os"
	"path/filepath"
	"slices"

	"github.com/jlsalvador/simple-registry/pkg/rbac"
	"github.com/jlsalvador/simple-registry/pkg/rbac/yaml"

	"golang.org/x/crypto/bcrypt"
)

func GetRBACEngineStatic(
	adminName,
	adminPwd,
	adminPwdFile string,
) (*rbac.Engine, error) {
	var b []byte
	var err error
	if adminPwdFile != "" {
		b, err = os.ReadFile(adminPwdFile)
		if err != nil {
			return nil, err
		}
	} else {
		b, err = bcrypt.GenerateFromPassword([]byte(adminPwd), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
	}
	pwd := string(b)

	return &rbac.Engine{
		Users: []rbac.User{
			{
				// Administrator.
				Name:         adminName,
				PasswordHash: pwd,
				Groups:       []string{"admins", "public"},
			},
			// {
			// 	// Anonymous.
			// 	Name:   "anonymous",
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
	}, nil
}

func LoadRBACFromYamlDir(dirName string) (*rbac.Engine, error) {
	entries, err := os.ReadDir(dirName)
	if err != nil {
		return nil, err
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

	return &rbac.Engine{
		Tokens:       tokens,
		Users:        users,
		Roles:        roles,
		RoleBindings: roleBindings,
	}, nil
}
