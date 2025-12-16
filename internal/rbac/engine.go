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

func (e *Engine) IsAllowed(username string, resource string, repo string, verb string) bool {
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

		// Match scopes and "repo".
		if i := slices.IndexFunc(rb.Scopes, func(s string) bool {
			re, err := regexp.Compile(s)
			return err == nil && re.MatchString(repo)
		}); i < 0 {
			continue
		}

		return true
	}

	return false
}
