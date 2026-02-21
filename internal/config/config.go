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
	"net/http"
	"os"
	"regexp"

	"github.com/jlsalvador/simple-registry/internal/data"
	"github.com/jlsalvador/simple-registry/internal/data/filesystem"
	"github.com/jlsalvador/simple-registry/pkg/rbac"

	"golang.org/x/crypto/bcrypt"
)

type Config struct {
	IsUIEnabled     bool
	WWWAuthenticate string
	Rbac            rbac.Engine
	Data            data.DataStorage
}

func New(adminName, adminPwd, adminPwdFile, dataDir string) (*Config, error) {
	var b []byte
	var err error
	if adminPwdFile != "" {
		b, err = os.ReadFile(adminPwdFile)
		if err != nil {
			return nil, err
		}
	} else if adminPwd != "" {
		b, err = bcrypt.GenerateFromPassword([]byte(adminPwd), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
	} else {
		panic("adminpwd is empty, please use flag -adminpwd")
	}
	pwd := string(b)

	rbacEngine := rbac.Engine{
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
				Scopes:   []regexp.Regexp{*regexp.MustCompile("^.*$")},
			},
		},
	}

	return &Config{
		WWWAuthenticate: `Basic realm="simple-registry"`,
		Rbac:            rbacEngine,
		Data:            filesystem.NewFilesystemDataStorage(dataDir),
	}, nil
}
