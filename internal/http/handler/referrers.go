package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	netHttp "net/http"

	"github.com/jlsalvador/simple-registry/pkg/rbac"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

type genericManifest struct {
	MediaType    string  `json:"mediaType"`
	ArtifactType *string `json:"artifactType,omitempty"`
	Config       *struct {
		MediaType string `json:"mediaType"`
	} `json:"config,omitempty"`
	Annotations map[string]string `json:"annotations"`
}

func isSkipableManifest(
	filterByArtifactType string,
	blobManifest genericManifest,
) bool {
	if filterByArtifactType == "" {
		return false
	}

	if blobManifest.ArtifactType != nil {
		// Modern artifact
		if *blobManifest.ArtifactType != filterByArtifactType {
			return true
		}
	} else {
		// Legacy artifact
		if blobManifest.MediaType != "application/vnd.oci.image.manifest.v1+json" {
			return true
		}

		if blobManifest.Config == nil || blobManifest.Config.MediaType != filterByArtifactType {
			return true
		}
	}

	return false
}

func (m *ServeMux) ReferrersGet(
	w netHttp.ResponseWriter,
	r *netHttp.Request,
) {
	username, err := m.authenticate(w, r)
	if err != nil {
		return
	}

	// "repo" must be a valid repository name.
	repo := r.PathValue("name")
	if !registry.RegExprName.MatchString(repo) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// "digest" must be a valid digest.
	dgst := r.PathValue("digest")
	if !registry.RegExprDigest.MatchString(dgst) {
		w.WriteHeader(netHttp.StatusBadRequest)
		return
	}

	// Check if the user is allowed to pull this manifest.
	if !m.cfg.Rbac.IsAllowed(username, "manifests", repo, netHttp.MethodGet) {
		if username == rbac.AnonymousUsername {
			w.Header().Set("WWW-Authenticate", m.cfg.WWWAuthenticate)
		}

		w.WriteHeader(netHttp.StatusUnauthorized)
		return
	}

	referrers, err := m.cfg.Data.ReferrersGet(repo, dgst)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	filterByArtifactType := r.URL.Query().Get("artifactType")

	index := registry.NewImageIndexManifest()
	if referrers != nil {
		for ref := range referrers {
			blob, size, err := m.cfg.Data.BlobsGet(repo, ref)
			if err != nil {
				w.WriteHeader(netHttp.StatusInternalServerError)
				return
			}
			defer blob.Close()

			blobManifest := genericManifest{}
			if err := json.NewDecoder(blob).Decode(&blobManifest); err != nil {
				continue
			}

			if !isSkipableManifest(filterByArtifactType, blobManifest) {
				index.Manifests = append(index.Manifests, registry.DescriptorManifest{
					MediaType:   blobManifest.MediaType,
					Digest:      ref,
					Size:        size,
					Annotations: blobManifest.Annotations,
				})
			}
		}
	}

	data, err := json.Marshal(index)
	if err != nil {
		w.WriteHeader(netHttp.StatusInternalServerError)
		return
	}

	header := w.Header()
	if filterByArtifactType != "" {
		header.Set("OCI-Filters-Applied", "artifactType")
	}
	header.Set("Content-Type", "application/vnd.oci.image.index.v1+json")
	header.Set("Content-Length", fmt.Sprint(len(data)))
	w.WriteHeader(netHttp.StatusOK)
	w.Write(data)
}
