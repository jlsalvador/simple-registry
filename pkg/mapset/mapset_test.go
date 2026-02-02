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

package mapset_test

import (
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/mapset"
)

func TestNewMapSet(t *testing.T) {
	s := mapset.NewMapSet[int]()

	if s == nil {
		t.Error("should not return nil")
	}

	if len(s) != 0 {
		t.Errorf("should create empty set, got length %d", len(s))
	}
}

func TestAdd(t *testing.T) {
	s := mapset.NewMapSet[int]()

	// Test adding single element.
	s.Add(1)
	if !s.Contains(1) {
		t.Error("Add() failed to add element 1")
	}

	// Test adding multiple elements.
	s.Add(2, 3, 4)
	for _, val := range []int{2, 3, 4} {
		if !s.Contains(val) {
			t.Errorf("Add() failed to add element %d", val)
		}
	}

	// Test adding duplicate (should not increase size).
	s.Add(1)
	if len(s) != 4 {
		t.Errorf("expected length 4 after adding duplicate, got %d", len(s))
	}

	// Test method chaining.
	s2 := mapset.NewMapSet[string]().Add("a").Add("b", "c")
	if len(s2) != 3 {
		t.Errorf("method chaining failed, expected length 3, got %d", len(s2))
	}
}

func TestContains(t *testing.T) {
	s := mapset.NewMapSet[string]()
	s.Add("hello", "world")

	tests := []struct {
		name     string
		element  string
		expected bool
	}{
		{"existing element 1", "hello", true},
		{"existing element 2", "world", true},
		{"non-existing element", "foo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := s.Contains(tt.element); got != tt.expected {
				t.Errorf("Contains(%q) = %v, want %v", tt.element, got, tt.expected)
			}
		})
	}
}

func TestEqual(t *testing.T) {
	tests := []struct {
		name     string
		set1     mapset.MapSet[int]
		set2     mapset.MapSet[int]
		expected bool
	}{
		{
			name:     "empty sets",
			set1:     mapset.NewMapSet[int](),
			set2:     mapset.NewMapSet[int](),
			expected: true,
		},
		{
			name:     "equal sets",
			set1:     mapset.NewMapSet[int]().Add(1, 2, 3),
			set2:     mapset.NewMapSet[int]().Add(3, 2, 1),
			expected: true,
		},
		{
			name:     "different lengths",
			set1:     mapset.NewMapSet[int]().Add(1, 2),
			set2:     mapset.NewMapSet[int]().Add(1, 2, 3),
			expected: false,
		},
		{
			name:     "same length different elements",
			set1:     mapset.NewMapSet[int]().Add(1, 2, 3),
			set2:     mapset.NewMapSet[int]().Add(1, 2, 4),
			expected: false,
		},
		{
			name:     "one empty one not",
			set1:     mapset.NewMapSet[int](),
			set2:     mapset.NewMapSet[int]().Add(1),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.set1.Equal(tt.set2); got != tt.expected {
				t.Errorf("Equal() = %v, want: %v", got, tt.expected)
			}
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() mapset.MapSet[int]
		expected string
	}{
		{
			name:     "empty set",
			setup:    func() mapset.MapSet[int] { return mapset.NewMapSet[int]() },
			expected: "{}",
		},
		{
			name:     "single element",
			setup:    func() mapset.MapSet[int] { return mapset.NewMapSet[int]().Add(1) },
			expected: "{1}",
		},
		{
			name:     "multiple elements sorted",
			setup:    func() mapset.MapSet[int] { return mapset.NewMapSet[int]().Add(3, 1, 2) },
			expected: "{1, 2, 3}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.setup()
			if got := s.String(); got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestStringWithStrings(t *testing.T) {
	s := mapset.NewMapSet[string]().Add("zebra", "apple", "banana")
	expected := "{apple, banana, zebra}"

	if got := s.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}

func TestGenericTypes(t *testing.T) {
	intSet := mapset.NewMapSet[int]().Add(1, 2, 3)
	if !intSet.Contains(2) {
		t.Error("int set failed")
	}

	stringSet := mapset.NewMapSet[string]().Add("a", "b", "c")
	if !stringSet.Contains("b") {
		t.Error("string set failed")
	}

	floatSet := mapset.NewMapSet[float64]().Add(1.1, 2.2, 3.3)
	if !floatSet.Contains(2.2) {
		t.Error("float64 set failed")
	}
}
