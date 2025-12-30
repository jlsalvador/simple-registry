package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"net/http"
	"regexp"
	"strings"

	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func (s *ProxyDataStorage) matchProxy(repo string) *Proxy {
	for _, p := range s.proxies {
		for _, scope := range p.Scopes {
			if regexp.MustCompile(scope).MatchString(repo) {
				return &p
			}
		}
	}
	return nil
}

var manifestAccept = []string{
	"application/vnd.oci.image.manifest.v1+json",
	"application/vnd.oci.image.index.v1+json",
	"application/vnd.docker.distribution.manifest.v2+json",
	"application/vnd.docker.distribution.manifest.list.v2+json",
}

func newUpstreamRequest(
	proxy *Proxy,
	method string,
	url string,
	accept []string,
) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	if proxy.Username != "" {
		req.SetBasicAuth(proxy.Username, proxy.Password)
	}

	if len(accept) > 0 {
		req.Header.Set("Accept", strings.Join(accept, ", "))
	}

	return req, nil
}

func doUpstreamRequest(
	proxy *Proxy,
	req *http.Request,
) (*http.Response, error) {

	client := &http.Client{Timeout: proxy.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	// Upstream requires authentication.
	// Do auth and fetch bearer token.

	chHeader := resp.Header.Get("WWW-Authenticate")
	resp.Body.Close()

	ch, err := parseBearerChallenge(chHeader)
	if err != nil {
		return nil, err
	}

	token, err := fetchBearerToken(proxy, ch)
	if err != nil {
		return nil, err
	}

	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+token)

	return client.Do(req2)
}

func fetchManifestFromUpstream(
	proxy *Proxy,
	repo string,
	reference string,
) (io.ReadCloser, int64, error) {
	url := fmt.Sprintf(
		"%s/v2/%s/manifests/%s",
		strings.TrimRight(proxy.Url, "/"),
		repo,
		reference,
	)

	req, err := newUpstreamRequest(proxy, http.MethodGet, url, manifestAccept)
	if err != nil {
		return nil, -1, err
	}

	resp, err := doUpstreamRequest(proxy, req)
	if err != nil {
		return nil, -1, err
	}

	// Docker Hub could return 401 on manifest not found.
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return nil, -1, fs.ErrNotExist
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, -1, errors.Join(
			ErrUpstreamError,
			fmt.Errorf("upstream manifest error: %s", resp.Status),
		)
	}

	return resp.Body, resp.ContentLength, nil
}

func fetchBlobFromUpstream(
	proxy *Proxy,
	repo string,
	digest string,
) (io.ReadCloser, int64, error) {
	url := fmt.Sprintf(
		"%s/v2/%s/blobs/%s",
		strings.TrimRight(proxy.Url, "/"),
		repo,
		digest,
	)

	req, err := newUpstreamRequest(proxy, http.MethodGet, url, nil)
	if err != nil {
		return nil, -1, err
	}

	resp, err := doUpstreamRequest(proxy, req)
	if err != nil {
		return nil, -1, err
	}

	// Docker Hub could return 401 on blob not found.
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		return nil, -1, fs.ErrNotExist
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, -1, errors.Join(
			ErrUpstreamError,
			fmt.Errorf("upstream blob error: %s", resp.Status),
		)
	}

	return resp.Body, resp.ContentLength, nil
}

func fetchTagsFromUpstream(
	proxy *Proxy,
	repo string,
) ([]string, error) {
	url := fmt.Sprintf(
		"%s/v2/%s/tags/list",
		strings.TrimRight(proxy.Url, "/"),
		repo,
	)

	req, err := newUpstreamRequest(proxy, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := doUpstreamRequest(proxy, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fs.ErrNotExist
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Join(ErrUpstreamError, fmt.Errorf("upstream tags error: %s", resp.Status))
	}

	var out struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return out.Tags, nil
}

func fetchReferrersFromUpstream(
	proxy Proxy,
	repo,
	dgst string,
) (digests iter.Seq[string], err error) {
	url := fmt.Sprintf(
		"%s/v2/%s/referrers/%s",
		proxy.Url,
		repo,
		dgst,
	)

	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Accept", "application/vnd.oci.image.index.v1+json")

	if proxy.Username != "" {
		req.SetBasicAuth(proxy.Username, proxy.Password)
	}

	resp, err := doUpstreamRequest(&proxy, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, fs.ErrNotExist
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, errors.Join(ErrUpstreamError, fmt.Errorf("upstream error: %s", resp.Status))
	}

	index := registry.ImageIndexManifest{}
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, err
	}

	return func(yield func(string) bool) {
		for _, m := range index.Manifests {
			if !yield(m.Digest) {
				return
			}
		}
	}, nil
}

func fetchManifestDigestHEAD(
	proxy *Proxy,
	repo,
	reference string,
) (string, error) {
	url := fmt.Sprintf(
		"%s/v2/%s/manifests/%s",
		strings.TrimRight(proxy.Url, "/"),
		repo,
		reference,
	)

	req, err := newUpstreamRequest(proxy, http.MethodHead, url, manifestAccept)
	if err != nil {
		return "", err
	}

	resp, err := doUpstreamRequest(proxy, req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", errors.Join(ErrUpstreamError, fmt.Errorf("upstream HEAD failed: %s", resp.Status))
	}

	dgst := resp.Header.Get("Docker-Content-Digest")
	if dgst == "" {
		return "", errors.New("missing Docker-Content-Digest")
	}

	return dgst, nil
}
