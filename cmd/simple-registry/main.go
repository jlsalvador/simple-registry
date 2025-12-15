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
	"net/http"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/internal/data/filesystem"
	"github.com/jlsalvador/simple-registry/internal/http/handler"
	"github.com/jlsalvador/simple-registry/internal/rbac"

	"golang.org/x/crypto/bcrypt"
)

func main() {

	rbacEngine := rbac.Engine{
		Users: []rbac.User{
			{
				// Administrator.
				Name: "admin",
				PasswordHash: func() string {
					pwd, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
					return string(pwd)
				}(),
			},
			{
				// Anonymous.
				Name:   "",
				Groups: []string{"public"},
			},
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
			{
				Name: "public",
				Subjects: []rbac.Subject{
					{
						Kind: "Group",
						Name: "public",
					},
				},
				RoleName: "write",
				Scopes:   []string{"^.*$"},
			},
			{
				Name: "admin",
				Subjects: []rbac.Subject{
					{
						Kind: "User",
						Name: "admin",
					},
				},
				RoleName: "write",
				Scopes:   []string{"^.*$"},
			},
		},
	}

	config := config.Config{
		Rbac: rbacEngine,
		Data: filesystem.NewFilesystemDataStorage("/tmp/registry"),
	}
	addr := ":5000"

	h := handler.NewHandler(config)
	if err := http.ListenAndServe(addr, h); err != nil {
		panic(err)
	}
}
