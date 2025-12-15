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

func TestSha512(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"valid", []byte("Hello world"), "b7f783baed8297f0db917462184ff4f08e69c2d5e5f79a942600f9725f58ce1f29c18139bf80b06c0fff2bdd34738452ecf40c488c22a7e3d80cdf6f9c1c0d47"},
		{"empty", []byte{}, "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e"},
		{"binary", []byte{0x01, 0x02, 0x03}, "27864cc5219a951a7a6e52b8c8dddf6981d098da1658d96258c870b2c88dfbcb51841aea172a28bafa6a79731165584677066045c959ed0f9929688d04defc29"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			hasher := hasher.NewSha512()
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

func TestSha512_EmptyStruct(t *testing.T) {
	t.Run("without data", func(t *testing.T) {
		t.Parallel()

		hasher := hasher.Sha512{}
		got := hasher.GetHash()
		if !bytes.Equal(got, []byte{}) {
			t.Errorf("hash mismatch: expected empty slice, got %v", got)
		}
	})

	t.Run("write data", func(t *testing.T) {
		t.Parallel()

		data := []byte{0x01, 0x02, 0x03}
		want := []byte{0x27, 0x86, 0x4c, 0xc5, 0x21, 0x9a, 0x95, 0x1a, 0x7a, 0x6e, 0x52, 0xb8, 0xc8, 0xdd, 0xdf, 0x69, 0x81, 0xd0, 0x98, 0xda, 0x16, 0x58, 0xd9, 0x62, 0x58, 0xc8, 0x70, 0xb2, 0xc8, 0x8d, 0xfb, 0xcb, 0x51, 0x84, 0x1a, 0xea, 0x17, 0x2a, 0x28, 0xba, 0xfa, 0x6a, 0x79, 0x73, 0x11, 0x65, 0x58, 0x46, 0x77, 0x06, 0x60, 0x45, 0xc9, 0x59, 0xed, 0x0f, 0x99, 0x29, 0x68, 0x8d, 0x04, 0xde, 0xfc, 0x29}

		hasher := hasher.Sha512{}
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
