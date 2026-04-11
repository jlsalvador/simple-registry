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
	"encoding/base64"
	"net/http"
	"regexp"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/internal/http/handler"
	"github.com/jlsalvador/simple-registry/pkg/rbac"

	"golang.org/x/crypto/bcrypt"
)

const (
	testUser                      = "testuser"
	testPwd                       = "testpwd"
	testUserWithoutPerms          = "without"
	testPwdWithoutPerms           = "without"
	testHeaderDockerUploadUUID    = "Docker-Upload-UUID"
	testHeaderDockerContentDigest = "Docker-Content-Digest"
)

var (
	testAuthHeader             = testBuildBasicAuth(testUser, testPwd)
	testAuthHeaderWithoutPerms = testBuildBasicAuth(testUserWithoutPerms, testPwdWithoutPerms)
	testPwdHash                = func() string {
		h1, _ := bcrypt.GenerateFromPassword([]byte(testPwd), bcrypt.MinCost)
		return string(h1)
	}()
	testPwdWithoutPermsHash = func() string {
		h2, _ := bcrypt.GenerateFromPassword([]byte(testPwdWithoutPerms), bcrypt.MinCost)
		return string(h2)
	}()
)

func testBuildBasicAuth(user, pwd string) string {
	auth := user + ":" + pwd
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func testSetupTestServeMux(t *testing.T) http.Handler {
	t.Helper()

	cfg, err := config.New(
		config.WithAdminName(testUser),
		config.WithAdminPwd([]byte(testPwd)),
		config.WithDataDir(t.TempDir()),
	)
	if err != nil {
		t.Fatal(err)
	}

	cfg.Rbac.Users = append(cfg.Rbac.Users, rbac.User{
		Name:         testUser,
		PasswordHash: testPwdHash,
	}, rbac.User{
		Name:         testUserWithoutPerms,
		PasswordHash: testPwdWithoutPermsHash,
	}, rbac.User{
		Name: rbac.AnonymousUsername,
	})

	cfg.Rbac.Roles = append(cfg.Rbac.Roles,
		rbac.Role{
			Name:      "everything",
			Resources: []string{"*"},
			Verbs: []string{
				http.MethodHead, http.MethodGet, http.MethodPost,
				http.MethodPut, http.MethodPatch, http.MethodDelete,
			},
		},
		rbac.Role{
			Name:      "read_index",
			Resources: []string{""},
			Verbs:     []string{http.MethodHead, http.MethodGet},
		},
	)

	cfg.Rbac.RoleBindings = append(cfg.Rbac.RoleBindings,
		rbac.RoleBinding{
			Name:     "everyone_to_just_one_repo",
			Subjects: []rbac.Subject{{Kind: "User", Name: rbac.AnonymousUsername}},
			RoleName: "everything",
			Scopes:   []regexp.Regexp{*regexp.MustCompile("^public/.+$")},
		},
		rbac.RoleBinding{
			Name:     "anonymous_read_index",
			Subjects: []rbac.Subject{{Kind: "User", Name: rbac.AnonymousUsername}},
			RoleName: "read_index",
			Scopes:   []regexp.Regexp{*regexp.MustCompile("^$")},
		},
	)

	return handler.NewHandler(*cfg, true)
}

type testRequestBuilder struct {
	requestFn  func(prevResp *http.Response) *http.Request
	statusCode int
}
