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
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/internal/http/handler"
	"github.com/jlsalvador/simple-registry/pkg/digest"
	"github.com/jlsalvador/simple-registry/pkg/rbac"

	"golang.org/x/crypto/bcrypt"
)

const testUser = "testuser"
const testPwd = "testpwd"
const testUserWithoutPerms = "without"
const testPwdWithoutPerms = "without"

var testAuthHeader = "Basic " + base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "%s:%s", testUser, testPwd))
var testAuthHeaderWithoutPerms = "Basic " + base64.StdEncoding.EncodeToString(fmt.Appendf(nil, "%s:%s", testUserWithoutPerms, testPwdWithoutPerms))

func setupTestServeMux(t *testing.T) http.Handler {
	t.Helper()

	cfg, err := config.New(testUser, testPwd, "", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	cfg.Rbac.Users = append(cfg.Rbac.Users, rbac.User{
		Name: testUserWithoutPerms,
		PasswordHash: func() string {
			pwd, err := bcrypt.GenerateFromPassword([]byte(testPwdWithoutPerms), bcrypt.DefaultCost)
			if err != nil {
				t.Fatal(err)
			}
			return string(pwd)
		}(),
	}, rbac.User{
		Name: rbac.AnonymousUsername,
	})

	cfg.Rbac.Roles = append(cfg.Rbac.Roles, rbac.Role{
		Name:      "everything",
		Resources: []string{"*"},
		Verbs: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
	})

	cfg.Rbac.RoleBindings = append(cfg.Rbac.RoleBindings, rbac.RoleBinding{
		Name:     "everyone_to_just_one_repo",
		Subjects: []rbac.Subject{{Kind: "User", Name: rbac.AnonymousUsername}},
		RoleName: "everything",
		Scopes:   []regexp.Regexp{*regexp.MustCompile("^public/.+$")},
	})

	return handler.NewHandler(*cfg)
}

func TestBlobsUploadsPost(t *testing.T) {
	data := []byte("hello world")
	dgst, err := digest.NewHasher("sha256")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dgst.Write(data); err != nil {
		t.Fatal(err)
	}
	hash := dgst.GetHashAsString()

	tests := []struct {
		name           string
		requests       []func(prevResp *http.Response) *http.Request
		expectedStatus []int
	}{
		{
			name: "successful POST then PUT initiation",
			requests: []func(prevResp *http.Response) *http.Request{
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					q := r.URL.Query()
					q.Add("digest", "sha256:"+hash)
					r.URL.RawQuery = q.Encode()
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Range", fmt.Sprintf("%d-%d", 0, len(data)))
					r.Header.Set("Content-Length", fmt.Sprint(len(data)))
					r.Body = io.NopCloser(bytes.NewReader(data))
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusCreated,
			},
		},
		{
			name: "unauthorized, no auth header",
			requests: []func(prevResp *http.Response) *http.Request{
				func(prevResp *http.Response) *http.Request {
					return httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
				},
			},
			expectedStatus: []int{
				http.StatusUnauthorized,
			},
		},
		{
			name: "forbidden, for PUT request",
			requests: []func(prevResp *http.Response) *http.Request{
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					q := r.URL.Query()
					q.Add("digest", "sha256:"+hash)
					r.URL.RawQuery = q.Encode()
					r.Header.Set("Authorization", testAuthHeaderWithoutPerms)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Length", fmt.Sprint(len(data)))
					r.Body = io.NopCloser(bytes.NewReader(data))
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusUnauthorized,
			},
		},
		{
			name: "forbidden, anonymous user for PUT request",
			requests: []func(prevResp *http.Response) *http.Request{
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					q := r.URL.Query()
					q.Add("digest", "sha256:"+hash)
					r.URL.RawQuery = q.Encode()
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Length", fmt.Sprint(len(data)))
					r.Body = io.NopCloser(bytes.NewReader(data))
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusUnauthorized,
			},
		},
		{
			name: "forbidden, authenticated but no permission",
			requests: []func(prevResp *http.Response) *http.Request{
				func(prevResp *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeaderWithoutPerms)
					return r
				},
			},
			expectedStatus: []int{
				http.StatusUnauthorized,
			},
		},
		{
			name: "invalid repository name",
			requests: []func(prevResp *http.Response) *http.Request{
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/invalid%20repo%20name%21/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
			},
			expectedStatus: []int{
				http.StatusNotFound,
			},
		},
		{
			name: "invalid digest",
			requests: []func(prevResp *http.Response) *http.Request{
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					q := r.URL.Query()
					q.Add("digest", "sha256:abc")
					r.URL.RawQuery = q.Encode()
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Length", fmt.Sprint(len(data)))
					r.Body = io.NopCloser(bytes.NewReader(data))
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusBadRequest,
			},
		},
		{
			name: "unknown digest",
			requests: []func(prevResp *http.Response) *http.Request{
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					invalidDgst := "f1234d75178d892a133a410355a5a990cf75d2f33eba25d575943d4df632f3a4" // "invalid"
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					q := r.URL.Query()
					q.Add("digest", "sha256:"+invalidDgst)
					r.URL.RawQuery = q.Encode()
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Length", fmt.Sprint(len(data)))
					r.Body = io.NopCloser(bytes.NewReader(data))
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusBadRequest,
			},
		},
		{
			name: "unknown uuid",
			requests: []func(prevResp *http.Response) *http.Request{
				func(prevResp *http.Response) *http.Request {
					uuid := "da551e8f-e411-4f93-bd29-481e481c6dbb"
					r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					q := r.URL.Query()
					q.Add("digest", "sha256:"+hash)
					r.URL.RawQuery = q.Encode()
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Length", fmt.Sprint(len(data)))
					r.Body = io.NopCloser(bytes.NewReader(data))
					return r
				},
			},
			expectedStatus: []int{
				http.StatusNotFound,
			},
		},
		{
			name: "empty blob",
			requests: []func(prevResp *http.Response) *http.Request{
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					data := []byte("")

					dgst, err := digest.NewHasher("sha256")
					if err != nil {
						t.Fatal(err)
					}
					if _, err := dgst.Write(data); err != nil {
						t.Fatal(err)
					}
					hash := dgst.GetHashAsString()

					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					q := r.URL.Query()
					q.Add("digest", "sha256:"+hash)
					r.URL.RawQuery = q.Encode()
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Length", fmt.Sprint(len(data)))
					r.Body = io.NopCloser(bytes.NewReader(data))
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusCreated,
			},
		},
		{
			name: "single POST request",
			requests: []func(prevResp *http.Response) *http.Request{
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads", nil)
					q := r.URL.Query()
					q.Add("digest", "sha256:"+hash)
					r.URL.RawQuery = q.Encode()
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Range", fmt.Sprintf("%d-%d", 0, len(data)))
					r.Header.Set("Content-Length", fmt.Sprint(len(data)))
					r.Body = io.NopCloser(bytes.NewReader(data))
					return r
				},
			},
			expectedStatus: []int{
				http.StatusCreated,
			},
		},
		{
			name: "POST with mount and empty from",
			requests: []func(prevResp *http.Response) *http.Request{
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					q := r.URL.Query()
					q.Add("digest", "sha256:"+hash)
					r.URL.RawQuery = q.Encode()
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Length", fmt.Sprint(len(data)))
					r.Body = io.NopCloser(bytes.NewReader(data))
					return r
				},
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/anotherrepo/anotherimage/blobs/uploads", nil)
					q := r.URL.Query()
					q.Add("mount", "sha256:"+hash)
					r.URL.RawQuery = q.Encode()
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusCreated,
				http.StatusCreated,
			},
		},
		{
			name: "POST with mount and unknown from",
			requests: []func(prevResp *http.Response) *http.Request{
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/anotherrepo/anotherimage/blobs/uploads", nil)
					q := r.URL.Query()
					q.Add("mount", "sha256:"+hash)
					r.URL.RawQuery = q.Encode()
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
			},
		},
		{
			name: "POST with mount and not allowed from",
			requests: []func(prevResp *http.Response) *http.Request{
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads", nil)
					q := r.URL.Query()
					q.Add("digest", "sha256:"+hash)
					r.URL.RawQuery = q.Encode()
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Length", fmt.Sprint(len(data)))
					r.Body = io.NopCloser(bytes.NewReader(data))
					return r
				},
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/public/anom/blobs/uploads", nil)
					q := r.URL.Query()
					q.Add("mount", "sha256:"+hash)
					q.Add("from", "myrepo/myimage")
					r.URL.RawQuery = q.Encode()
					return r
				},
			},
			expectedStatus: []int{
				http.StatusCreated,
				http.StatusUnauthorized,
			},
		},
		{
			name: "successful GET",
			requests: []func(prevResp *http.Response) *http.Request{
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusNoContent,
			},
		},
		{
			name: "unauthorized GET as anonymous",
			requests: []func(prevResp *http.Response) *http.Request{
				func(_ *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusUnauthorized,
			},
		},
		{
			name: "successful PATCH",
			requests: []func(prevResp *http.Response) *http.Request{
				func(prevResp *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Range", fmt.Sprintf("%d-%d", 0, len(data)))
					r.Header.Set("Content-Length", fmt.Sprint(len(data)))
					r.Body = io.NopCloser(bytes.NewReader(data))
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusAccepted,
			},
		},
		{
			name: "successful PATCH by ranges",
			requests: []func(prevResp *http.Response) *http.Request{
				func(prevResp *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Range", "0-2")
					r.Header.Set("Content-Length", "2")
					r.Body = io.NopCloser(bytes.NewReader([]byte("ho")))
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Range", "2-4")
					r.Header.Set("Content-Length", "2")
					r.Body = io.NopCloser(bytes.NewReader([]byte("la")))
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusAccepted,
				http.StatusAccepted,
			},
		},
		{
			name: "unsuccessful PATCH with invalid Content-Type",
			requests: []func(prevResp *http.Response) *http.Request{
				func(prevResp *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Range", fmt.Sprintf("%d-%d", 0, len(data)))
					r.Header.Set("Content-Length", fmt.Sprint(len(data)))
					r.Body = io.NopCloser(bytes.NewReader(data))
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusBadRequest,
			},
		},
		{
			name: "unsuccessful PATCH with unauthorized user",
			requests: []func(prevResp *http.Response) *http.Request{
				func(prevResp *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Range", fmt.Sprintf("%d-%d", 0, len(data)))
					r.Header.Set("Content-Length", fmt.Sprint(len(data)))
					r.Body = io.NopCloser(bytes.NewReader(data))
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusUnauthorized,
			},
		},
		{
			name: "unsuccessful PATCH with invalid UUID",
			requests: []func(prevResp *http.Response) *http.Request{
				func(prevResp *http.Response) *http.Request {
					uuid := "da551e8f-e411-4f93-bd29-481e481c6dbb"
					r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Range", fmt.Sprintf("%d-%d", 0, len(data)))
					r.Header.Set("Content-Length", fmt.Sprint(len(data)))
					r.Body = io.NopCloser(bytes.NewReader(data))
					return r
				},
			},
			expectedStatus: []int{
				http.StatusNotFound,
			},
		},
		{
			name: "unsuccessful PATCH with invalid Content-Range",
			requests: []func(prevResp *http.Response) *http.Request{
				func(prevResp *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Range", "0-1")
					r.Header.Set("Content-Length", "1")
					r.Body = io.NopCloser(bytes.NewReader([]byte{data[0]}))
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					r.Header.Set("Authorization", testAuthHeader)
					r.Header.Set("Content-Type", "application/octet-stream")
					r.Header.Set("Content-Range", "2-3")
					r.Header.Set("Content-Length", "1")
					r.Body = io.NopCloser(bytes.NewReader([]byte{data[2]}))
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusAccepted,
				http.StatusRequestedRangeNotSatisfiable,
			},
		},
		{
			name: "successful DELETE",
			requests: []func(prevResp *http.Response) *http.Request{
				func(prevResp *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusNoContent,
			},
		},
		{
			name: "unsuccessful DELETE by anonymous user",
			requests: []func(prevResp *http.Response) *http.Request{
				func(prevResp *http.Response) *http.Request {
					r := httptest.NewRequest(http.MethodPost, "/v2/myrepo/myimage/blobs/uploads/", nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
				func(prevResp *http.Response) *http.Request {
					uuid := prevResp.Header.Get("Docker-Upload-UUID")
					r := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					return r
				},
			},
			expectedStatus: []int{
				http.StatusAccepted,
				http.StatusUnauthorized,
			},
		},
		{
			name: "unsuccessful DELETE with unknown UUID",
			requests: []func(prevResp *http.Response) *http.Request{
				func(prevResp *http.Response) *http.Request {
					uuid := "da551e8f-e411-4f93-bd29-481e481c6dbb"
					r := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/v2/myrepo/myimage/blobs/uploads/%s", uuid), nil)
					r.Header.Set("Authorization", testAuthHeader)
					return r
				},
			},
			expectedStatus: []int{
				http.StatusNotFound,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := setupTestServeMux(t)

			var prev *http.Response
			for i, fnr := range tt.requests {
				w := httptest.NewRecorder()
				r := fnr(prev)
				mux.ServeHTTP(w, r)
				prev = w.Result()

				if prev != nil && prev.StatusCode != tt.expectedStatus[i] {
					d, _ := io.ReadAll(prev.Body)
					if len(d) > 0 {
						t.Log(string(d))
					}
					t.Errorf("%q: expected status %d, got %d", tt.name, tt.expectedStatus[i], prev.StatusCode)
				}
			}
		})
	}
}
