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
	"encoding/base64"
	"errors"
	netHttp "net/http"
	"regexp"
	"slices"
	"strings"
	"time"

	httpErrors "github.com/jlsalvador/simple-registry/pkg/http/errors"
)

const AnonymousUsername = "anonymous"

var httpAuthBasicRegexp = regexp.MustCompile(`^Basic\s+([a-zA-Z0-9+/]+={0,2})$`)
var httpAuthBearerRegexp = regexp.MustCompile(`^Bearer\s+([a-zA-Z0-9+/]+={0,2})$`)

func (e *Engine) getUsernameFromAuthBasic(matches []string, isAnonymousUserEnabled bool) (string, error) {
	encoded := matches[1]

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.Join(httpErrors.ErrBadRequest, err)
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", httpErrors.ErrBadRequest
	}

	username := parts[0]
	password := parts[1]

	// Empty username & password could be a valid anonymous user.
	if username == "" && password == "" && isAnonymousUserEnabled {
		return AnonymousUsername, nil
	}

	// Check if the user exists and password is valid.
	if i := slices.IndexFunc(e.Users, func(user User) bool {
		return user.Name == username && user.IsPasswordValid(password)
	}); i >= 0 {
		return e.Users[i].Name, nil
	}

	return "", httpErrors.ErrUnauthorized
}

func (e *Engine) getUsernameFromAuthBearer(matches []string) (string, error) {
	token := matches[1]

	for _, t := range e.Tokens {
		if t.Value != token || time.Now().After(t.ExpiresAt) {
			continue
		}

		for _, u := range e.Users {
			if u.Name == t.Username {
				return u.Name, nil
			}
		}
	}

	return "", httpErrors.ErrUnauthorized
}

// GetUsernameFromHttpRequest extracts the username from an HTTP request.
//
// Returns [AnonymousUsername] if the request does not contain any
// authorization header and there is an anonymous user in RBAC users.
func (e *Engine) GetUsernameFromHttpRequest(r *netHttp.Request) (string, error) {
	if r == nil {
		return "", httpErrors.ErrBadRequest
	}

	isAnonymousUserEnabled := slices.IndexFunc(e.Users, func(u User) bool {
		return u.Name == AnonymousUsername
	}) >= 0

	v := r.Header.Get("Authorization")

	// Basic auth.
	if matches := httpAuthBasicRegexp.FindStringSubmatch(v); len(matches) == 2 {
		return e.getUsernameFromAuthBasic(matches, isAnonymousUserEnabled)
	}

	// Bearer token auth.
	if matches := httpAuthBearerRegexp.FindStringSubmatch(v); len(matches) == 2 {
		return e.getUsernameFromAuthBearer(matches)
	}

	// No authentication header found, return anonymous user if there is one.
	if isAnonymousUserEnabled {
		return AnonymousUsername, nil
	}

	return "", httpErrors.ErrUnauthorized
}
