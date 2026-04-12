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
	"github.com/jlsalvador/simple-registry/internal/version"
	"github.com/jlsalvador/simple-registry/pkg/log"
	"github.com/jlsalvador/simple-registry/pkg/rbac"

	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	Rbac rbac.Engine
	Data data.DataStorage
}

type options struct {
	adminName    string
	adminPwd     []byte
	tokenSecret  []byte
	tokenTimeout time.Duration

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

func WithTokenSecret(secret []byte) Option {
	return func(o *options) {
		o.tokenSecret = secret
	}
}

func WithTokenSecretFile(file string) Option {
	return func(o *options) {
		b, err := os.ReadFile(file)
		if err != nil {
			panic(err)
		}
		o.tokenSecret = b
	}
}

func WithTokenTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.tokenTimeout = timeout
	}
}

func WithDataDir(dir string) Option {
	return func(o *options) {
		o.data = filesystem.NewFilesystemDataStorage(dir)
	}
}

func WithCfgDirs(dir []string) Option {
	//FIXME
	return func(o *options) {
		panic("unimplemented")
	}
}

func New(opts ...Option) (*Config, error) {
	o := options{}
	for _, opt := range opts {
		opt(&o)
	}

	if string(o.adminPwd) == "" {
		panic("adminpwd is empty, please use flag -adminpwd")
	}
	if o.data == nil {
		panic("datadir is empty, please use flag -datadir")
	}
	if string(o.tokenSecret) == "" {
		o.tokenSecret = []byte(rand.Text())

		log.Info(
			"service.name", version.AppName,
			"service.version", version.AppVersion,
			"event.dataset", "cmd.serve",
			"message", fmt.Sprintf("generated token secret: %s", o.tokenSecret),
		).Print()
	}
	if o.tokenTimeout == 0 {
		o.tokenTimeout = time.Second * 30
	}

	if o.rbacEngine == nil {
		adminPwdHash, err := bcrypt.GenerateFromPassword(
			o.adminPwd,
			bcrypt.DefaultCost,
		)
		if err != nil {
			return nil, err
		}

		o.rbacEngine = &rbac.Engine{
			TokenSecret:  o.tokenSecret,
			TokenTimeout: o.tokenTimeout,

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

	return &Config{
		Rbac: *o.rbacEngine,
		Data: o.data,
	}, nil
}
