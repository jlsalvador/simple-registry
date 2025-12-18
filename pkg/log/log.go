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
//
// Note: DefaultStdout, DefaultStderr and DefaultPrettyPrint should be
// configured during initialization and not modified concurrently.
//
// Example:
//
//	log.Info(
//	    "msg", "server started",
//	    "port", 8080,
//	).Print()
package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	LevelDebug = "DEBUG" // Output msg to [os.Stdout].
	LevelInfo  = "INFO"  // Output msg to [os.Stdout].
	LevelWarn  = "WARN"  // Output msg to [os.Stderr].
	LevelError = "ERROR" // Output msg to [os.Stderr].
)

const (
	FieldTimestamp = "@timestamp"
	FieldLevel     = "log.level"
)

var (
	DefaultStderr      io.Writer = os.Stderr // Sets output for LevelWarn and LevelError.
	DefaultStdout      io.Writer = os.Stdout // Sets output for LevelDebug and LevelInfo.
	DefaultPrettyPrint           = false     // Indent output as multiline JSON.
)

// Entry represents a log entry.
//
// Please, use Debug(), Info(), Warn(), and Error().
type Entry struct {
	stderr      io.Writer
	stdout      io.Writer
	prettyPrint bool
	fields      map[string]any
}

func (e *Entry) jsonMarshal(pretty bool) string {
	var b []byte
	var err error

	if pretty {
		b, err = json.MarshalIndent(e.fields, "", "  ")
	} else {
		b, err = json.Marshal(e.fields)
	}

	if err != nil {
		return Error("err", err).JSON()
	}

	return string(b)
}

func (e *Entry) JSON() string       { return e.jsonMarshal(false) }
func (e *Entry) JSONIndent() string { return e.jsonMarshal(true) }

// With adds key-value pairs to the logger.
func (e *Entry) With(kv ...any) *Entry {
	if e.fields == nil {
		e.stderr = DefaultStderr
		e.stdout = DefaultStdout
		e.prettyPrint = DefaultPrettyPrint
		e.fields = map[string]any{
			FieldTimestamp: time.Now().Format(time.RFC3339),
			FieldLevel:     LevelInfo,
		}
	}

	for i := 0; i+1 < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			continue
		}
		e.fields[key] = kv[i+1]
	}

	return e
}

// Print outputs the logger to stdout or stderr based on the log level.
func (e *Entry) Print() {
	if e.fields == nil {
		e.With()
	}

	out := e.stdout
	if l := e.fields["log.level"]; l == LevelWarn || l == LevelError {
		out = e.stderr
	}

	fmt.Fprintf(out, "%s\n", e.jsonMarshal(e.prettyPrint))
}

func Debug(kv ...any) *Entry { return (&Entry{}).With("log.level", LevelDebug).With(kv...) }
func Info(kv ...any) *Entry  { return (&Entry{}).With("log.level", LevelInfo).With(kv...) }
func Warn(kv ...any) *Entry  { return (&Entry{}).With("log.level", LevelWarn).With(kv...) }
func Error(kv ...any) *Entry { return (&Entry{}).With("log.level", LevelError).With(kv...) }
