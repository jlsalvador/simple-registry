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
)

const AnonymousUsername = "anonymous"

var ErrBadRequest = errors.New("bad request")
var ErrUnauthorized = errors.New("unauthorized")

var httpAuthBasicRegexp = regexp.MustCompile(`^Basic\s+([a-zA-Z0-9+/]+={0,2})$`)
var httpAuthBearerRegexp = regexp.MustCompile(`^Bearer\s+([a-zA-Z0-9+/]+={0,2})$`)

// GetUsernameFromHttpRequest extracts the username from an HTTP request.
//
// Returns [AnonymousUsername] if the request does not contain any
// authorization header and there is an anonymous user in RBAC users.
func (e *Engine) GetUsernameFromHttpRequest(r *netHttp.Request) (string, error) {
	if r == nil {
		return "", ErrBadRequest
	}

	v := r.Header.Get("Authorization")

	// Basic auth.
	if matches := httpAuthBasicRegexp.FindStringSubmatch(v); len(matches) == 2 {
		encoded := matches[1]

		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return "", errors.Join(ErrBadRequest, err)
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return "", ErrBadRequest
		}

		username := parts[0]
		password := parts[1]

		if i := slices.IndexFunc(e.Users, func(user User) bool {
			return user.Name == username && user.IsPasswordValid(password)
		}); i >= 0 {
			return e.Users[i].Name, nil
		}

		return "", ErrUnauthorized
	}

	// Bearer token auth.
	if matches := httpAuthBearerRegexp.FindStringSubmatch(v); len(matches) == 2 {
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

		return "", ErrUnauthorized
	}

	return "", ErrUnauthorized
}
