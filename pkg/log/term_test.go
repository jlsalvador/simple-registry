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
	"strings"
	"testing"
)

func TestEnhanceJSONForTerminal(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		level    string
		contains []string
	}{
		{
			name:  "log.level error with bold message",
			json:  `{"message":"fatal error","status":500}`,
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
			json:  `{"message":"hello"}`,
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
