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

package hasher

import (
	"crypto/sha512"
	"encoding/hex"
	"hash"
)

type Sha512 struct {
	h hash.Hash
}

// NewSha512 creates a [Hasher] instance that uses SHA-512 hashing algorithm.
func NewSha512() *Sha512 {
	return &Sha512{h: sha512.New()}
}

// Write writes data to the hash.
func (s *Sha512) Write(p []byte) (n int, err error) {
	if s.h == nil {
		s.h = sha512.New()
	}
	return s.h.Write(p)
}

// GetHash returns the hash value as a byte slice.
func (s *Sha512) GetHash() []byte {
	if s.h == nil {
		return nil
	}
	return s.h.Sum(nil)
}

// GetHashAsString returns the hash value as a hexadecimal string.
func (s *Sha512) GetHashAsString() string {
	return hex.EncodeToString(s.GetHash())
}
