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

// Package log provides a key-value logger and JSON formatter.
package log

import (
	"encoding/json"
	"time"
)

const LEVEL_DEBUG = "DEBUG"
const LEVEL_INFO = "INFO"
const LEVEL_WARN = "WARN"
const LEVEL_ERROR = "ERROR"

// Entry represents a log entry.
//
// Please, use [Debug], [Info], [Warn], and [Error] functions.
type Entry struct {
	fields map[string]any
}

// JSON returns the logger as a JSON string.
func (l *Entry) JSON() string {
	b, _ := json.Marshal(l.fields)
	return string(b)
}

func (l *Entry) JSONIndent() string {
	b, _ := json.MarshalIndent(l.fields, "", "  ")
	return string(b)
}

// With adds key-value pairs to the logger.
func (l *Entry) With(kv ...any) *Entry {
	if l.fields == nil {
		l.fields = map[string]any{
			"@timestamp": time.Now().Format(time.RFC3339),
			"log.level":  LEVEL_INFO,
		}
	}

	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			continue
		}
		l.fields[key] = kv[i+1]
	}

	return l
}

func Debug(kv ...any) *Entry { return (&Entry{}).With("log.level", LEVEL_DEBUG).With(kv...) }
func Info(kv ...any) *Entry  { return (&Entry{}).With("log.level", LEVEL_INFO).With(kv...) }
func Warn(kv ...any) *Entry  { return (&Entry{}).With("log.level", LEVEL_WARN).With(kv...) }
func Error(kv ...any) *Entry { return (&Entry{}).With("log.level", LEVEL_ERROR).With(kv...) }
