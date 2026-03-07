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
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCatalog(t *testing.T) {
	tests := []struct {
		name     string
		requests []testRequestBuilder
	}{
		{
			name: "successful index",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodGet, "/v2/", nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "successful index without auth because anonymous user is enabled",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodGet, "/v2/", nil)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "successful catalog list with empty repository",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodGet, "/v2/_catalog", nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "successful catalog list with some images",
			requests: []testRequestBuilder{
				{
					func(prevResp *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", bytes.NewReader([]byte("hola")))
						r.Header.Set("Authorization", testAuthHeader)
						r.Header.Set("Content-Type", "application/octet-stream")
						r.Header.Set("Content-Length", "4")
						r.Header.Set("Content-Range", "0-4")
						return r
					},
					http.StatusAccepted,
				},
				testRequestBuilderPutManifests,
				{
					func(_ *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodGet, "/v2/_catalog", nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "unsuccessful catalog list without auth",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						r := httptest.NewRequest(http.MethodGet, "/v2/_catalog", nil)
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
