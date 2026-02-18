// Copyright 2026 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
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

package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/digest"
	"github.com/jlsalvador/simple-registry/pkg/registry"
)

var testManifest = registry.ImageManifest{
	SchemaVersion: 2,
	MediaType:     registry.MediaTypeOCIImageManifest,
	Config: registry.DescriptorManifest{
		MediaType: registry.MediaTypeOCIImageConfig,
		Size:      4,
		Digest:    testBlobDigest,
	},
	Layers: []registry.DescriptorManifest{
		{
			MediaType: "application/vnd.docker.image.rootfs.diff.tar.gzip",
			Size:      4,
			Digest:    testBlobDigest,
		},
	},
}
var testManifestDigest = func() string {
	d, _ := json.Marshal(testManifest)
	h, _ := digest.NewHasher("sha256")
	h.Write(d)
	return h.GetHashAsString()
}()

var testRequestBuilderPutManifests = testRequestBuilder{
	func(_ *http.Response) *http.Request {
		manifestJSON, _ := json.Marshal(testManifest)
		r := httptest.NewRequest(http.MethodPut, "/v2/myrepo/myimage/manifests/latest", bytes.NewReader(manifestJSON))
		r.Header.Set("Authorization", testAuthHeader)
		r.Header.Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		return r
	},
	http.StatusCreated,
}

func TestManifests(t *testing.T) {
	tests := []struct {
		name     string
		requests []testRequestBuilder
	}{
		{
			name: "successful manifests PUT",
			requests: []testRequestBuilder{
				testRequestBuilderPutManifests,
			},
		},
		{
			name: "unsuccessful manifests PUT without auth",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						manifestJSON, _ := json.Marshal(testManifest)
						r := httptest.NewRequest(http.MethodPut, "/v2/myrepo/myimage/manifests/latest", bytes.NewReader(manifestJSON))
						r.Header.Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
						return r
					},
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "unsuccessful manifests PUT invalid JSON",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						manifestJSON := []byte("{invalid json}")
						r := httptest.NewRequest(http.MethodPut, "/v2/myrepo/myimage/manifests/latest", bytes.NewReader(manifestJSON))
						r.Header.Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					//FIXME are you sure this is the correct status code for invalid manifests JSON?
					http.StatusInternalServerError,
				},
			},
		},
		{
			name: "successful manifests GET by tag",
			requests: []testRequestBuilder{
				testRequestBuilderPutManifests,
				{
					func(prevResp *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodGet, "/v2/myrepo/myimage/manifests/latest", nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "successful manifests GET by digest",
			requests: []testRequestBuilder{
				testRequestBuilderPutManifests,
				{
					func(prevResp *http.Response) *http.Request {
						repo := "myrepo/myimage"
						digest := prevResp.Header.Get(testHeaderDockerContentDigest)
						url := fmt.Sprintf("/v2/%s/manifests/%s", repo, digest)
						r := httptest.NewRequest(http.MethodGet, url, nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "unsuccessful manifests GET without auth",
			requests: []testRequestBuilder{
				{
					func(prevResp *http.Response) *http.Request {
						repo := "myrepo/myimage"
						digest := "sha256:unknown"
						url := fmt.Sprintf("/v2/%s/manifests/%s", repo, digest)
						r := httptest.NewRequest(http.MethodGet, url, nil)
						return r
					},
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "unsuccessful unknown manifests GET",
			requests: []testRequestBuilder{
				{
					func(prevResp *http.Response) *http.Request {
						repo := "myrepo/myimage"
						digest := "sha256:unknown"
						url := fmt.Sprintf("/v2/%s/manifests/%s", repo, digest)
						r := httptest.NewRequest(http.MethodGet, url, nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusNotFound,
				},
			},
		},
		{
			name: "successful manifests DELETE by tag",
			requests: []testRequestBuilder{
				testRequestBuilderPutManifests,
				{
					func(prevResp *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodDelete, "/v2/myrepo/myimage/manifests/latest", nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusAccepted,
				},
			},
		},
		{
			name: "unsuccessful manifests DELETE without auth",
			requests: []testRequestBuilder{
				testRequestBuilderPutManifests,
				{
					func(prevResp *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodDelete, "/v2/myrepo/myimage/manifests/latest", nil)
						return r
					},
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "unsuccessful manifests DELETE unknown tag name",
			requests: []testRequestBuilder{
				{
					func(prevResp *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodDelete, "/v2/myrepo/myimage/manifests/unknown", nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusNotFound,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mux := testSetupTestServeMux(t)

			var prev *http.Response
			for i, trb := range tt.requests {
				w := httptest.NewRecorder()
				r := trb.requestFn(prev)
				mux.ServeHTTP(w, r)
				prev = w.Result()

				if prev.StatusCode != trb.statusCode {
					body, _ := io.ReadAll(prev.Body)
					t.Errorf("step: %d, want %d, got %d. body: %s", i+1, trb.statusCode, prev.StatusCode, string(body))
				}
			}
		})
	}
}
