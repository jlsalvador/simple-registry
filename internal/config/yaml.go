package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jlsalvador/simple-registry/internal/data/filesystem"
	"github.com/jlsalvador/simple-registry/internal/data/proxy"
	"github.com/jlsalvador/simple-registry/pkg/log"
	"github.com/jlsalvador/simple-registry/pkg/rbac"
	"github.com/jlsalvador/simple-registry/pkg/yamlscheme"
)

func parseYamlDir(dirName string) (manifests []any, err error) {
	entries, err := os.ReadDir(dirName)
	if err != nil {
		return nil, nil
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
			return nil, err
		}

		manifests = append(manifests, m...)
	}

	return manifests, nil
}

func NewFromYamlDir(
	dirsName []string,
	dataDir string,
) (*Config, error) {
	manifests := []any{}
	for _, dirName := range dirsName {
		ms, err := parseYamlDir(dirName)
		if err != nil {
			return nil, err
		}
		manifests = append(manifests, ms...)
	}

	tokens, users, roles, roleBindings, err := GetTokensUsersRolesRoleBindingsFromManifests(manifests)
	if err != nil {
		return nil, err
	}

	rbacEngine := rbac.Engine{
		Tokens:       tokens,
		Users:        users,
		Roles:        roles,
		RoleBindings: roleBindings,
	}

	proxies, err := GetProxiesFromManifests(manifests)
	if err != nil {
		return nil, err
	}

	fs := filesystem.NewFilesystemDataStorage(dataDir)
	ds := proxy.NewProxyDataStorage(fs, proxies)

	return &Config{
		WWWAuthenticate: `Basic realm="simple-registry"`,
		Rbac:            rbacEngine,
		Data:            ds,
	}, nil
}
