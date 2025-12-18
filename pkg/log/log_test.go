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

package log_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/log"
)

// helper: parse JSON output
func parseJSON(t *testing.T, s string) map[string]any {
	t.Helper()

	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(s)), &m); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, s)
	}
	return m
}

func TestInfoPrintStdout(t *testing.T) {
	var buf bytes.Buffer
	log.DefaultStdout = &buf
	log.DefaultStderr = &buf
	log.DefaultPrettyPrint = false

	log.Info("msg", "hello").Print()

	m := parseJSON(t, buf.String())

	if m["msg"] != "hello" {
		t.Fatalf("expected msg=hello, got %v", m["msg"])
	}
	if m[log.FieldLevel] != log.LevelInfo {
		t.Fatalf("expected level INFO, got %v", m[log.FieldLevel])
	}
}

func TestWarnPrintStderr(t *testing.T) {
	var out, err bytes.Buffer
	log.DefaultStdout = &out
	log.DefaultStderr = &err

	log.Warn("warn", true).Print()

	if out.Len() != 0 {
		t.Fatalf("stdout should be empty")
	}
	if err.Len() == 0 {
		t.Fatalf("stderr should not be empty")
	}

	m := parseJSON(t, err.String())
	if m["warn"] != true {
		t.Fatalf("expected warn=true")
	}
}

func TestPrettyPrint(t *testing.T) {
	var buf bytes.Buffer
	log.DefaultStdout = &buf
	log.DefaultStderr = &buf
	log.DefaultPrettyPrint = true

	log.Info("a", 1).Print()

	if !strings.Contains(buf.String(), "\n") {
		t.Fatalf("expected pretty printed json")
	}
}

func TestJSONAndJSONIndent(t *testing.T) {
	e := log.Info("x", 1)

	j1 := e.JSON()
	j2 := e.JSONIndent()

	if !strings.HasPrefix(j1, "{") {
		t.Fatalf("invalid JSON()")
	}
	if !strings.Contains(j2, "\n") {
		t.Fatalf("expected multiline JSONIndent()")
	}
}

func TestWithIgnoresNonStringKey(t *testing.T) {
	var buf bytes.Buffer
	log.DefaultStdout = &buf
	log.DefaultStderr = &buf

	log.Info(123, "ignored", "ok", true).Print()

	m := parseJSON(t, buf.String())
	if _, exists := m["ignored"]; exists {
		t.Fatalf("non-string key should be ignored")
	}
	if m["ok"] != true {
		t.Fatalf("expected ok=true")
	}
}

func TestPrintWithoutWith(t *testing.T) {
	var buf bytes.Buffer
	log.DefaultStdout = &buf
	log.DefaultStderr = &buf

	var e log.Entry
	e.Print()

	m := parseJSON(t, buf.String())
	if m[log.FieldLevel] != log.LevelInfo {
		t.Fatalf("expected default level INFO")
	}
}

func TestJSONMarshalErrorPath(t *testing.T) {
	var buf bytes.Buffer
	log.DefaultStdout = &buf
	log.DefaultStderr = &buf

	ch := make(chan int) // not JSON-marshalable
	log.Info("bad", ch).Print()

	m := parseJSON(t, buf.String())
	if m[log.FieldLevel] != log.LevelError {
		t.Fatalf("expected LevelError on marshal failure")
	}
	if _, ok := m["err"]; !ok {
		t.Fatalf("expected err field")
	}
}

func TestAllLevels(t *testing.T) {
	var buf bytes.Buffer
	log.DefaultStdout = &buf
	log.DefaultStderr = &buf

	log.Debug().Print()
	log.Info().Print()
	log.Warn().Print()
	log.Error().Print()

	if buf.Len() == 0 {
		t.Fatalf("expected output")
	}
}
