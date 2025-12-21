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
	"regexp"
	"strings"
)

const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiFaint  = "\033[2m"
	ansiNormal = "\033[22m"
	ansiRed    = "\033[31m"
	ansiYellow = "\033[33m"
	ansiGrey   = "\033[90m"
)

var RegexKeys = regexp.MustCompile(`(\"[^\"]+\"\s*:)`)
var RegexBold = regexp.MustCompile(`(\"(?:m|msg|message|d|dbg|debug|i|inf|info|information|informational|e|err|error|w|warn|warning|http\.request\.method|url\.original|http\.response\.status_code)\"\s*:\s*)(\"(?:\\.|[^"\\])*\"|\d+)`)

// enhanceJSONForTerminal applies ANSI escape codes to make the log output
// more readable in a terminal.
func enhanceJSONForTerminal(jsonStr string, level string) string {
	var fullLineColorCode string

	switch level {
	case LevelError:
		fullLineColorCode = ansiRed
	case LevelWarn:
		fullLineColorCode = ansiYellow
	case LevelDebug:
		fullLineColorCode = ansiGrey
	}

	enhanced := jsonStr

	// Bold messages. This regexp requires unmodified keys, so it needs to be
	// run before faint keys.
	enhanced = RegexBold.ReplaceAllString(enhanced, "$1"+ansiBold+"$2"+ansiNormal)

	// Faint keys.
	enhanced = RegexKeys.ReplaceAllString(enhanced, ansiFaint+"$1"+ansiNormal)

	// Full line color by levels.
	if fullLineColorCode != "" {
		enhanced = fullLineColorCode + enhanced + ansiReset

		// Some terms will reset color after bold/faint, so we force it to
		// reapply the line color.
		enhanced = strings.ReplaceAll(enhanced, ansiNormal, ansiNormal+fullLineColorCode)
	}

	return enhanced
}
