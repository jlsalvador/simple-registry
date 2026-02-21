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

package version_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	versioncmd "github.com/jlsalvador/simple-registry/internal/cmd/version"
	"github.com/jlsalvador/simple-registry/internal/version"
)

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

func TestCmdFn(t *testing.T) {
	flush := captureStdout(t)

	if err := versioncmd.CmdFn(); err != nil {
		t.Fatalf("CmdFn returned unexpected error: %v", err)
	}

	output := flush()

	if !strings.Contains(output, version.AppName) {
		t.Errorf("expected output to contain AppName %q, got: %q", version.AppName, output)
	}
	if !strings.Contains(output, version.AppVersion) {
		t.Errorf("expected output to contain AppVersion %q, got: %q", version.AppVersion, output)
	}
}

func TestCmdName(t *testing.T) {
	if versioncmd.CmdName == "" {
		t.Error("CmdName should not be empty")
	}
}

func TestCmdHelp(t *testing.T) {
	if versioncmd.CmdHelp == "" {
		t.Error("CmdHelp should not be empty")
	}
}
