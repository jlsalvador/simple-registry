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

package uuid_test

import (
	"io"
	"regexp"
	"testing"

	u "github.com/jlsalvador/simple-registry/pkg/uuid"
)

func TestNew(t *testing.T) {
	uuid, err := u.New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if len(*uuid) != 16 {
		t.Errorf("UUID length = %d, want 16", len(*uuid))
	}

	// Check version (4) in bits 12-15 of octet 6
	version := ((*uuid)[6] >> 4) & 0x0f
	if version != 4 {
		t.Errorf("UUID version = %d, want 4", version)
	}

	// Check variant (RFC 4122) in bits 6-7 of octet 8
	variant := ((*uuid)[8] >> 6) & 0x03
	if variant != 2 {
		t.Errorf("UUID variant = %d, want 2 (RFC 4122)", variant)
	}
}

func TestNew_Uniqueness(t *testing.T) {
	uuid1, err := u.New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	uuid2, err := u.New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if uuid1.String() == uuid2.String() {
		t.Error("Two consecutive UUIDs should not be equal")
	}
}

func TestNew_Error(t *testing.T) {
	// Save original rand.Reader
	oldReader := u.RandRead
	defer func() { u.RandRead = oldReader }()

	// Replace with failing reader
	u.RandRead = failingReader

	_, err := u.New()
	if err == nil {
		t.Error("New() should return error when rand.Read fails")
	}
}

func TestUUID_String(t *testing.T) {
	uuid, err := u.New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	str := uuid.String()

	// Check format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	pattern := `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`
	matched, err := regexp.MatchString(pattern, str)
	if err != nil {
		t.Fatalf("regexp.MatchString() error: %v", err)
	}

	if !matched {
		t.Errorf("UUID string format = %q, want format matching %q", str, pattern)
	}

	// Verify version in string (should be '4' at position 14)
	if str[14] != '4' {
		t.Errorf("UUID version in string = %c, want '4'", str[14])
	}
}

func TestMustNew(t *testing.T) {
	uuid := u.MustNew()

	if len(*uuid) != 16 {
		t.Errorf("UUID length = %d, want 16", len(*uuid))
	}

	// Check version and variant
	version := ((*uuid)[6] >> 4) & 0x0f
	if version != 4 {
		t.Errorf("UUID version = %d, want 4", version)
	}

	variant := ((*uuid)[8] >> 6) & 0x03
	if variant != 2 {
		t.Errorf("UUID variant = %d, want 2", variant)
	}
}

func TestMustNew_Panic(t *testing.T) {
	// Save original rand.Reader
	oldReader := u.RandRead
	defer func() { u.RandRead = oldReader }()

	// Replace with failing reader
	u.RandRead = failingReader

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustNew() should panic when New() returns error")
		}
	}()

	u.MustNew()
}

// failingReader is a reader that always returns an error
func failingReader(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func BenchmarkNew(b *testing.B) {
	for b.Loop() {
		_, _ = u.New()
	}
}

func BenchmarkMustNew(b *testing.B) {
	for b.Loop() {
		_ = u.MustNew()
	}
}

func BenchmarkString(b *testing.B) {
	uuid := u.MustNew()

	for b.Loop() {
		_ = uuid.String()
	}
}
