package rbac_test

import (
	"testing"
	"time"

	"github.com/jlsalvador/simple-registry/internal/rbac"
)

func TestCleanupExpiredTokens(t *testing.T) {
	e := rbac.Engine{
		Tokens: []rbac.Token{
			{"valid", "123", time.Now().Add(time.Hour), "admin"},
			{"expired", "123", time.Now().Add(-1 * time.Hour), "admin"},
		},
	}

	e.CleanupExpiredTokens()

	if len(e.Tokens) != 1 || e.Tokens[0].Name != "valid" {
		t.Errorf("Expected one valid token, got %q", e.Tokens)
	}
}
