package handler

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"regexp"

	"github.com/jlsalvador/simple-registry/pkg/digest"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func (m *ServeMux) ReferrersGet(
	w http.ResponseWriter,
	r *http.Request,
) {
	username, err := m.cfg.Rbac.GetUsernameFromHttpRequest(r)
	if err != nil {
		w.Header().Set("WWW-Authenticate", "Basic realm=\"simple-registry\"")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// "repo" must be a valid repository name.
	repo := r.PathValue("name")
	if !regexp.MustCompile(registry.RegExpName).MatchString(repo) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// "digest" must be a valid digest.
	dgst := r.PathValue("digest")
	if _, _, err := digest.Parse(dgst); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Check if the user is allowed to pull this manifest.
	if !m.cfg.Rbac.IsAllowed(username, "manifests", repo, http.MethodGet) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	f, size, err := m.cfg.Data.ReferrersGet(repo, dgst)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer f.Close()

	header := w.Header()
	header.Set("Content-Type", "application/vnd.oci.image.index.v1+json")
	header.Set("Content-Length", fmt.Sprint(size))
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, f)
}
