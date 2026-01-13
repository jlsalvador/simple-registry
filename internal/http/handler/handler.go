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
	"errors"
	"net/http"
	"slices"
	"strings"

	"github.com/jlsalvador/simple-registry/internal/config"
	httpErrors "github.com/jlsalvador/simple-registry/pkg/http/errors"
	"github.com/jlsalvador/simple-registry/pkg/http/log"
	"github.com/jlsalvador/simple-registry/pkg/http/route"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

type ServeMux struct {
	cfg config.Config
	mux *http.ServeMux
}

func (m *ServeMux) authenticate(w http.ResponseWriter, r *http.Request) (username string, err error) {
	username, err = m.cfg.Rbac.GetUsernameFromHttpRequest(r)
	if err != nil {
		if errors.Is(err, httpErrors.ErrUnauthorized) {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
		}
		w.WriteHeader(httpErrors.StatusCodeFromError(err))
		return "", err
	}
	return username, nil
}

// registerRoutes registers the routes for the HTTP server.
func (m *ServeMux) registerRoutes() {
	exprName := strings.TrimPrefix(strings.TrimSuffix(registry.ExprName, "$"), "^")
	exprDigest := strings.TrimPrefix(strings.TrimSuffix(registry.ExprDigest, "$"), "^")
	exprTag := strings.TrimPrefix(strings.TrimSuffix(registry.ExprTag, "$"), "^")
	exprUUID := strings.TrimPrefix(strings.TrimSuffix(registry.ExprUUID, "$"), "^")

	// IMPORTANT:
	// - Routes are matched in order.
	// - First RegExp match wins.
	// - More specific paths MUST appear before generic ones.
	var routes = []route.Route{
		route.NewRoute(
			http.MethodGet,
			`^/v2/?$`,
			m.Index,
		),
		route.NewRoute(
			http.MethodGet,
			`^/v2/_catalog/?$`,
			m.CatalogList,
		),

		// Referrers:
		route.NewRoute(
			http.MethodGet,
			"^/v2/(?P<name>"+exprName+")/referrers/(?P<digest>"+exprDigest+")/?$",
			m.ReferrersGet,
		),

		// Tags:
		route.NewRoute(
			http.MethodGet,
			"^/v2/(?P<name>"+exprName+")/tags/list/?$",
			m.TagsList,
		),

		// Blobs:
		route.NewRoute(
			http.MethodGet,
			"^/v2/(?P<name>"+exprName+")/blobs/(?P<digest>"+exprDigest+")/?$",
			m.BlobsGet,
		),
		route.NewRoute(
			http.MethodDelete,
			"^/v2/(?P<name>"+exprName+")/blobs/(?P<digest>"+exprDigest+")/?$",
			m.BlobsDelete,
		),

		// Blobs uploads:
		route.NewRoute(
			http.MethodPost,
			"^/v2/(?P<name>"+exprName+")/blobs/uploads/?$",
			m.BlobsUploadsPost,
		),
		route.NewRoute(
			http.MethodGet,
			"^/v2/(?P<name>"+exprName+")/blobs/uploads/(?P<uuid>"+exprUUID+")/?$",
			m.BlobsUploadsGet,
		),
		route.NewRoute(
			http.MethodPatch,
			"^/v2/(?P<name>"+exprName+")/blobs/uploads/(?P<uuid>"+exprUUID+")/?$",
			m.BlobsUploadsPatch,
		),
		route.NewRoute(
			http.MethodPut,
			"^/v2/(?P<name>"+exprName+")/blobs/uploads/(?P<uuid>"+exprUUID+")/?$",
			m.BlobsUploadsPut,
		),
		route.NewRoute(
			http.MethodDelete,
			"^/v2/(?P<name>"+exprName+")/blobs/uploads/(?P<uuid>"+exprUUID+")/?$",
			m.BlobsUploadsDelete,
		),

		// Manifests:
		route.NewRoute(
			http.MethodGet,
			"^/v2/(?P<name>"+exprName+")/manifests/(?P<reference>(?:"+exprTag+")|(?:"+exprDigest+"))/?$",
			m.ManifestsGet,
		),
		route.NewRoute(
			http.MethodPut,
			"^/v2/(?P<name>"+exprName+")/manifests/(?P<reference>(?:"+exprTag+")|(?:"+exprDigest+"))/?$",
			m.ManifestsPut,
		),
		route.NewRoute(
			http.MethodDelete,
			"^/v2/(?P<name>"+exprName+")/manifests/(?P<reference>(?:"+exprTag+")|(?:"+exprDigest+"))/?$",
			m.ManifestsDelete,
		),
	}

	if m.cfg.IsWebUIEnabled {
		routes = append(routes, route.NewRoute(
			http.MethodGet,
			"^/web",
			m.Web,
		))
	}

	m.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		allowed := map[string]struct{}{}

		for _, route := range routes {
			route.Handler(w, r)

			if route.IsMatchUrl && route.IsMatchMethod {
				return
			}

			// Save the route method if the URL matches but not the method, for
			// the later Allow HTTP header.
			if route.IsMatchUrl && !route.IsMatchMethod {
				allowed[route.Method] = struct{}{}
				if route.Method == http.MethodGet {
					allowed[http.MethodHead] = struct{}{}
				}
			}
		}

		if len(allowed) > 0 {
			// Print Allow HTTP Header.
			methods := make([]string, 0, len(allowed))
			for m := range allowed {
				methods = append(methods, m)
			}
			slices.Sort(methods)
			w.Header().Set("Allow", strings.Join(methods, ", "))
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// 404.
		w.WriteHeader(http.StatusNotFound)
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

	return log.LoggingMiddleware(mux.mux)
}
