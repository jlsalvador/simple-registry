package rbac_test

import (
	"testing"

	"github.com/jlsalvador/simple-registry/internal/rbac"
	"golang.org/x/crypto/bcrypt"
)

func TestIsPasswordValid(t *testing.T) {
	var user = rbac.User{
		Name: "testuser",
		PasswordHash: func() string {
			pwd, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
			return string(pwd)
		}(),
	}

	tcs := []struct {
		name          string
		plainPassword string
		want          bool
	}{
		{
			name:          "valid password",
			plainPassword: "password123",
			want:          true,
		},
		{
			name:          "invalid password",
			plainPassword: "123456",
			want:          false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := user.IsPasswordValid(tc.plainPassword)
			if got != tc.want {
				t.Errorf("IsPasswordValid() = %v, want %v", got, tc.want)
			}
		})
	}
}
