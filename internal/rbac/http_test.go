package rbac_test

import (
	"encoding/base64"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/jlsalvador/simple-registry/internal/rbac"
	"golang.org/x/crypto/bcrypt"
)

func TestGetUsernameFromHttpRequest(t *testing.T) {
	e := &rbac.Engine{
		Tokens: []rbac.Token{
			{"", "123", time.Now().Add(time.Hour), "admin"},
			{"", "456", time.Now().Add(-1 * time.Hour), "admin"},
		},
		Users: []rbac.User{
			{"admin", func() string {
				pwd, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
				return string(pwd)
			}(), nil},
		},
	}

	tests := []struct {
		name    string
		request *http.Request
		want    string
		wantErr error
	}{
		{
			name: "valid user",
			request: &http.Request{
				Header: http.Header{
					"Authorization": []string{"Basic " + base64.StdEncoding.EncodeToString([]byte("admin:admin"))},
				},
			},
			want:    "admin",
			wantErr: nil,
		},
		{
			name: "valid basic auth with blank password",
			request: &http.Request{
				Header: http.Header{
					"Authorization": []string{"Basic " + base64.StdEncoding.EncodeToString([]byte("blank_password:"))},
				},
			},
			want:    "",
			wantErr: nil,
		},
		{
			name: "invalid user",
			request: &http.Request{
				Header: http.Header{
					"Authorization": []string{"Basic " + base64.StdEncoding.EncodeToString([]byte("user:password"))},
				},
			},
			want:    "",
			wantErr: nil,
		},
		{
			name: "valid token",
			request: &http.Request{
				Header: http.Header{
					"Authorization": []string{"Bearer 123"},
				},
			},
			want:    "admin",
			wantErr: nil,
		},
		{
			name: "expired token",
			request: &http.Request{
				Header: http.Header{
					"Authorization": []string{"Bearer 456"},
				},
			},
			want:    "",
			wantErr: nil,
		},
		{
			name: "unreferenced token",
			request: &http.Request{
				Header: http.Header{
					"Authorization": []string{"Bearer 789"},
				},
			},
			want:    "",
			wantErr: nil,
		},
		{
			name:    "without auth header",
			request: &http.Request{},
			want:    "",
			wantErr: nil,
		},
		{
			name: "unsupported auth scheme",
			request: &http.Request{
				Header: http.Header{
					"Authorization": []string{"Digest 123"},
				},
			},
			want: "",
		},
		{
			name: "invalid basic auth value",
			request: &http.Request{
				Header: http.Header{
					"Authorization": []string{"Basic 123"},
				},
			},
			want:    "",
			wantErr: rbac.ErrBasicAuthInvalid,
		},
		{
			name: "invalid basic auth without password",
			request: &http.Request{
				Header: http.Header{
					"Authorization": []string{"Basic " + base64.StdEncoding.EncodeToString([]byte("admin"))},
				},
			},
			want:    "",
			wantErr: rbac.ErrBasicAuthInvalid,
		},
		{
			name:    "invalid request",
			request: nil,
			want:    "",
			wantErr: rbac.ErrHttpRequestInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := e.GetUsernameFromHttpRequest(tt.request)
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("got error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("e.GetUsernameFromHttpRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetUsernameFromHttpRequest_Anonymous(t *testing.T) {
	e := &rbac.Engine{
		Users: []rbac.User{
			{"", "", nil},
		},
	}
	r := &http.Request{}

	username, err := e.GetUsernameFromHttpRequest(r)
	if err != nil {
		t.Errorf("e.GetUsernameFromHttpRequest() error = %v", err)
	}
	if username != "" {
		t.Errorf("e.GetUsernameFromHttpRequest() username = %v, want empty string", username)
	}
}
