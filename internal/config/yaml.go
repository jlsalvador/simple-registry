package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jlsalvador/simple-registry/internal/data/filesystem"
	"github.com/jlsalvador/simple-registry/pkg/log"
	"github.com/jlsalvador/simple-registry/pkg/rbac"
	"github.com/jlsalvador/simple-registry/pkg/yamlscheme"
)

func parseYamlDir(dirName string) (
	tokens []rbac.Token,
	users []rbac.User,
	roles []rbac.Role,
	roleBindings []rbac.RoleBinding,
	err error,
) {
	entries, err := os.ReadDir(dirName)
	if err != nil {
		return tokens, users, roles, roleBindings, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		filename := filepath.Join(dirName, name)
		f, err := os.Open(filename)
		if err != nil {
			log.Error(
				"filename", filename,
				"err", err,
			).Print()
		}
		defer f.Close()

		m, err := yamlscheme.DecodeAll(f)
		if err != nil {
			return nil, nil, nil, nil, err
		}

		t, u, r, rb := GetTokensUsersRolesRoleBindingsFromManifests(m)
		tokens = append(tokens, t...)
		users = append(users, u...)
		roles = append(roles, r...)
		roleBindings = append(roleBindings, rb...)
	}

	return tokens, users, roles, roleBindings, nil
}

func NewFromYamlDir(
	dirsName []string,
	dataDir string,
) (*Config, error) {
	tokens := []rbac.Token{}
	users := []rbac.User{}
	roles := []rbac.Role{}
	roleBindings := []rbac.RoleBinding{}

	for _, dirName := range dirsName {
		ts, us, rs, rbs, err := parseYamlDir(dirName)
		if err != nil {
			return nil, err
		}

		tokens = append(tokens, ts...)
		users = append(users, us...)
		roles = append(roles, rs...)
		roleBindings = append(roleBindings, rbs...)
	}

	rbacEngine := rbac.Engine{
		Tokens:       tokens,
		Users:        users,
		Roles:        roles,
		RoleBindings: roleBindings,
	}

	return &Config{
		WWWAuthenticate: `Basic realm="simple-registry"`,
		Rbac:            rbacEngine,
		Data:            filesystem.NewFilesystemDataStorage(dataDir),
	}, nil
}
