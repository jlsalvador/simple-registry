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

package route

import (
	"net/http"
	"regexp"
)

type Route struct {
	Method    string
	rePattern *regexp.Regexp
	next      http.HandlerFunc

	// Used to precalculate the path parameters.
	paramIndex map[string]int
}

func (rt *Route) setPathParams(r *http.Request, match []string) {
	for name, idx := range rt.paramIndex {
		r.SetPathValue(name, match[idx])
	}
}

func (rt *Route) Handler(w http.ResponseWriter, r *http.Request) (matchUrl bool, matchMethod bool) {
	match := rt.rePattern.FindStringSubmatch(r.URL.Path)
	if match == nil {
		return false, false
	}

	// Method HEAD == GET, but without body.
	if r.Method == http.MethodHead && rt.Method == http.MethodGet {
		// Discard response body for HEAD.
		w = &HeadResponseWriter{ResponseWriter: w}
		r.Method = http.MethodGet
	}

	if r.Method != rt.Method {
		return true, false
	}

	rt.setPathParams(r, match)
	rt.next(w, r)

	return true, true
}

func NewRoute(method string, pattern string, next http.HandlerFunc) Route {
	r := Route{
		Method:    method,
		rePattern: regexp.MustCompile(pattern),
		next:      next,
	}

	pI := make(map[string]int)
	for idx, name := range r.rePattern.SubexpNames() {
		if name != "" {
			pI[name] = idx
		}
	}
	r.paramIndex = pI

	return r
}
