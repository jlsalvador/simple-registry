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
	"os"
	"regexp"
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

type ConfigurationManifest struct {
	yamlscheme.CommonManifest

	Metadata struct {
		Name string `json:"name" yaml:"name"`
	} `json:"metadata" yaml:"metadata"`
	Spec struct {
		DataDir string `json:"dataDir" yaml:"dataDir"`

		Http struct {
			Addr         string `json:"addr" yaml:"addr"`
			TokenSecret  string `json:"tokenSecret" yaml:"tokenSecret"`
			TokenTimeout int    `json:"tokenTimeout" yaml:"tokenTimeout"`
			UI           bool   `json:"ui" yaml:"ui"`
			CertFile     string `json:"certfile" yaml:"certfile"`
			KeyFile      string `json:"keyfile" yaml:"keyfile"`
		} `json:"http" yaml:"http"`
	} `json:"spec" yaml:"spec"`
}

func init() {
	yamlscheme.Register[TokenManifest](ApiVersion, "Token")
	yamlscheme.Register[UserManifest](ApiVersion, "User")
	yamlscheme.Register[RoleManifest](ApiVersion, "Role")
	yamlscheme.Register[RoleBindingManifest](ApiVersion, "RoleBinding")
	yamlscheme.Register[PullThroughCacheManifest](ApiVersion, "PullThroughCache")
	yamlscheme.Register[ConfigurationManifest](ApiVersion, "Configuration")
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

			rb := rbac.RoleBinding{
				Name:     m.Metadata.Name,
				RoleName: m.Spec.RoleRef.Name,
				Subjects: subjects,
			}

			for _, s := range m.Spec.Scopes {
				var re *regexp.Regexp
				re, err = regexp.Compile(s)
				if err != nil {
					return
				}
				rb.Scopes = append(rb.Scopes, *re)
			}

			roleBindings = append(roleBindings, rb)
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

func GetDataDirFromManifests(manifests []any) (dataDir string) {
	for _, manifest := range manifests {
		if m, ok := manifest.(*ConfigurationManifest); ok {
			if m.Spec.DataDir != "" {
				dataDir = m.Spec.DataDir
			}
		}
	}

	return
}

func GetHttpFromManifests(manifests []any) (http Http) {
	for _, manifest := range manifests {
		if m, ok := manifest.(*ConfigurationManifest); ok {
			if m.Spec.Http.Addr != "" {
				http.Addr = m.Spec.Http.Addr
			}
			if m.Spec.Http.TokenSecret != "" {
				http.TokenSecret = []byte(m.Spec.Http.TokenSecret)
			}
			if m.Spec.Http.TokenTimeout != 0 {
				http.TokenTimeout = time.Duration(m.Spec.Http.TokenTimeout) * time.Second
			}
			if m.Spec.Http.UI {
				http.UI = m.Spec.Http.UI
			}
			if m.Spec.Http.CertFile != "" {
				http.CertFile = m.Spec.Http.CertFile
			}
			if m.Spec.Http.KeyFile != "" {
				http.KeyFile = m.Spec.Http.KeyFile
			}
		}
	}

	return
}
