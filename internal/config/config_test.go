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

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	pwdFile := filepath.Join(tmpDir, "pwd.txt")
	err := os.WriteFile(pwdFile, []byte("secret"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("with valid adminPwdFile", func(t *testing.T) {
		cfg, err := New(
			WithAdminName("admin"),
			WithAdminPwdFile(pwdFile),
			WithDataDir(tmpDir),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.Rbac.Users) == 0 || cfg.Rbac.Users[0].Name != "admin" {
			t.Errorf("expected admin user to be created")
		}
	})

	t.Run("with missing adminPwdFile", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected a panic when reading missing file")
			}
		}()

		New(
			WithAdminName("admin"),
			WithAdminPwdFile(filepath.Join(tmpDir, "missing.txt")),
			WithDataDir(tmpDir),
		)
	})

	t.Run("with adminPwd string", func(t *testing.T) {
		cfg, err := New(
			WithAdminName("admin"),
			WithAdminPwd([]byte("secret")),
			WithDataDir(tmpDir),
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg == nil {
			t.Fatal("expected config, got nil")
		}
	})
}

func TestNewWithInvalidAdminPwd(t *testing.T) {
	tmpDir := t.TempDir()

	// bcrypt returns an error if the password exceeds 72 bytes.
	longPwd := strings.Repeat("a", 73)
	_, err := New(
		WithAdminName("admin"),
		WithAdminPwd([]byte(longPwd)),
		WithDataDir(tmpDir),
	)
	if err == nil {
		t.Fatal("expected error due to bcrypt byte limit")
	}
}

func TestNewWithHttpTokenSecretFile(t *testing.T) {
	tmpDir := t.TempDir()
	tokenFile := filepath.Join(tmpDir, "token.txt")
	err := os.WriteFile(tokenFile, []byte("secret-token"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := New(
		WithAdminName("admin"),
		WithAdminPwd([]byte("pwd")),
		WithDataDir(tmpDir),
		WithHttpTokenSecretFile(tokenFile),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
}

func TestNewWithMissingHttpTokenSecretFile(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected a panic when token secret file is missing")
		}
	}()

	tmpDir := t.TempDir()

	New(
		WithAdminName("admin"),
		WithAdminPwd([]byte("pwd")),
		WithDataDir(tmpDir),
		WithHttpTokenSecretFile(filepath.Join(tmpDir, "missing-token.txt")),
	)
}

func TestNewWithHttpCertFile(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	err := os.WriteFile(certFile, []byte("cert-content"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := New(
		WithAdminName("admin"),
		WithAdminPwd([]byte("pwd")),
		WithDataDir(tmpDir),
		WithHttpCertFile(certFile),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Web.CertFile != certFile {
		t.Errorf("expected cert file %s, got %s", certFile, cfg.Web.CertFile)
	}
}

func TestNewWithHttpKeyFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "key.pem")
	err := os.WriteFile(keyFile, []byte("key-content"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := New(
		WithAdminName("admin"),
		WithAdminPwd([]byte("pwd")),
		WithDataDir(tmpDir),
		WithHttpKeyFile(keyFile),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
	if cfg.Web.KeyFile != keyFile {
		t.Errorf("expected key file %s, got %s", keyFile, cfg.Web.KeyFile)
	}
}

func TestNewWithExplicitHttpSettings(t *testing.T) {
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")
	if err := os.WriteFile(certFile, []byte("cert-content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyFile, []byte("key-content"), 0o644); err != nil {
		t.Fatal(err)
	}
	tokenSecret := []byte("super-secret-token")
	addr := "127.0.0.1:4321"

	cfg, err := New(
		WithAdminName("admin"),
		WithAdminPwd([]byte("pwd")),
		WithDataDir(tmpDir),
		WithHttpAddr(addr),
		WithHttpTokenSecret(tokenSecret),
		WithHttpTokenTimeout(15*time.Second),
		WithHttpUI(false),
		WithHttpCertFile(certFile),
		WithHttpKeyFile(keyFile),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Web.Addr != addr {
		t.Fatalf("expected addr %s, got %s", addr, cfg.Web.Addr)
	}
	if string(cfg.Web.TokenSecret) != string(tokenSecret) {
		t.Fatalf("expected token secret %q, got %q", tokenSecret, cfg.Web.TokenSecret)
	}
	if cfg.Web.TokenTimeout != 15*time.Second {
		t.Fatalf("expected token timeout 15s, got %s", cfg.Web.TokenTimeout)
	}
	if cfg.Web.UI != false {
		t.Fatalf("expected ui false, got %v", cfg.Web.UI)
	}
	if cfg.Web.CertFile != certFile {
		t.Fatalf("expected cert file %s, got %s", certFile, cfg.Web.CertFile)
	}
	if cfg.Web.KeyFile != keyFile {
		t.Fatalf("expected key file %s, got %s", keyFile, cfg.Web.KeyFile)
	}
}

func TestNewPanicsWithoutDataDir(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic without data dir")
		}
	}()

	New(
		WithAdminName("admin"),
		WithAdminPwd([]byte("pwd")),
	)
}

func TestNewPanicsWithoutAdminPwd(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic without admin password")
		}
	}()

	tmpDir := t.TempDir()

	New(WithDataDir(tmpDir))
}
