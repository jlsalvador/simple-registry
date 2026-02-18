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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTags(t *testing.T) {
	tests := []struct {
		name     string
		requests []testRequestBuilder
	}{
		{
			name: "successful tags GET",
			requests: []testRequestBuilder{
				testRequestBuilderPutManifests,
				{
					func(prevResp *http.Response) *http.Request {
						repo := "myrepo/myimage"
						url := fmt.Sprintf("/v2/%s/tags/list", repo)
						r := httptest.NewRequest(http.MethodGet, url, nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusOK,
				},
			},
		},
		{
			name: "unsuccessful tags GET unknown repo",
			requests: []testRequestBuilder{
				{
					func(prevResp *http.Response) *http.Request {
						repo := "unknownrepo"
						url := fmt.Sprintf("/v2/%s/tags/list", repo)
						r := httptest.NewRequest(http.MethodGet, url, nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusNotFound,
				},
			},
		},
		{
			name: "unsuccessful tags GET without auth",
			requests: []testRequestBuilder{
				testRequestBuilderPutManifests,
				{
					func(prevResp *http.Response) *http.Request {
						repo := "myrepo/myimage"
						url := fmt.Sprintf("/v2/%s/tags/list", repo)
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
