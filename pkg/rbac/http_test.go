package rbac_test

import (
	"encoding/base64"
	"errors"
	netHttp "net/http"
	"testing"
	"time"

	"github.com/jlsalvador/simple-registry/pkg/http"
	"github.com/jlsalvador/simple-registry/pkg/rbac"

	"golang.org/x/crypto/bcrypt"
)

func TestGetUsernameFromHttpRequest(t *testing.T) {
	e := &rbac.Engine{
		Tokens: []rbac.Token{
			{"token_a", "123", time.Now().Add(time.Hour), "admin"},
			{"token_b", "456", time.Now().Add(-1 * time.Hour), "admin"},
		},
		Users: []rbac.User{
			{"admin", func() string {
				pwd, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
				return string(pwd)
			}(), nil},
		},
	}

	tcs := []struct {
		name    string
		request *netHttp.Request
		want    string
		wantErr error
	}{
		{
			name: "valid user",
			request: &netHttp.Request{
				Header: netHttp.Header{
					"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("admin:admin"))},
				},
			},
			want:    "admin",
			wantErr: nil,
		},
		{
			name: "valid basic auth with blank password",
			request: &netHttp.Request{
				Header: netHttp.Header{
					"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("blank_password:"))},
				},
			},
			want:    "",
			wantErr: nil,
		},
		{
			name: "invalid user",
			request: &netHttp.Request{
				Header: netHttp.Header{
					"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("user:password"))},
				},
			},
			want:    "",
			wantErr: nil,
		},
		{
			name: "valid token",
			request: &netHttp.Request{
				Header: netHttp.Header{
					"Authorization": {"Bearer 123"},
				},
			},
			want:    "admin",
			wantErr: nil,
		},
		{
			name: "expired token",
			request: &netHttp.Request{
				Header: netHttp.Header{
					"Authorization": {"Bearer 456"},
				},
			},
			want:    "",
			wantErr: nil,
		},
		{
			name: "unreferenced token",
			request: &netHttp.Request{
				Header: netHttp.Header{
					"Authorization": {"Bearer 789"},
				},
			},
			want:    "",
			wantErr: nil,
		},
		{
			name:    "without auth header",
			request: &netHttp.Request{},
			want:    "",
			wantErr: nil,
		},
		{
			name: "unsupported auth scheme",
			request: &netHttp.Request{
				Header: netHttp.Header{
					"Authorization": {"Digest 123"},
				},
			},
			want: "",
		},
		{
			name: "invalid basic auth value",
			request: &netHttp.Request{
				Header: netHttp.Header{
					"Authorization": {"Basic 123"},
				},
			},
			want:    "",
			wantErr: http.ErrBadRequest,
		},
		{
			name: "invalid basic auth without password",
			request: &netHttp.Request{
				Header: netHttp.Header{
					"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte("admin"))},
				},
			},
			want:    "",
			wantErr: http.ErrBadRequest,
		},
		{
			name:    "invalid request",
			request: nil,
			want:    "",
			wantErr: http.ErrBadRequest,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := e.GetUsernameFromHttpRequest(tc.request)
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Fatalf("got error = %v, want %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("e.GetUsernameFromHttpRequest() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGetUsernameFromHttpRequest_Anonymous(t *testing.T) {
	e := &rbac.Engine{
		Users: []rbac.User{
			{rbac.AnonymousUsername, "hash", nil},
		},
	}
	tcs := []struct {
		name    string
		request *netHttp.Request
		want    string
	}{
		{
			name:    "without auth header",
			request: &netHttp.Request{},
			want:    rbac.AnonymousUsername,
		},
		{
			name: "with empty auth header",
			request: &netHttp.Request{
				Header: map[string][]string{
					"Authorization": {"Basic " + base64.StdEncoding.EncodeToString([]byte(":"))},
				},
			},
			want: rbac.AnonymousUsername,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			username, err := e.GetUsernameFromHttpRequest(tc.request)
			if err != nil {
				t.Errorf("e.GetUsernameFromHttpRequest() error = %v", err)
			}
			if username != rbac.AnonymousUsername {
				t.Errorf("e.GetUsernameFromHttpRequest() username: %v, want: %v", username, rbac.AnonymousUsername)
			}
		})
	}
}
