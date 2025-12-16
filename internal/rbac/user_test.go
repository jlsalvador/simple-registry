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

	tests := []struct {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := user.IsPasswordValid(tt.plainPassword)
			if got != tt.want {
				t.Errorf("IsPasswordValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
