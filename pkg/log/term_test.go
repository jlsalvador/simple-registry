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

package log

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestIsTerminal(t *testing.T) {
	// Cleanup cache before tests.
	isTerminalCacheMu.Lock()
	isTerminalCache = map[uintptr]bool{}
	isTerminalCacheMu.Unlock()

	t.Run("not a file", func(t *testing.T) {
		buf := new(bytes.Buffer)
		if IsTerminal(buf) {
			t.Error("expected false for bytes.Buffer")
		}
	})

	t.Run("regular file is not tty", func(t *testing.T) {
		tmp, err := os.CreateTemp("", "test_log")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmp.Name())
		defer tmp.Close()

		if IsTerminal(tmp) {
			t.Error("expected false for regular file")
		}

		// Check cache.
		if IsTerminal(tmp) {
			t.Error("expected false (cached) for regular file")
		}
	})

	t.Run("pipe is not tty", func(t *testing.T) {
		r, w, _ := os.Pipe()
		defer r.Close()
		defer w.Close()

		if IsTerminal(w) {
			t.Error("expected false for pipe")
		}
	})
}

func TestIsTerminalForcedCache(t *testing.T) {
	tmp, _ := os.CreateTemp("", "forced_test")
	defer os.Remove(tmp.Name())
	fd := tmp.Fd()

	// Manually inject into the cache.
	isTerminalCacheMu.Lock()
	isTerminalCache[fd] = true
	isTerminalCacheMu.Unlock()

	if !IsTerminal(tmp) {
		t.Error("expected true because it was manually injected in cache")
	}
}

func TestEnhanceJSONForTerminal(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		level    string
		contains []string
	}{
		{
			name:  "log.level error with bold msg",
			json:  `{"msg":"fatal error","status":500}`,
			level: LevelError,
			contains: []string{
				ansiRed,
				ansiBold,
				ansiFaint,
				ansiReset,
			},
		},
		{
			name:  "log.level warn with status code",
			json:  `{"http.response.status_code":404}`,
			level: LevelWarn,
			contains: []string{
				ansiYellow,
				"404",
				ansiBold,
			},
		},
		{
			name:  "log.level debug with grey",
			json:  `{"d":"debugging"}`,
			level: LevelDebug,
			contains: []string{
				ansiGrey,
				"debugging",
			},
		},
		{
			name:  "log.level info no full line color",
			json:  `{"msg":"hello"}`,
			level: LevelInfo,
			contains: []string{
				ansiBold,
				ansiFaint,
			},
		},
		{
			name:  "url.original bold",
			json:  `{"url.original":"/test"}`,
			level: LevelInfo,
			contains: []string{
				"url.original",
				ansiBold,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := enhanceJSONForTerminal(tt.json, tt.level)
			for _, c := range tt.contains {
				if !strings.Contains(got, c) {
					t.Errorf("expected output to contain %q, got: %q", c, got)
				}
			}
		})
	}
}
