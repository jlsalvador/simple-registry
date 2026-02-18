// Copyright 2026 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"strings"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/digest"
)

var (
	testBlob       = []byte("hello world")
	testBlobDigest = func() string {
		dgst, _ := digest.NewHasher("sha256")
		dgst.Write(testBlob)
		return dgst.GetHashAsString()
	}()
)

func TestBlobsUploads(t *testing.T) {
	reqPost := func(repo, auth string, queries map[string]string) func(prevResp *http.Response) *http.Request {
		return func(_ *http.Response) *http.Request {
			url := fmt.Sprintf("/v2/%s/blobs/uploads/", repo)
			r := httptest.NewRequest(http.MethodPost, url, nil)
			for k, v := range queries {
				q := r.URL.Query()
				q.Add(k, v)
				r.URL.RawQuery = q.Encode()
			}
			if auth != "" {
				r.Header.Set("Authorization", auth)
			}
			return r
		}
	}
	reqPut := func(repo, auth string, queries map[string]string, body []byte) func(prevResp *http.Response) *http.Request {
		return func(prev *http.Response) *http.Request {
			uuid := prev.Header.Get(testHeaderDockerUploadUUID)
			url := fmt.Sprintf("/v2/%s/blobs/uploads/%s", repo, uuid)
			r := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(body))
			for k, v := range queries {
				q := r.URL.Query()
				q.Add(k, v)
				r.URL.RawQuery = q.Encode()
			}
			if auth != "" {
				r.Header.Set("Authorization", auth)
			}
			r.Header.Set("Content-Type", "application/octet-stream")
			r.Header.Set("Content-Length", fmt.Sprint(len(body)))
			if len(body) > 0 {
				r.Header.Set("Content-Range", fmt.Sprintf("0-%d", len(body)))
			}
			return r
		}
	}

	tests := []struct {
		name     string
		requests []testRequestBuilder
	}{
		{
			name: "successful POST then PUT initiation",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", testAuthHeader, map[string]string{}),
					http.StatusAccepted,
				},
				{
					reqPut(
						"myrepo/myimage",
						testAuthHeader,
						map[string]string{"digest": "sha256:" + testBlobDigest},
						testBlob,
					),
					http.StatusCreated,
				},
			},
		},
		{
			name: "single successful POST",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						repo := "myrepo/myimage"
						url := fmt.Sprintf("/v2/%s/blobs/uploads", repo)
						r := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(testBlob))
						q := r.URL.Query()
						q.Add("digest", "sha256:"+testBlobDigest)
						r.URL.RawQuery = q.Encode()
						r.Header.Set("Authorization", testAuthHeader)
						r.Header.Set("Content-Type", "application/octet-stream")
						r.Header.Set("Content-Length", fmt.Sprint(len(testBlob)))
						r.Header.Set("Content-Range", "0-"+fmt.Sprint(len(testBlob)))
						return r
					},
					http.StatusCreated,
				},
			},
		},
		{
			name: "unauthorized, no auth header",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", "", map[string]string{}),
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "forbidden, for PUT request",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", testAuthHeader, map[string]string{}),
					http.StatusAccepted,
				},
				{
					reqPut(
						"myrepo/myimage",
						testAuthHeaderWithoutPerms,
						map[string]string{"digest": "sha256:" + testBlobDigest},
						testBlob,
					),
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "unsuccessful PUT with invalid digest",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", testAuthHeader, map[string]string{}),
					http.StatusAccepted,
				},
				{
					reqPut(
						"myrepo/myimage",
						testAuthHeader,
						map[string]string{"digest": "invalid"},
						testBlob,
					),
					http.StatusBadRequest,
				},
			},
		},
		{
			name: "invalid repository name",
			requests: []testRequestBuilder{
				{
					reqPost("invalid%20repo%20name%21", testAuthHeader, map[string]string{}),
					http.StatusNotFound,
				},
			},
		},
		{
			name: "invalid digest",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", testAuthHeader, map[string]string{}),
					http.StatusAccepted,
				},
				{
					reqPut(
						"myrepo/myimage",
						testAuthHeader,
						map[string]string{"digest": "sha256:abc"},
						testBlob,
					),
					http.StatusBadRequest,
				},
			},
		},
		{
			name: "unknown digest",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", testAuthHeader, map[string]string{}),
					http.StatusAccepted,
				},
				{
					reqPut(
						"myrepo/myimage",
						testAuthHeader,
						map[string]string{"digest": "sha256:f1234d75178d892a133a410355a5a990cf75d2f33eba25d575943d4df632f3a4"},
						testBlob,
					),
					http.StatusBadRequest,
				},
			},
		},
		{
			name: "successful mount",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", testAuthHeader, map[string]string{}),
					http.StatusAccepted,
				},
				{
					reqPut(
						"myrepo/myimage",
						testAuthHeader,
						map[string]string{"digest": "sha256:" + testBlobDigest},
						testBlob,
					),
					http.StatusCreated,
				},
				{
					reqPost(
						"anotherrepo/otherimage",
						testAuthHeader,
						map[string]string{
							"mount": "sha256:" + testBlobDigest,
							"from":  "myrepo/myimage",
						},
					),
					http.StatusCreated,
				},
			},
		},
		{
			name: "mount not found",
			requests: []testRequestBuilder{
				{
					reqPost(
						"myrepo/myimage",
						testAuthHeader,
						map[string]string{"mount": "sha256:" + testBlobDigest},
					),
					http.StatusAccepted,
				},
			},
		},
		{
			name: "invalid mount",
			requests: []testRequestBuilder{
				{
					reqPost(
						"myrepo/myimage",
						testAuthHeader,
						map[string]string{"mount": "invalid mount"},
					),
					http.StatusBadRequest,
				},
			},
		},
		{
			name: "from without permissions",
			requests: []testRequestBuilder{
				{
					reqPost(
						"myrepo/myimage",
						testAuthHeader,
						map[string]string{"mount": "sha256:" + testBlobDigest},
					),
					http.StatusAccepted,
				},
				{
					reqPost(
						"public/from_myrepo_myimage",
						"",
						map[string]string{
							"mount": "sha256:" + testBlobDigest,
							"from":  "myrepo/myimage",
						},
					),
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "invalid from",
			requests: []testRequestBuilder{
				{
					reqPost(
						"myrepo/myimage",
						testAuthHeader,
						map[string]string{"from": "invalid from"},
					),
					http.StatusBadRequest,
				},
			},
		},
		{
			name: "unknown uuid",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						url := "/v2/myrepo/myimage/blobs/uploads/da551e8f-e411-4f93-bd29-481e481c6dbb"
						r := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(testBlob))
						q := r.URL.Query()
						q.Add("digest", "sha256:"+testBlobDigest)
						r.URL.RawQuery = q.Encode()
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusNotFound,
				},
			},
		},
		{
			name: "empty blob",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", testAuthHeader, map[string]string{}),
					http.StatusAccepted,
				},
				{
					func(prev *http.Response) *http.Request {
						emptyHash := "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s?digest=%s", uuid, emptyHash)
						r := httptest.NewRequest(http.MethodPut, url, nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusCreated,
				},
			},
		},
		{
			name: "successful GET",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", testAuthHeader, map[string]string{}),
					http.StatusAccepted,
				},
				{
					func(prevResp *http.Response) *http.Request {
						uuid := prevResp.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						r := httptest.NewRequest(http.MethodGet, url, nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusNoContent,
				},
			},
		},
		{
			name: "unsuccessful GET, without auth",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", testAuthHeader, map[string]string{}),
					http.StatusAccepted,
				},
				{
					func(prevResp *http.Response) *http.Request {
						uuid := prevResp.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						r := httptest.NewRequest(http.MethodGet, url, nil)
						return r
					},
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "successful PATCH by ranges",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", testAuthHeader, map[string]string{}),
					http.StatusAccepted,
				},
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), strings.NewReader("ho"))
						r.Header.Set("Authorization", testAuthHeader)
						r.Header.Set("Content-Type", "application/octet-stream")
						r.Header.Set("Content-Range", "0-2")
						r.Header.Set("Content-Length", "2")
						return r
					},
					http.StatusAccepted,
				},
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), strings.NewReader("la"))
						r.Header.Set("Authorization", testAuthHeader)
						r.Header.Set("Content-Type", "application/octet-stream")
						r.Header.Set("Content-Range", "2-4")
						r.Header.Set("Content-Length", "2")
						return r
					},
					http.StatusAccepted,
				},
			},
		},
		{
			name: "unsuccessful PATCH with invalid ranges",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", testAuthHeader, map[string]string{}),
					http.StatusAccepted,
				},
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), strings.NewReader("ho"))
						r.Header.Set("Authorization", testAuthHeader)
						r.Header.Set("Content-Type", "application/octet-stream")
						r.Header.Set("Content-Range", "1-3")
						r.Header.Set("Content-Length", "2")
						return r
					},
					http.StatusRequestedRangeNotSatisfiable,
				},
			},
		},
		{
			name: "unsuccessful PATCH for unknown blob",
			requests: []testRequestBuilder{
				{
					func(_ *http.Response) *http.Request {
						uuid := "8fb11ce6-e936-4cfc-aea9-f2d41c1a967c"
						r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), strings.NewReader("hola"))
						r.Header.Set("Authorization", testAuthHeader)
						r.Header.Set("Content-Type", "application/octet-stream")
						r.Header.Set("Content-Range", "0-4")
						r.Header.Set("Content-Length", "4")
						return r
					},
					http.StatusNotFound,
				},
			},
		},
		{
			name: "unsuccessful PATCH with invalid Content-Type",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", testAuthHeader, map[string]string{}),
					http.StatusAccepted,
				},
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						r := httptest.NewRequest(http.MethodPatch, url, bytes.NewReader(testBlob))
						r.Header.Set("Authorization", testAuthHeader)
						// Without Content-Type on proposal.
						r.Header.Set("Content-Range", fmt.Sprintf("0-%s", fmt.Sprint(len(testBlob))))
						return r
					},
					http.StatusBadRequest,
				},
			},
		},
		{
			name: "unsuccessful PATCH without auth",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", testAuthHeader, map[string]string{}),
					http.StatusAccepted,
				},
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						r := httptest.NewRequest(http.MethodPatch, url, bytes.NewReader(testBlob))
						r.Header.Set("Content-Type", "application/octet-stream")
						r.Header.Set("Content-Range", fmt.Sprintf("0-%s", fmt.Sprint(len(testBlob))))
						return r
					},
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "successful DELETE",
			requests: []testRequestBuilder{
				{
					reqPost("myrepo/myimage", testAuthHeader, map[string]string{}),
					http.StatusAccepted,
				},
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						r := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusNoContent,
				},
			},
		},
		{
			name: "unsuccessful DELETE with unknown UUID",
			requests: []testRequestBuilder{
				{
					func(prev *http.Response) *http.Request {
						uuid := "8fb11ce6-e936-4cfc-aea9-f2d41c1a967c"
						r := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
						r.Header.Set("Authorization", testAuthHeader)
						return r
					},
					http.StatusNotFound,
				},
			},
		},
		{
			name: "unsuccessful DELETE without auth",
			requests: []testRequestBuilder{
				{
					func(prev *http.Response) *http.Request {
						uuid := "8fb11ce6-e936-4cfc-aea9-f2d41c1a967c"
						r := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
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
