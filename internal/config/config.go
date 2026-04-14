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
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/jlsalvador/simple-registry/internal/data"
	"github.com/jlsalvador/simple-registry/internal/data/filesystem"
	"github.com/jlsalvador/simple-registry/internal/data/proxy"
	"github.com/jlsalvador/simple-registry/internal/version"
	"github.com/jlsalvador/simple-registry/pkg/log"
	"github.com/jlsalvador/simple-registry/pkg/rbac"

	"golang.org/x/crypto/bcrypt"
)

type Http struct {
	Addr         string
	TokenSecret  []byte
	TokenTimeout time.Duration
	UI           bool
	CertFile     string
	KeyFile      string
}

type Config struct {
	Http Http
	Rbac rbac.Engine
	Data data.DataStorage
}

type options struct {
	adminName    string
	adminPwd     []byte
	tokenSecret  []byte
	tokenTimeout time.Duration
	addr         string
	ui           bool
	certfile     string
	keyfile      string

	rbacEngine *rbac.Engine
	data       data.DataStorage
}

type Option func(*options)

func WithAdminName(name string) Option {
	return func(o *options) {
		o.adminName = name
	}
}

func WithAdminPwd(pwd []byte) Option {
	return func(o *options) {
		o.adminPwd = pwd
	}
}

func WithAdminPwdFile(file string) Option {
	return func(o *options) {
		b, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}
		o.adminPwd = b
	}
}

func WithHttpTokenSecret(secret []byte) Option {
	return func(o *options) {
		o.tokenSecret = secret
	}
}

func WithHttpTokenSecretFile(file string) Option {
	return func(o *options) {
		b, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}
		o.tokenSecret = b
	}
}

func WithHttpTokenTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.tokenTimeout = timeout
	}
}

func WithDataDir(dir string) Option {
	return func(o *options) {
		o.data = filesystem.NewFilesystemDataStorage(dir)
	}
}

func WithHttpAddr(addr string) Option {
	return func(o *options) {
		o.addr = addr
	}
}

func WithHttpUI(enable bool) Option {
	return func(o *options) {
		o.ui = true
	}
}

func WithHttpCertFile(certFile string) Option {
	return func(o *options) {
		o.certfile = certFile
	}
}

func WithHttpKeyFile(keyFile string) Option {
	return func(o *options) {
		o.keyfile = keyFile
	}
}

func WithCfgDirs(dirs []string) Option {
	return func(o *options) {
		manifests := []any{}
		for _, dir := range dirs {
			ms, err := parseYamlDir(dir)
			if err != nil {
				panic(err)
			}
			manifests = append(manifests, ms...)
		}

		tokens, users, roles, roleBindings, err := GetTokensUsersRolesRoleBindingsFromManifests(manifests)
		if err != nil {
			panic(err)
		}

		o.rbacEngine = &rbac.Engine{
			Tokens:       tokens,
			Users:        users,
			Roles:        roles,
			RoleBindings: roleBindings,
		}

		proxies, err := GetProxiesFromManifests(manifests)
		if err != nil {
			panic(err)
		}

		dataDir := GetDataDirFromManifests(manifests)
		if dataDir == "" {
			panic("datadir is empty, please use flag -datadir or use YAML Configuration.spec.dataDir")
		}

		fs := filesystem.NewFilesystemDataStorage(dataDir)
		o.data = proxy.NewProxyDataStorage(fs, proxies)

		http := GetHttpFromManifests(manifests)
		if http.Addr != "" {
			WithHttpAddr(http.Addr)(o)
		}
		if len(http.TokenSecret) > 0 {
			WithHttpTokenSecret(http.TokenSecret)(o)
		}
		if http.TokenTimeout > 0 {
			WithHttpTokenTimeout(http.TokenTimeout)(o)
		}
		if http.UI {
			WithHttpUI(http.UI)(o)
		}
		if http.CertFile != "" {
			WithHttpCertFile(http.CertFile)(o)
		}
		if http.KeyFile != "" {
			WithHttpKeyFile(http.KeyFile)(o)
		}
	}
}

func New(opts ...Option) (*Config, error) {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	// Data
	if o.data == nil {
		panic("datadir is empty, please use flag -datadir")
	}

	// RBAC
	if o.rbacEngine == nil {
		if string(o.adminPwd) == "" {
			panic("adminpwd is empty, please use flag -adminpwd")
		}
		adminPwdHash, err := bcrypt.GenerateFromPassword(
			o.adminPwd,
			bcrypt.DefaultCost,
		)
		if err != nil {
			return nil, err
		}

		o.rbacEngine = &rbac.Engine{
			Users: []rbac.User{
				{
					// Administrator.
					Name:         o.adminName,
					PasswordHash: string(adminPwdHash),
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
					Scopes:   []regexp.Regexp{*regexp.MustCompile("^.*$")},
				},
			},
		}
	}

	// Http
	if o.addr == "" {
		o.addr = "0.0.0.0:5000"
	}
	if len(o.tokenSecret) == 0 {
		o.tokenSecret = []byte(rand.Text())

		log.Info(
			"service.name", version.AppName,
			"service.version", version.AppVersion,
			"event.dataset", "internal.config",
			"message", fmt.Sprintf("generated token secret: %s", o.tokenSecret),
		).Print()
	}
	if o.tokenTimeout == 0 {
		o.tokenTimeout = time.Second * 30
	}
	http := Http{
		Addr:         "0.0.0.0:5000",
		TokenSecret:  o.tokenSecret,
		TokenTimeout: o.tokenTimeout,
		UI:           o.ui,
		CertFile:     o.certfile,
		KeyFile:      o.keyfile,
	}

	return &Config{
		Http: http,
		Rbac: *o.rbacEngine,
		Data: o.data,
	}, nil
}
