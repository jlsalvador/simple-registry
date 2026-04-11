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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/digest"
)

func TestBlobs(t *testing.T) {
	// Common setup.

	payload := []byte("hello world")
	dgst, err := digest.NewHasher("sha256")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dgst.Write(payload); err != nil {
		t.Fatal(err)
	}
	validHash := dgst.GetHashAsString()
	payloadLen := fmt.Sprint(len(payload))

	// Common testRequestBuilders
	rbPost := testRequestBuilder{
		func(_ *http.Response) *http.Request {
			repo := "myrepo/myimage"
			url := fmt.Sprintf("/v2/%s/blobs/uploads", repo)
			r := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
			q := r.URL.Query()
			q.Add("digest", "sha256:"+validHash)
			r.URL.RawQuery = q.Encode()
			r.SetBasicAuth(testUser, testPwd)
			r.Header.Set("Content-Type", "application/octet-stream")
			r.Header.Set("Content-Length", payloadLen)
			r.Header.Set("Content-Range", fmt.Sprintf("0-%d", len(payload)))
			return r
		},
		http.StatusCreated,
	}

	tests := []struct {
		name     string
		requests []testRequestBuilder
	}{
		{
			name: "successful GET blob",
			requests: []testRequestBuilder{
				rbPost,
				{
					func(_ *http.Response) *http.Request {
						repo := "myrepo/myimage"
						url := fmt.Sprintf("/v2/%s/blobs/sha256:%s", repo, validHash)
						r := httptest.NewRequest(http.MethodGet, url, nil)
						r.SetBasicAuth(testUser, testPwd)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "unsuccessful GET blob without auth",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						repo := "myrepo/myimage"
						url := fmt.Sprintf("/v2/%s/blobs/sha256:%s", repo, validHash)
						r := httptest.NewRequest(http.MethodGet, url, nil)
						return r
					},
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "unsuccessful GET unknown blob",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						repo := "myrepo/myimage"
						url := fmt.Sprintf("/v2/%s/blobs/sha256:%s", repo, "unknown")
						r := httptest.NewRequest(http.MethodGet, url, nil)
						r.SetBasicAuth(testUser, testPwd)
						return r
					},
					http.StatusNotFound,
				},
			},
		},
		{
			name: "successful DELETE blob",
			requests: []testRequestBuilder{
				rbPost,
				{
					func(_ *http.Response) *http.Request {
						repo := "myrepo/myimage"
						url := fmt.Sprintf("/v2/%s/blobs/sha256:%s", repo, validHash)
						r := httptest.NewRequest(http.MethodDelete, url, nil)
						r.SetBasicAuth(testUser, testPwd)
						return r
					},
					http.StatusAccepted,
				},
			},
		},
		{
			name: "unsuccessful DELETE blob without auth",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						repo := "myrepo/myimage"
						url := fmt.Sprintf("/v2/%s/blobs/sha256:%s", repo, validHash)
						r := httptest.NewRequest(http.MethodDelete, url, nil)
						return r
					},
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "unsuccessful DELETE unknown blob",
			requests: []testRequestBuilder{
				rbPost,
				{
					func(_ *http.Response) *http.Request {
						repo := "myrepo/myimage"
						url := fmt.Sprintf("/v2/%s/blobs/sha256:%s", repo, "unknown")
						r := httptest.NewRequest(http.MethodDelete, url, nil)
						r.SetBasicAuth(testUser, testPwd)
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
