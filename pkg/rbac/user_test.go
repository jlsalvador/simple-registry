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

func TestIsPasswordValid(t *testing.T) {
	var user = rbac.User{
		Name: "testuser",
		PasswordHash: func() string {
			pwd, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
			return string(pwd)
		}(),
	}

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

			got := user.IsPasswordValid(tc.plainPassword)
			if got != tc.want {
				t.Errorf("IsPasswordValid() = %v, want %v", got, tc.want)
			}
		})
	}
}
