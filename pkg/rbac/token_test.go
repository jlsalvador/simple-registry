// Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
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

package rbac_test

import (
	"testing"
	"time"

	"github.com/jlsalvador/simple-registry/pkg/rbac"
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
