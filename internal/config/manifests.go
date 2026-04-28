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

const apiVersion = "simple-registry.jlsalvador.online/v1beta1"

type tokenManifest struct {
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

type userManifest struct {
	yamlscheme.CommonManifest

	Metadata struct {
		Name string `json:"name" yaml:"name"`
	} `json:"metadata" yaml:"metadata"`
	Spec struct {
		PasswordHash string   `json:"passwordHash,omitempty" yaml:"passwordHash,omitempty"` // bcrypt hashed password.
		Groups       []string `json:"groups" yaml:"groups"`
	} `json:"spec" yaml:"spec"`
}

type roleManifest struct {
	yamlscheme.CommonManifest

	Metadata struct {
		Name string `json:"name" yaml:"name"`
	} `json:"metadata" yaml:"metadata"`
	Spec struct {
		Resources []string `json:"resources" yaml:"resources"` // "catalog", "blobs", "manifests", "tags", or "*".
		Verbs     []string `json:"verbs" yaml:"verbs"`         // "HEAD", "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "TRACE", or "*".
	} `json:"spec" yaml:"spec"`
}

type roleBindingManifest struct {
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

type pullThroughCacheManifest struct {
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

type configurationManifest struct {
	yamlscheme.CommonManifest

	Metadata struct {
		Name string `json:"name" yaml:"name"`
	} `json:"metadata" yaml:"metadata"`
	Spec struct {
		DataDir string `json:"dataDir" yaml:"dataDir"`

		Web struct {
			Addr         string `json:"addr" yaml:"addr"`
			TokenSecret  string `json:"tokenSecret" yaml:"tokenSecret"`
			TokenTimeout int    `json:"tokenTimeout" yaml:"tokenTimeout"`
			UI           bool   `json:"ui" yaml:"ui"`
			CertFile     string `json:"certfile" yaml:"certfile"`
			KeyFile      string `json:"keyfile" yaml:"keyfile"`
		} `json:"web" yaml:"web"`
	} `json:"spec" yaml:"spec"`
}

func init() {
	yamlscheme.Register[tokenManifest](apiVersion, "Token")
	yamlscheme.Register[userManifest](apiVersion, "User")
	yamlscheme.Register[roleManifest](apiVersion, "Role")
	yamlscheme.Register[roleBindingManifest](apiVersion, "RoleBinding")
	yamlscheme.Register[pullThroughCacheManifest](apiVersion, "PullThroughCache")
	yamlscheme.Register[configurationManifest](apiVersion, "Configuration")
}

func getTokensUsersRolesRoleBindingsFromManifests(manifests []any) (
	tokens []rbac.Token,
	users []rbac.User,
	roles []rbac.Role,
	roleBindings []rbac.RoleBinding,
	err error,
) {
	for _, manifest := range manifests {
		switch m := manifest.(type) {

		case *tokenManifest:
			tokens = append(tokens, rbac.Token{
				Name:      m.Metadata.Name,
				Value:     m.Spec.Value,
				Username:  m.Spec.Username,
				ExpiresAt: m.Spec.ExpiresAt,
			})

		case *userManifest:
			users = append(users, rbac.User{
				Name:         m.Metadata.Name,
				PasswordHash: m.Spec.PasswordHash,
				Groups:       m.Spec.Groups,
			})

		case *roleManifest:
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

		case *roleBindingManifest:
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

func getProxiesFromManifests(manifests []any) (proxies []proxy.Proxy, err error) {
	for _, manifest := range manifests {
		if m, ok := manifest.(*pullThroughCacheManifest); ok {
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

func getDataDirFromManifests(manifests []any) (dataDir string) {
	for _, manifest := range manifests {
		if m, ok := manifest.(*configurationManifest); ok {
			if m.Spec.DataDir != "" {
				dataDir = m.Spec.DataDir
			}
		}
	}

	return
}

func getWebFromManifests(manifests []any) (web Web) {
	for _, manifest := range manifests {
		if m, ok := manifest.(*configurationManifest); ok {
			if m.Spec.Web.Addr != "" {
				web.Addr = m.Spec.Web.Addr
			}
			if m.Spec.Web.TokenSecret != "" {
				web.TokenSecret = []byte(m.Spec.Web.TokenSecret)
			}
			if m.Spec.Web.TokenTimeout != 0 {
				web.TokenTimeout = time.Duration(m.Spec.Web.TokenTimeout) * time.Second
			}
			if m.Spec.Web.UI {
				web.UI = m.Spec.Web.UI
			}
			if m.Spec.Web.CertFile != "" {
				web.CertFile = m.Spec.Web.CertFile
			}
			if m.Spec.Web.KeyFile != "" {
				web.KeyFile = m.Spec.Web.KeyFile
			}
		}
	}

	return
}
