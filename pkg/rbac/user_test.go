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

package rbac_test

import (
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/rbac"

	"golang.org/x/crypto/bcrypt"
)

var TestUser = rbac.User{
	Name: "testuser",
	PasswordHash: func() string {
		pwd, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
		return string(pwd)
	}(),
}

func TestIsPasswordValid(t *testing.T) {
	tcs := []struct {
		name          string
		plainPassword string
		want          bool
	}{
		{
			name:          "valid password",
			plainPassword: "password123",
			want:          true,
		},
		{
			name:          "invalid password",
			plainPassword: "123456",
			want:          false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := TestUser.IsPasswordValid(tc.plainPassword)
			if got != tc.want {
				t.Errorf("IsPasswordValid() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHasUser(t *testing.T) {
	// Create an engine with the test user.
	engine := &rbac.Engine{
		Users: []rbac.User{TestUser},
	}

	tcs := []struct {
		name     string
		username string
		password string
		want     bool
	}{
		{
			name:     "valid user and password",
			username: "testuser",
			password: "password123",
			want:     true,
		},
		{
			name:     "invalid password",
			username: "testuser",
			password: "wrongpassword",
			want:     false,
		},
		{
			name:     "nonexistent user",
			username: "nonexistent",
			password: "password123",
			want:     false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := engine.HasUser(tc.username, tc.password)
			if got != tc.want {
				t.Errorf("HasUser() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsAnonymousUserEnabled(t *testing.T) {
	tcs := []struct {
		name  string
		users []rbac.User
		want  bool
	}{
		{
			name: "anonymous user enabled",
			users: []rbac.User{
				TestUser,
				{Name: rbac.AnonymousUsername},
			},
			want: true,
		},
		{
			name:  "anonymous user disabled",
			users: []rbac.User{TestUser},
			want:  false,
		},
		{
			name:  "no users",
			users: []rbac.User{},
			want:  false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			engine := &rbac.Engine{
				Users: tc.users,
			}

			got := engine.IsAnonymousUserEnabled()
			if got != tc.want {
				t.Errorf("IsAnonymousUserEnabled() = %v, want %v", got, tc.want)
			}
		})
	}
}
