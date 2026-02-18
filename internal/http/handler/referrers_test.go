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

	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func TestReferrers(t *testing.T) {
	// reqPutReferrer returns a request builder that pushes a referrer manifest
	// whose Subject points to testManifestDigest. The tag is a simple slug so
	// it never contains slashes that would confuse the router.
	reqPutReferrer := func(tag, artifactType string) func(*http.Response) *http.Request {
		return func(_ *http.Response) *http.Request {
			referrerManifest := map[string]any{
				"schemaVersion": 2,
				"mediaType":     string(registry.MediaTypeOCIImageManifest),
				"artifactType":  artifactType,
				"config": map[string]any{
					"mediaType": string(registry.MediaTypeOCIImageConfig),
					"size":      4,
					"digest":    "sha256:" + testBlobDigest,
				},
				"layers": []any{},
				"subject": map[string]any{
					"mediaType": string(registry.MediaTypeOCIImageManifest),
					"size":      4,
					"digest":    "sha256:" + testManifestDigest,
				},
			}

			manifestJSON, _ := json.Marshal(referrerManifest)
			url := fmt.Sprintf("/v2/myrepo/myimage/manifests/%s", tag)
			r := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(manifestJSON))
			r.Header.Set("Authorization", testAuthHeader)
			r.Header.Set("Content-Type", string(registry.MediaTypeOCIImageManifest))
			return r
		}
	}

	tests := []struct {
		name     string
		requests []testRequestBuilder
	}{
		{
			name: "successful referrers GET empty list",
			requests: []testRequestBuilder{
				// Ensure the repository exists before querying referrers.
				// OCI spec: "Assuming a repository is found, this request MUST
				// return a 200 OK". Querying a non-existent repo may return 404.
				testRequestBuilderPutManifests,
				// No referrer has been pushed, so the list should be empty.
				{
					func(_ *http.Response) *http.Request {
						url := fmt.Sprintf("/v2/myrepo/myimage/referrers/sha256:%s", testManifestDigest)
						r := httptest.NewRequest(http.MethodGet, url, nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "successful referrers GET filled list",
			requests: []testRequestBuilder{
				// 1. Push the subject manifest.
				testRequestBuilderPutManifests,
				// 2. Push a referrer pointing to the subject.
				{
					reqPutReferrer("referrer-sbom", "application/vnd.example.sbom"),
					http.StatusCreated,
				},
				// 3. GET referrers for the subject digest — list should be non-empty.
				{
					func(_ *http.Response) *http.Request {
						url := fmt.Sprintf("/v2/myrepo/myimage/referrers/sha256:%s", testManifestDigest)
						r := httptest.NewRequest(http.MethodGet, url, nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "successful referrers GET with artifactType filter",
			requests: []testRequestBuilder{
				testRequestBuilderPutManifests,
				{
					reqPutReferrer("referrer-sig", "application/vnd.example.signature"),
					http.StatusCreated,
				},
				{
					func(_ *http.Response) *http.Request {
						url := fmt.Sprintf("/v2/myrepo/myimage/referrers/sha256:%s", testManifestDigest)
						r := httptest.NewRequest(http.MethodGet, url, nil)
						q := r.URL.Query()
						q.Add("artifactType", "application/vnd.example.signature")
						r.URL.RawQuery = q.Encode()
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "unsuccessful referrers GET without auth",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						url := fmt.Sprintf("/v2/myrepo/myimage/referrers/sha256:%s", testManifestDigest)
						r := httptest.NewRequest(http.MethodGet, url, nil)
						return r
					},
					http.StatusUnauthorized,
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
