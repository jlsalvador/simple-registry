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

package rbac

import (
	"slices"

	"golang.org/x/crypto/bcrypt"
)

const AnonymousUsername = "anonymous"

func (e *Engine) HasUser(usr string, pwd string) bool {
	if i := slices.IndexFunc(e.Users, func(user User) bool {
		return user.Name == usr && user.IsPasswordValid(pwd)
	}); i >= 0 {
		return true
	}
	return false
}

// IsAnonymousUserEnabled check if anonymous user is enabled.
func (e *Engine) IsAnonymousUserEnabled() bool {
	return slices.IndexFunc(e.Users, func(u User) bool {
		return u.Name == AnonymousUsername
	}) >= 0
}

type User struct {
	Name         string
	PasswordHash string
	Groups       []string
}

func (u *User) IsPasswordValid(pwd string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(pwd)) == nil
}
