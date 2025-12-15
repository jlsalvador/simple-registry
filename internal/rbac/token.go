package rbac

import (
	"slices"
	"time"
)

type Token struct {
	Name      string
	Value     string
	ExpiresAt time.Time
	Username  string
}

func (e *Engine) CleanupExpiredTokens() {
	now := time.Now()
	e.Tokens = slices.DeleteFunc(e.Tokens, func(t Token) bool {
		return now.Compare(t.ExpiresAt) > 0
	})
}
