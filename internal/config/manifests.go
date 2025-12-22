package config

import (
	"os"
	"time"

	"github.com/jlsalvador/simple-registry/internal/data/proxy"
	"github.com/jlsalvador/simple-registry/pkg/rbac"
	"github.com/jlsalvador/simple-registry/pkg/yamlscheme"
)

const ApiVersion = "simple-registry.jlsalvador.online/v1beta1"

type TokenManifest struct {
	yamlscheme.CommonManifest

	Metadata struct {
		Name string `json:"name" yaml:"name"`
	} `json:"metadata" yaml:"metadata"`
	Spec struct {
		Value     string    `json:"value" yaml:"value"`
		ExpiresAt time.Time `json:"expiresAt" yaml:"expiresAt"` // RFC3339 timestamp.
		Username  string    `json:"username" yaml:"username"`
	} `json:"spec" yaml:"spec"`
}

type UserManifest struct {
	yamlscheme.CommonManifest

	Metadata struct {
		Name string `json:"name" yaml:"name"`
	} `json:"metadata" yaml:"metadata"`
	Spec struct {
		PasswordHash string   `json:"passwordHash,omitempty" yaml:"passwordHash,omitempty"` // bcrypt hashed password.
		Groups       []string `json:"groups" yaml:"groups"`
	} `json:"spec" yaml:"spec"`
}

type RoleManifest struct {
	yamlscheme.CommonManifest

	Metadata struct {
		Name string `json:"name" yaml:"name"`
	} `json:"metadata" yaml:"metadata"`
	Spec struct {
		Resources []string `json:"resources" yaml:"resources"` // "catalog", "blobs", "manifests", "tags", or "*".
		Verbs     []string `json:"verbs" yaml:"verbs"`         // "HEAD", "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "TRACE", or "*".
	} `json:"spec" yaml:"spec"`
}

type RoleBindingManifest struct {
	yamlscheme.CommonManifest

	Metadata struct {
		Name string `json:"name" yaml:"name"`
	} `json:"metadata" yaml:"metadata"`
	Spec struct {
		Subjects []struct {
			Kind string `json:"kind" yaml:"kind"` // "User" or "Group".
			Name string `json:"name" yaml:"name"`
		} `json:"subjects" yaml:"subjects"`
		RoleRef struct {
			Name string `json:"name" yaml:"name"`
		} `json:"roleRef" yaml:"roleRef"`
		Scopes []string `json:"scopes" yaml:"scopes"` // Regular expressions matching the repository path."
	} `json:"spec" yaml:"spec"`
}

type PullThroughCacheManifest struct {
	yamlscheme.CommonManifest

	Metadata struct {
		Name string `json:"name" yaml:"name"`
	} `json:"metadata" yaml:"metadata"`
	Spec struct {
		Upstream struct {
			URL          string        `json:"url" yaml:"url"`
			Timeout      time.Duration `json:"timeout" yaml:"timeout"`
			Username     string        `json:"username" yaml:"username"`
			Password     string        `json:"password" yaml:"password"`
			PasswordFile string        `json:"passwordFile" yaml:"passwordFile"`
			TTL          string        `json:"ttl" yaml:"ttl"`
		}
		Scopes []string `json:"scopes" yaml:"scopes"` // Regular expressions matching the repository path."
	} `json:"spec" yaml:"spec"`
}

func init() {
	yamlscheme.Register[TokenManifest](ApiVersion, "Token")
	yamlscheme.Register[UserManifest](ApiVersion, "User")
	yamlscheme.Register[RoleManifest](ApiVersion, "Role")
	yamlscheme.Register[RoleBindingManifest](ApiVersion, "RoleBinding")
	yamlscheme.Register[PullThroughCacheManifest](ApiVersion, "PullThroughCache")
}

func GetTokensUsersRolesRoleBindingsFromManifests(manifests []any) (
	tokens []rbac.Token,
	users []rbac.User,
	roles []rbac.Role,
	roleBindings []rbac.RoleBinding,
	err error,
) {
	for _, manifest := range manifests {
		switch m := manifest.(type) {

		case *TokenManifest:
			tokens = append(tokens, rbac.Token{
				Name:      m.Metadata.Name,
				Value:     m.Spec.Value,
				Username:  m.Spec.Username,
				ExpiresAt: m.Spec.ExpiresAt,
			})

		case *UserManifest:
			users = append(users, rbac.User{
				Name:         m.Metadata.Name,
				PasswordHash: m.Spec.PasswordHash,
				Groups:       m.Spec.Groups,
			})

		case *RoleManifest:
			var verbs []string
			verbs, err = rbac.ParseVerbs(m.Spec.Verbs)
			if err != nil {
				return
			}
			roles = append(roles, rbac.Role{
				Name:      m.Metadata.Name,
				Resources: m.Spec.Resources,
				Verbs:     verbs,
			})

		case *RoleBindingManifest:
			subjects := []rbac.Subject{}
			for _, s := range m.Spec.Subjects {
				subjects = append(subjects, rbac.Subject{
					Kind: s.Kind,
					Name: s.Name,
				})
			}

			roleBindings = append(roleBindings, rbac.RoleBinding{
				Name:     m.Metadata.Name,
				RoleName: m.Spec.RoleRef.Name,
				Subjects: subjects,
				Scopes:   m.Spec.Scopes,
			})
		}
	}

	return
}

func GetProxiesFromManifests(manifests []any) (proxies []proxy.Proxy, err error) {
	for _, manifest := range manifests {
		if m, ok := manifest.(*PullThroughCacheManifest); ok {
			if m.Spec.Upstream.PasswordFile != "" {
				password, err := os.ReadFile(m.Spec.Upstream.PasswordFile)
				if err != nil {
					return nil, err
				}
				m.Spec.Upstream.Password = string(password)
			}

			proxies = append(proxies, proxy.Proxy{
				Url:      m.Spec.Upstream.URL,
				Timeout:  m.Spec.Upstream.Timeout,
				Username: m.Spec.Upstream.Username,
				Password: m.Spec.Upstream.Password,
				Scopes:   m.Spec.Scopes,
			})
		}
	}

	return
}
