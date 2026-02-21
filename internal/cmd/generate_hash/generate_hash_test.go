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

package generatehash_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	generatehash "github.com/jlsalvador/simple-registry/internal/cmd/generate_hash"

	"golang.org/x/crypto/bcrypt"
)

// pipeStdin replaces os.Stdin with a pipe carrying the given content and
// returns a cleanup function that restores the original os.Stdin.
func pipeStdin(t *testing.T, content string) func() {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	if _, err = io.WriteString(w, content); err != nil {
		t.Fatalf("writing to pipe: %v", err)
	}
	w.Close()

	orig := os.Stdin
	os.Stdin = r
	return func() {
		os.Stdin = orig
		r.Close()
	}
}

// captureStdout replaces os.Stdout with a pipe and returns a function that
// restores the original os.Stdout and returns everything written so far.
func captureStdout(t *testing.T) func() string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}

	orig := os.Stdout
	os.Stdout = w
	return func() string {
		w.Close()
		os.Stdout = orig

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, r); err != nil {
			t.Fatalf("reading captured stdout: %v", err)
		}
		r.Close()
		return buf.String()
	}
}

// forceIsTerminal overrides IsTerminal for the duration of the test.
func forceIsTerminal(t *testing.T, isTerminal bool) {
	t.Helper()
	orig := generatehash.IsTerminal
	generatehash.IsTerminal = func(io.Writer) bool { return isTerminal }
	t.Cleanup(func() { generatehash.IsTerminal = orig })
}

// forceReadPassword overrides ReadPassword for the duration of the test.
func forceReadPassword(t *testing.T, fn func(int) ([]byte, error)) {
	t.Helper()
	orig := generatehash.ReadPassword
	generatehash.ReadPassword = fn
	t.Cleanup(func() { generatehash.ReadPassword = orig })
}

func TestCmdFn_pipedInput(t *testing.T) {
	const password = "s3cr3t-p@ssw0rd"

	forceIsTerminal(t, false)
	defer pipeStdin(t, password)()
	flush := captureStdout(t)

	if err := generatehash.CmdFn(); err != nil {
		t.Fatalf("CmdFn returned unexpected error: %v", err)
	}

	hash := strings.TrimSpace(flush())
	if hash == "" {
		t.Fatal("expected a hash on stdout, got empty string")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Errorf("hash does not match original password: %v", err)
	}
}

func TestCmdFn_outputEndsWithNewline(t *testing.T) {
	forceIsTerminal(t, false)
	defer pipeStdin(t, "anypassword")()
	flush := captureStdout(t)

	if err := generatehash.CmdFn(); err != nil {
		t.Fatalf("CmdFn returned unexpected error: %v", err)
	}

	if output := flush(); !strings.HasSuffix(output, "\n") {
		t.Errorf("expected output to end with newline, got: %q", output)
	}
}

func TestCmdFn_emptyPassword(t *testing.T) {
	forceIsTerminal(t, false)
	defer pipeStdin(t, "")()
	flush := captureStdout(t)

	if err := generatehash.CmdFn(); err != nil {
		t.Fatalf("CmdFn returned unexpected error: %v", err)
	}

	hash := strings.TrimSpace(flush())
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte("")); err != nil {
		t.Errorf("hash does not match empty password: %v", err)
	}
}

func TestCmdFn_pipedReadError(t *testing.T) {
	forceIsTerminal(t, false)

	r, _, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	// Close both ends so io.ReadAll returns an error.
	r.Close()

	orig := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = orig }()

	flush := captureStdout(t)
	defer flush()

	if err := generatehash.CmdFn(); err == nil {
		t.Error("expected an error when stdin is a closed pipe, got nil")
	}
}

func TestCmdFn_terminalInput(t *testing.T) {
	const password = "terminal-password"

	forceIsTerminal(t, true)
	forceReadPassword(t, func(_ int) ([]byte, error) { return []byte(password), nil })
	flush := captureStdout(t)

	if err := generatehash.CmdFn(); err != nil {
		t.Fatalf("CmdFn returned unexpected error: %v", err)
	}

	hash := strings.TrimSpace(flush())
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Errorf("hash does not match original password: %v", err)
	}
}

func TestCmdFn_terminalReadError(t *testing.T) {
	forceIsTerminal(t, true)
	forceReadPassword(t, func(_ int) ([]byte, error) {
		return nil, errors.New("read error")
	})

	flush := captureStdout(t)
	defer flush()

	if err := generatehash.CmdFn(); err == nil {
		t.Error("expected an error when ReadPassword fails, got nil")
	}
}

// TestCmdFn_bcryptError triggers bcrypt.ErrPasswordTooLong by passing a
// password longer than 72 bytes.
func TestCmdFn_bcryptError(t *testing.T) {
	forceIsTerminal(t, false)
	defer pipeStdin(t, strings.Repeat("a", 73))()

	flush := captureStdout(t)
	defer flush()

	if err := generatehash.CmdFn(); err == nil {
		t.Error("expected bcrypt error for password >72 bytes, got nil")
	}
}

func TestCmdName(t *testing.T) {
	if generatehash.CmdName == "" {
		t.Error("CmdName should not be empty")
	}
}

func TestCmdHelp(t *testing.T) {
	if generatehash.CmdHelp == "" {
		t.Error("CmdHelp should not be empty")
	}
}
