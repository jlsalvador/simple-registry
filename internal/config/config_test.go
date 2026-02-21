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

package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jlsalvador/simple-registry/internal/config"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	pwdFile := filepath.Join(tmpDir, "pwd.txt")
	err := os.WriteFile(pwdFile, []byte("secret"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("with valid adminPwdFile", func(t *testing.T) {
		cfg, err := config.New("admin", "", pwdFile, tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Rbac.Users) == 0 || cfg.Rbac.Users[0].Name != "admin" {
			t.Errorf("expected admin user to be created")
		}
	})

	t.Run("with missing adminPwdFile", func(t *testing.T) {
		_, err := config.New("admin", "", filepath.Join(tmpDir, "missing.txt"), tmpDir)
		if err == nil {
			t.Fatal("expected error reading missing file")
		}
	})

	t.Run("with adminPwd string", func(t *testing.T) {
		cfg, err := config.New("admin", "secret", "", tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg == nil {
			t.Fatal("expected config, got nil")
		}
	})

	t.Run("with invalid adminPwd length for bcrypt", func(t *testing.T) {
		// bcrypt returns an error if the password exceeds 72 bytes.
		longPwd := strings.Repeat("a", 73)
		_, err := config.New("admin", longPwd, "", tmpDir)
		if err == nil {
			t.Fatal("expected error due to bcrypt byte limit")
		}
	})

	t.Run("with empty passwords (panic expected)", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected a panic when both passwords are empty")
			}
		}()
		config.New("admin", "", "", tmpDir)
	})
}
