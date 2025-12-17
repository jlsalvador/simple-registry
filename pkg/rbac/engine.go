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
	"regexp"
	"slices"
)

type Engine struct {
	Tokens       []Token
	Users        []User
	Roles        []Role
	RoleBindings []RoleBinding
}

func (e *Engine) IsAllowed(username string, resource string, scope string, verb string) bool {
	// Get user from "username".
	var user *User
	if i := slices.IndexFunc(e.Users, func(u User) bool {
		return u.Name == username
	}); i >= 0 {
		user = &e.Users[i]
	} else {
		return false
	}

	for _, rb := range e.RoleBindings {
		// Match role.
		if i := slices.IndexFunc(e.Roles, func(r Role) bool {
			return r.Name == rb.RoleName && slices.Contains(r.Verbs, verb) && (slices.Contains(r.Resources, resource) || slices.Contains(r.Resources, "*"))
		}); i < 0 {
			continue
		}

		// Match subjects and "username".
		if i := slices.IndexFunc(rb.Subjects, func(s Subject) bool {
			return s.Kind == "User" && s.Name == user.Name || s.Kind == "Group" && slices.Contains(user.Groups, s.Name)
		}); i < 0 {
			continue
		}

		// Match scopes.
		if i := slices.IndexFunc(rb.Scopes, func(s string) bool {
			re, err := regexp.Compile(s)
			return err == nil && re.MatchString(scope)
		}); i < 0 {
			continue
		}

		return true
	}

	return false
}
