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

package flag_test

import (
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/cli/flag"
)

// TestStringSlice_Set verifies that Set appends elements to the slice
// correctly.
//
// Covers: *s = append(*s, value) and return nil.
func TestStringSlice_Set(t *testing.T) {
	var s flag.StringSlice

	// Step 1: Add the first value.
	if err := s.Set("config/dir1"); err != nil {
		t.Errorf("Set returned an unexpected error: %v", err)
	}

	// Verification.
	if len(s) != 1 {
		t.Errorf("Expected length 1, got %d", len(s))
	}
	if s[0] != "config/dir1" {
		t.Errorf("Expected 'config/dir1', got '%s'", s[0])
	}

	// Step 2: Add a second value (test append).
	if err := s.Set("config/dir2"); err != nil {
		t.Errorf("Set returned an unexpected error: %v", err)
	}

	// Verification.
	if len(s) != 2 {
		t.Errorf("Expected length 2, got %d", len(s))
	}
	if s[1] != "config/dir2" {
		t.Errorf("Expected 'config/dir2', got '%s'", s[1])
	}
}

// TestStringSlice_String verifies the output format.
//
// Covers: return strings.Join(*s, ", ")
func TestStringSlice_String(t *testing.T) {
	tests := []struct {
		name     string
		slice    flag.StringSlice
		expected string
	}{
		{
			name:     "Empty slice (nil)",
			slice:    nil,
			expected: "",
		},
		{
			name:     "Empty slice (initialized)",
			slice:    flag.StringSlice{},
			expected: "",
		},
		{
			name:     "Single value",
			slice:    flag.StringSlice{"val1"},
			expected: "val1",
		},
		{
			name:     "Multiple values",
			slice:    flag.StringSlice{"val1", "val2", "val3"},
			expected: "val1, val2, val3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.slice.String()
			if got != tt.expected {
				t.Errorf("String() = %q, expected %q", got, tt.expected)
			}
		})
	}
}
