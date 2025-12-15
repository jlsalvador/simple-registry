package rbac

import (
	"encoding/base64"
	"errors"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"time"
)

var httpAuthBasicRegexp = regexp.MustCompile(`^Basic\s+([a-zA-Z0-9+/]+={0,2})$`)
var httpAuthBearerRegexp = regexp.MustCompile(`^Bearer\s+([a-zA-Z0-9+/]+={0,2})$`)

func (e *Engine) GetUsernameFromHttpRequest(r *http.Request) (string, error) {
	if r == nil {
		return "", ErrHttpRequestInvalid
	}

	v := r.Header.Get("Authorization")

	// Basic auth.
	if matches := httpAuthBasicRegexp.FindStringSubmatch(v); len(matches) == 2 {
		encoded := matches[1]

		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return "", errors.Join(ErrBasicAuthInvalid, err)
		}

		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return "", ErrBasicAuthInvalid
		}

		username := parts[0]
		password := parts[1]

		if i := slices.IndexFunc(e.Users, func(user User) bool {
			return user.Name == username && user.IsPasswordValid([]byte(password))
		}); i >= 0 {
			return e.Users[i].Name, nil
		}

		return "", ErrAuthCredentialsInvalid
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

		return "", ErrAuthCredentialsInvalid
	}

	// Check for anonymous user.
	for _, u := range e.Users {
		if u.Name == "" {
			return "", nil
		}
	}

	return "", ErrAuthCredentialsInvalid
}
