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

package common_test

import (
	"os"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/common"
)

func TestGetEnv(t *testing.T) {
	var got string
	var want string

	// Fallback
	os.Unsetenv("TESTING")
	got = common.GetEnv("TESTING", "empty")
	want = "empty"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Value
	os.Setenv("TESTING", "something")
	got = common.GetEnv("TESTING", "anotherthing")
	want = "something"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestGetBool(t *testing.T) {
	for _, tt := range []struct {
		name string
		val  string
		want bool
	}{
		{"true lower case", "true", true},
		{"true upper case", "TRUE", true},
		{"true mixed cases", "tRuE", true},
		{"true single char", "t", true},
		{"true with leading space", " t", true},
		{"true with trailing space", " t ", true},
		{"true with trailing space", "t ", true},
		{"true as 1", "1", true},
		{"false lower case", "false", false},
		{"false upper case", "FALSE", false},
		{"false mixed case", "fAlSe", false},
		{"false single case", "f", false},
		{"false with leading space", " f", false},
		{"false with trailing space", " f ", false},
		{"false with trailing space", "f ", false},
		{"false as 0", "0", false},
		{"empty", "f", false},
		{"non-boolean value", "abc", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := common.GetBool(tt.val); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
func TestGetInt64(t *testing.T) {
	for _, tt := range []struct {
		name string
		val  string
		want int64
	}{
		{"positive number", "42", 42},
		{"negative number", "-42", -42},
		{"zero", "0", 0},
		{"large number", "9223372036854775807", 9223372036854775807},
		{"small number", "-9223372036854775808", -9223372036854775808},
		{"with leading space", " 42", 42},
		{"with trailing space", "42 ", 42},
		{"with both spaces", " 42 ", 42},
		{"invalid number", "abc", 0},
		{"empty string", "", 0},
		{"float number", "42.5", 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := common.GetInt64(tt.val); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
