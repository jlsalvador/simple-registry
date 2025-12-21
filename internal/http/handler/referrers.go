package handler

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	netHttp "net/http"
	"regexp"

	"github.com/jlsalvador/simple-registry/pkg/digest"
	httpErrors "github.com/jlsalvador/simple-registry/pkg/http/errors"
	"github.com/jlsalvador/simple-registry/pkg/rbac"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func (m *ServeMux) ReferrersGet(
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	username, err := m.cfg.Rbac.GetUsernameFromHttpRequest(r)
	if err, ok := err.(*httpErrors.HttpError); ok {
		w.WriteHeader(err.Status)
		return
	}

	// "repo" must be a valid repository name.
	repo := r.PathValue("name")
	if !regexp.MustCompile(registry.RegExpName).MatchString(repo) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// "digest" must be a valid digest.
	dgst := r.PathValue("digest")
	if _, _, err := digest.Parse(dgst); err != nil {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// Check if the user is allowed to pull this manifest.
	if !m.cfg.Rbac.IsAllowed(username, "manifests", repo, netHttp.MethodGet) {
		if username == rbac.AnonymousUsername {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		} else {
			w.WriteHeader(netHttp.StatusUnauthorized)
			return
		}
	}

	f, size, err := m.cfg.Data.ReferrersGet(repo, dgst)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(netHttp.StatusNotFound)
			return
		}
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}
	defer f.Close()

	header := w.Header()
	header.Set("Content-Type", "application/vnd.oci.image.index.v1+json")
	header.Set("Content-Length", fmt.Sprint(size))
	w.WriteHeader(netHttp.StatusOK)
	_, _ = io.Copy(w, f)
}
