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

package handler

import (
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

type headResponseWriter struct {
	http.ResponseWriter
}

func (h *headResponseWriter) Write(b []byte) (int, error) {
	// Discart response body for HEAD.
	return len(b), nil
}

type ServeMux struct {
	cfg config.Config
	mux *http.ServeMux
}

type route struct {
	Method     string
	Regexp     *regexp.Regexp
	ParamIndex map[string]int
	Handler    http.HandlerFunc
}

func setPathParams(r *http.Request, params map[string]int, match []string) {
	for name, idx := range params {
		r.SetPathValue(name, match[idx])
	}
}

// registerRoutes registers the routes for the HTTP server.
func (m *ServeMux) registerRoutes() {
	// IMPORTANT:
	// - Routes are matched in order.
	// - First RegExp match wins.
	// - More specific paths MUST appear before generic ones.
	var routes = []route{
		{
			http.MethodGet,
			regexp.MustCompile(`^/v2/?$`),
			nil,
			m.Index,
		},
		{
			http.MethodGet,
			regexp.MustCompile(`^/v2/_catalog/?$`),
			nil,
			m.CatalogList,
		},

		// Referrers:
		{
			http.MethodGet,
			regexp.MustCompile("^/v2/(?P<name>" + registry.RegExpName + ")/referrers/(?P<digest>" + registry.RegExpDigest + ")/?$"),
			nil,
			m.ReferrersGet,
		},

		// Tags:
		{
			http.MethodGet,
			regexp.MustCompile("^/v2/(?P<name>" + registry.RegExpName + ")/tags/list/?$"),
			nil,
			m.TagsList,
		},

		// Blobs:
		{
			http.MethodGet,
			regexp.MustCompile("^/v2/(?P<name>" + registry.RegExpName + ")/blobs/(?P<digest>" + registry.RegExpDigest + ")/?$"),
			nil,
			m.BlobsGet,
		},
		{
			http.MethodDelete,
			regexp.MustCompile("^/v2/(?P<name>" + registry.RegExpName + ")/blobs/(?P<digest>" + registry.RegExpDigest + ")/?$"),
			nil,
			m.BlobsDelete,
		},

		// Blobs uploads:
		{
			http.MethodPost,
			regexp.MustCompile("^/v2/(?P<name>" + registry.RegExpName + ")/blobs/uploads/?$"),
			nil,
			m.BlobsUploadsPost,
		},
		{
			http.MethodGet,
			regexp.MustCompile("^/v2/(?P<name>" + registry.RegExpName + ")/blobs/uploads/(?P<uuid>" + registry.RegExpUUID + ")/?$"),
			nil,
			m.BlobsUploadsGet,
		},
		{
			http.MethodPatch,
			regexp.MustCompile("^/v2/(?P<name>" + registry.RegExpName + ")/blobs/uploads/(?P<uuid>" + registry.RegExpUUID + ")/?$"),
			nil,
			m.BlobsUploadsPatch,
		},
		{
			http.MethodPut,
			regexp.MustCompile("^/v2/(?P<name>" + registry.RegExpName + ")/blobs/uploads/(?P<uuid>" + registry.RegExpUUID + ")/?$"),
			nil,
			m.BlobsUploadsPut,
		},
		{
			http.MethodDelete,
			regexp.MustCompile("^/v2/(?P<name>" + registry.RegExpName + ")/blobs/uploads/(?P<uuid>" + registry.RegExpUUID + ")/?$"),
			nil,
			m.BlobsUploadsDelete,
		},

		// Manifests:
		{
			http.MethodGet,
			regexp.MustCompile("^/v2/(?P<name>" + registry.RegExpName + ")/manifests/(?P<reference>(?:" + registry.RegExpTag + ")|(?:" + registry.RegExpDigest + "))/?$"),
			nil,
			m.ManifestsGet,
		},
		{
			http.MethodPut,
			regexp.MustCompile("^/v2/(?P<name>" + registry.RegExpName + ")/manifests/(?P<reference>(?:" + registry.RegExpTag + ")|(?:" + registry.RegExpDigest + "))/?$"),
			nil,
			m.ManifestsPut,
		},
		{
			http.MethodDelete,
			regexp.MustCompile("^/v2/(?P<name>" + registry.RegExpName + ")/manifests/(?P<reference>(?:" + registry.RegExpTag + ")|(?:" + registry.RegExpDigest + "))/?$"),
			nil,
			m.ManifestsDelete,
		},
	}
	for i, rt := range routes {
		paramIndex := make(map[string]int)

		for idx, name := range rt.Regexp.SubexpNames() {
			if name != "" {
				paramIndex[name] = idx
			}
		}

		routes[i].ParamIndex = paramIndex
	}

	m.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		pathMatched := false
		allowed := map[string]struct{}{}

		for _, route := range routes {
			match := route.Regexp.FindStringSubmatch(r.URL.Path)
			if match == nil {
				continue
			}

			pathMatched = true
			allowed[route.Method] = struct{}{}
			if route.Method == http.MethodGet {
				allowed[http.MethodHead] = struct{}{}
			}

			if r.Method == http.MethodHead && route.Method == http.MethodGet {
				// Do GET as HEAD.
				setPathParams(r, route.ParamIndex, match)

				// Discart response body for HEAD.
				hw := &headResponseWriter{ResponseWriter: w}

				route.Handler(hw, r)
				return
			}

			if r.Method != route.Method {
				continue
			}

			setPathParams(r, route.ParamIndex, match)
			route.Handler(w, r)
			return
		}

		if pathMatched {
			methods := make([]string, 0, len(allowed))
			for m := range allowed {
				methods = append(methods, m)
			}
			slices.Sort(methods)
			w.Header().Set("Allow", strings.Join(methods, ", "))
			w.WriteHeader(http.StatusMethodNotAllowed)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

// NewHandler creates a new HTTP handler that complies with the
// [Docker Registry API v2.0 specification].
//
// [Docker Registry API v2.0 specification]: https://github.com/opencontainers/distribution-spec/blob/v1.1.1/spec.md
func NewHandler(cfg config.Config) http.Handler {
	mux := &ServeMux{
		cfg: cfg,
		mux: http.NewServeMux(),
	}
	mux.registerRoutes()

	return mux.mux
}
