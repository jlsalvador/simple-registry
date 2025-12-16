package rbac

import (
	"net/http"
	"slices"
	"strings"
)

func ParseVerbs(verbs []string) ([]string, error) {
	m := map[string]struct{}{}

	for _, s := range verbs {
		s = strings.ToUpper(strings.TrimSpace(s))

		if s == "*" {
			m[http.MethodHead] = struct{}{}
			m[http.MethodGet] = struct{}{}
			m[http.MethodPost] = struct{}{}
			m[http.MethodPut] = struct{}{}
			m[http.MethodPatch] = struct{}{}
			m[http.MethodDelete] = struct{}{}
			m[http.MethodConnect] = struct{}{}
			m[http.MethodOptions] = struct{}{}
			m[http.MethodTrace] = struct{}{}
			break
		}

		switch s {
		case http.MethodHead, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			m[s] = struct{}{}

		default:
			return nil, ErrInvalidVerb
		}
	}

	r := []string{}
	for k := range m {
		r = append(r, k)
	}

	slices.Sort(r)

	return r, nil
}
