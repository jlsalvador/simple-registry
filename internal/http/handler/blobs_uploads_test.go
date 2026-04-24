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
	"maps"
	"net/http"
	"net/http/httptest"
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
	newReq := func(
		method string,
		url string,
		headers map[string]string,
		queries map[string]string,
		body []byte,
	) func(prev *http.Response) *http.Request {
		return func(_ *http.Response) *http.Request {
			var bodyReader io.Reader
			if body != nil {
				bodyReader = bytes.NewReader(body)
			}
			r := httptest.NewRequest(method, url, bodyReader)

			if body != nil {
				if headers == nil {
					headers = map[string]string{}
				} else {
					// Make a copy to avoid mutating the original map.
					headersCopy := map[string]string{}
					maps.Copy(headersCopy, headers)
					headers = headersCopy
				}

				if _, ok := headers["Content-Type"]; !ok {
					headers["Content-Type"] = "application/octet-stream"
				}
				if _, ok := headers["Content-Length"]; !ok {
					headers["Content-Length"] = fmt.Sprint(len(body))
				}
				if _, ok := headers["Content-Range"]; !ok {
					headers["Content-Range"] = fmt.Sprintf("0-%d", len(body)-1)
				}
			}

			for k, v := range headers {
				if v == "" {
					continue
				}
				r.Header.Set(k, v)
			}

			q := r.URL.Query()
			for k, v := range queries {
				q.Add(k, v)
			}
			r.URL.RawQuery = q.Encode()

			return r
		}
	}

	successUUID := testRequestBuilder{
		newReq(
			http.MethodPost,
			"/v2/myrepo/myimage/blobs/uploads/",
			map[string]string{"Authorization": testAuthHeader},
			nil,
			nil,
		),
		http.StatusAccepted,
	}

	tests := []struct {
		name     string
		requests []testRequestBuilder
	}{
		{
			name: "successful single POST",
			requests: []testRequestBuilder{
				{
					newReq(
						http.MethodPost,
						"/v2/myrepo/myimage/blobs/uploads/",
						map[string]string{"Authorization": testAuthHeader},
						map[string]string{"digest": "sha256:" + testBlobDigest},
						testBlob,
					),
					http.StatusCreated,
				},
			},
		},
		{
			name: "unauthorized POST without auth",
			requests: []testRequestBuilder{
				{
					newReq(
						http.MethodPost,
						"/v2/myrepo/myimage/blobs/uploads/",
						nil,
						nil,
						nil,
					),
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "successful single anonymous POST to public repository",
			requests: []testRequestBuilder{
				{
					newReq(
						http.MethodPost,
						"/v2/public/myimage/blobs/uploads",
						nil,
						map[string]string{"digest": "sha256:" + testBlobDigest},
						testBlob,
					),
					http.StatusCreated,
				},
			},
		},
		{
			name: "unsuccessful single anonymous POST to private repository",
			requests: []testRequestBuilder{
				{
					newReq(
						http.MethodPost,
						"/v2/private/myimage/blobs/uploads",
						nil,
						map[string]string{"digest": "sha256:" + testBlobDigest},
						testBlob,
					),
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "unsuccessful POST with invalid digest",
			requests: []testRequestBuilder{
				{
					newReq(
						http.MethodPost,
						"/v2/myrepo/myimage/blobs/uploads/",
						map[string]string{"Authorization": testAuthHeader},
						map[string]string{"digest": "invalid"},
						testBlob,
					),
					http.StatusBadRequest,
				},
			},
		},
		{
			name: "unsuccessful POST with unknown digest",
			requests: []testRequestBuilder{
				{
					newReq(
						http.MethodPost,
						"/v2/myrepo/myimage/blobs/uploads/",
						map[string]string{"Authorization": testAuthHeader},
						map[string]string{"digest": "sha256:abc"},
						testBlob,
					),
					http.StatusBadRequest,
				},
			},
		},
		{
			name: "unsuccessful POST with invalid repository name",
			requests: []testRequestBuilder{
				{
					newReq(
						http.MethodPost,
						"/v2/invalid%20repo%20name%21/blobs/uploads/",
						map[string]string{"Authorization": testAuthHeader},
						nil,
						nil,
					),
					http.StatusNotFound,
				},
			},
		},
		{
			name: "successful POST then PUT",
			requests: []testRequestBuilder{
				successUUID,
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						return newReq(
							http.MethodPut,
							url,
							map[string]string{"Authorization": testAuthHeader},
							map[string]string{"digest": "sha256:" + testBlobDigest},
							testBlob,
						)(prev)
					},
					http.StatusCreated,
				},
			},
		},
		{
			name: "proper POST but unauthorized PUT",
			requests: []testRequestBuilder{
				successUUID,
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						return newReq(
							http.MethodPut,
							url,
							map[string]string{"Authorization": testAuthHeaderWithoutPerms},
							map[string]string{"digest": "sha256:" + testBlobDigest},
							testBlob,
						)(prev)
					},
					http.StatusForbidden,
				},
			},
		},
		{
			name: "proper POST but PUT with invalid digest syntax",
			requests: []testRequestBuilder{
				successUUID,
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						return newReq(
							http.MethodPut,
							url,
							map[string]string{"Authorization": testAuthHeader},
							map[string]string{"digest": "invalid"},
							testBlob,
						)(prev)
					},
					http.StatusBadRequest,
				},
			},
		},
		{
			name: "proper POST but PUT with mismatched digest",
			requests: []testRequestBuilder{
				successUUID,
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						return newReq(
							http.MethodPut,
							url,
							map[string]string{"Authorization": testAuthHeader},
							map[string]string{"digest": "sha256:abc"},
							testBlob,
						)(prev)
					},
					http.StatusBadRequest,
				},
			},
		},
		{
			name: "proper POST but PUT with unknown digest",
			requests: []testRequestBuilder{
				successUUID,
				{
					func(prev *http.Response) *http.Request {
						digestUnknown := "sha256:f1234d75178d892a133a410355a5a990cf75d2f33eba25d575943d4df632f3a4"
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						return newReq(
							http.MethodPut,
							url,
							map[string]string{"Authorization": testAuthHeader},
							map[string]string{"digest": digestUnknown},
							testBlob,
						)(prev)
					},
					http.StatusBadRequest,
				},
			},
		},
		{
			name: "successful mount",
			requests: []testRequestBuilder{
				successUUID,
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						return newReq(
							http.MethodPut,
							url,
							map[string]string{"Authorization": testAuthHeader},
							map[string]string{"digest": "sha256:" + testBlobDigest},
							testBlob,
						)(prev)
					},
					http.StatusCreated,
				},
				{
					newReq(
						http.MethodPost,
						"/v2/anotherrepo/otherimage/blobs/uploads/",
						map[string]string{"Authorization": testAuthHeader},
						map[string]string{
							"mount": "sha256:" + testBlobDigest,
							"from":  "myrepo/myimage",
						},
						nil,
					),
					http.StatusCreated,
				},
			},
		},
		{
			name: "mount not found",
			requests: []testRequestBuilder{
				{
					newReq(
						http.MethodPost,
						"/v2/myrepo/myimage/blobs/uploads/",
						map[string]string{"Authorization": testAuthHeader},
						map[string]string{"mount": "sha256:" + testBlobDigest},
						nil,
					),
					http.StatusAccepted,
				},
			},
		},
		{
			name: "invalid mount",
			requests: []testRequestBuilder{
				{
					newReq(
						http.MethodPost,
						"/v2/myrepo/myimage/blobs/uploads/",
						map[string]string{"Authorization": testAuthHeader},
						map[string]string{"mount": "invalid mount"},
						nil,
					),
					http.StatusBadRequest,
				},
			},
		},
		{
			name: "from without permissions",
			requests: []testRequestBuilder{
				{
					newReq(
						http.MethodPost,
						"/v2/myrepo/myimage/blobs/uploads/",
						map[string]string{"Authorization": testAuthHeader},
						map[string]string{"mount": "sha256:" + testBlobDigest},
						nil,
					),
					http.StatusAccepted,
				},
				{
					newReq(
						http.MethodPost,
						"/v2/public/from_myrepo_myimage/blobs/uploads/",
						nil,
						map[string]string{
							"mount": "sha256:" + testBlobDigest,
							"from":  "myrepo/myimage",
						},
						nil,
					),
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "invalid from",
			requests: []testRequestBuilder{
				{
					newReq(
						http.MethodPost,
						"/v2/myrepo/myimage/blobs/uploads/",
						map[string]string{"Authorization": testAuthHeader},
						map[string]string{"from": "invalid from"},
						nil,
					),
					http.StatusBadRequest,
				},
			},
		},
		{
			name: "unknown uuid",
			requests: []testRequestBuilder{
				{
					newReq(
						http.MethodPut,
						"/v2/myrepo/myimage/blobs/uploads/da551e8f-e411-4f93-bd29-481e481c6dbb",
						map[string]string{"Authorization": testAuthHeader},
						map[string]string{"digest": "sha256:" + testBlobDigest},
						testBlob,
					),
					http.StatusNotFound,
				},
			},
		},
		{
			name: "empty blob",
			requests: []testRequestBuilder{
				successUUID,
				{
					func(prev *http.Response) *http.Request {
						digestEmpty := "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						return newReq(
							http.MethodPut,
							url,
							map[string]string{"Authorization": testAuthHeader},
							map[string]string{"digest": digestEmpty},
							nil,
						)(prev)
					},
					http.StatusCreated,
				},
			},
		},
		{
			name: "successful GET",
			requests: []testRequestBuilder{
				successUUID,
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						return newReq(
							http.MethodGet,
							url,
							map[string]string{"Authorization": testAuthHeader},
							nil,
							nil,
						)(prev)
					},
					http.StatusNoContent,
				},
			},
		},
		{
			name: "unsuccessful GET, without auth",
			requests: []testRequestBuilder{
				successUUID,
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						return newReq(
							http.MethodGet,
							url,
							nil,
							nil,
							nil,
						)(prev)
					},
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "successful PATCH by ranges",
			requests: []testRequestBuilder{
				successUUID,
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						halfBlob := testBlob[:len(testBlob)/2]
						return newReq(
							http.MethodPatch,
							url,
							map[string]string{"Authorization": testAuthHeader},
							nil,
							halfBlob,
						)(prev)
					},
					http.StatusAccepted,
				},
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						restBlob := testBlob[len(testBlob)/2:]
						return newReq(
							http.MethodPatch,
							url,
							map[string]string{
								"Authorization": testAuthHeader,
								"Content-Range": fmt.Sprintf("%d-%d", len(testBlob)/2, len(testBlob)-1),
							},
							nil,
							restBlob,
						)(prev)
					},
					http.StatusAccepted,
				},
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						return newReq(
							http.MethodPut,
							url,
							map[string]string{"Authorization": testAuthHeader},
							map[string]string{"digest": "sha256:" + testBlobDigest},
							nil,
						)(prev)
					},
					http.StatusCreated,
				},
			},
		},
		{
			name: "unsuccessful PATCH with invalid ranges",
			requests: []testRequestBuilder{
				successUUID,
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						return newReq(
							http.MethodPatch,
							url,
							map[string]string{
								"Authorization": testAuthHeader,
								"Content-Range": fmt.Sprintf("1-%d", len(testBlob)-1),
							},
							nil,
							testBlob[1:],
						)(prev)
					},
					http.StatusRequestedRangeNotSatisfiable,
				},
			},
		},
		{
			name: "unsuccessful PATCH for unknown blob",
			requests: []testRequestBuilder{
				{
					func(prev *http.Response) *http.Request {
						uuidUnknown := "8fb11ce6-e936-4cfc-aea9-f2d41c1a967c"
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuidUnknown)
						return newReq(
							http.MethodPatch,
							url,
							map[string]string{"Authorization": testAuthHeader},
							nil,
							testBlob,
						)(prev)
					},
					http.StatusNotFound,
				},
			},
		},
		{
			name: "unsuccessful PATCH with invalid Content-Type",
			requests: []testRequestBuilder{
				successUUID,
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						return newReq(
							http.MethodPatch,
							url,
							map[string]string{
								"Authorization": testAuthHeader,
								"Content-Type":  "invalid/content-type",
							},
							nil,
							testBlob,
						)(prev)
					},
					http.StatusBadRequest,
				},
			},
		},
		{
			name: "unsuccessful PATCH without auth",
			requests: []testRequestBuilder{
				successUUID,
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						return newReq(
							http.MethodPatch,
							url,
							nil,
							nil,
							testBlob,
						)(prev)
					},
					http.StatusUnauthorized,
				},
			},
		},
		{
			name: "successful DELETE",
			requests: []testRequestBuilder{
				successUUID,
				{
					func(prev *http.Response) *http.Request {
						uuid := prev.Header.Get(testHeaderDockerUploadUUID)
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid)
						return newReq(
							http.MethodDelete,
							url,
							map[string]string{"Authorization": testAuthHeader},
							nil,
							nil,
						)(prev)
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
						uuidUnknown := "8fb11ce6-e936-4cfc-aea9-f2d41c1a967c"
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuidUnknown)
						return newReq(
							http.MethodDelete,
							url,
							map[string]string{"Authorization": testAuthHeader},
							nil,
							nil,
						)(prev)
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
						uuidUnknown := "8fb11ce6-e936-4cfc-aea9-f2d41c1a967c"
						url := fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuidUnknown)
						return newReq(
							http.MethodDelete,
							url,
							nil,
							nil,
							nil,
						)(prev)
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
