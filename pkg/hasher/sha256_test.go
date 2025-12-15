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

package hasher_test

import (
	"bytes"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/hasher"
)

func TestSha256(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"valid", []byte("Hello world"), "64ec88ca00b268e5ba1a35678a1b5316d212f4f366b2477232534a8aeca37f3c"},
		{"empty", []byte{}, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"binary", []byte{0x01, 0x02, 0x03}, "039058c6f2c0cb492c533b0a4d14ef77cc0f78abccced5287d84a1a2011cfb81"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hasher := hasher.NewSha256()
			size, err := hasher.Write(tc.data)
			if err != nil {
				t.Error(err)
			}
			if size != len(tc.data) {
				t.Errorf("write size mismatch: expected %d, got %d", len(tc.data), size)
			}
			got := hasher.GetHashAsString()
			if got != tc.expected {
				t.Errorf("hash mismatch: expected %s, got %s", tc.expected, got)
			}
		})
	}
}

func TestSha256_EmptyStruct(t *testing.T) {
	t.Run("without data", func(t *testing.T) {
		t.Parallel()

		hasher := hasher.Sha256{}
		got := hasher.GetHash()
		if !bytes.Equal(got, []byte{}) {
			t.Errorf("hash mismatch: expected empty slice, got %v", got)
		}
	})

	t.Run("write data", func(t *testing.T) {
		t.Parallel()

		data := []byte{0x01, 0x02, 0x03}
		want := []byte{0x03, 0x90, 0x58, 0xc6, 0xf2, 0xc0, 0xcb, 0x49, 0x2c, 0x53, 0x3b, 0x0a, 0x4d, 0x14, 0xef, 0x77, 0xcc, 0x0f, 0x78, 0xab, 0xcc, 0xce, 0xd5, 0x28, 0x7d, 0x84, 0xa1, 0xa2, 0x01, 0x1c, 0xfb, 0x81}

		hasher := hasher.Sha256{}
		size, err := hasher.Write(data)
		if err != nil {
			t.Error(err)
		}
		if size != 3 {
			t.Errorf("write size mismatch: expected %d, got %d", 3, size)
		}
		got := hasher.GetHash()
		if !bytes.Equal(got, want) {
			t.Errorf("hash mismatch: expected %v, got %v", want, got)
		}
	})
}
