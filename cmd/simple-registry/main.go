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

package main

import (
	"flag"
	"net/http"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/internal/data/filesystem"
	"github.com/jlsalvador/simple-registry/internal/http/handler"
	"github.com/jlsalvador/simple-registry/internal/rbac"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	addr := flag.String("addr", "0.0.0.0:5000", "Listening address")
	datadir := flag.String("datadir", "./data", "Data directory")
	adminName := flag.String("adminname", "admin", "Administrator name")
	adminPwd := flag.String("adminpwd", "admin", "Administrator password")
	cert := flag.String("cert", "", "TLS certificate")
	key := flag.String("key", "", "TLS key")
	flag.Parse()

	rbacEngine := rbac.Engine{
		Users: []rbac.User{
			{
				// Administrator.
				Name: *adminName,
				PasswordHash: func() string {
					pwd, _ := bcrypt.GenerateFromPassword([]byte(*adminPwd), bcrypt.DefaultCost)
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
				Verbs: []rbac.Action{
					rbac.ActionPull,
					rbac.ActionPush,
					rbac.ActionDelete,
				},
			},
			// Read-Only
			{
				Name:      "readonly",
				Resources: []string{"*"},
				Verbs: []rbac.Action{
					rbac.ActionPull,
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

	config := config.Config{
		Rbac: rbacEngine,
		Data: filesystem.NewFilesystemDataStorage(*datadir),
	}

	h := handler.NewHandler(config)

	if *cert != "" && *key != "" {
		if err := http.ListenAndServeTLS(*addr, *cert, *key, h); err != nil {
			panic(err)
		}
	} else {
		if err := http.ListenAndServe(*addr, h); err != nil {
			panic(err)
		}
	}
}
