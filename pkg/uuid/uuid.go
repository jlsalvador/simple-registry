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

// Package uuid provides functions to generate
// Universally Unique Identifiers (UUIDs).
package uuid

import (
	"crypto/rand"
	"fmt"
)

// RandRead is the function used to read random bytes.
// It can be replaced for testing purposes.
var RandRead = rand.Read

type UUID []byte

func (u *UUID) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		(*u)[0:4],
		(*u)[4:6],
		(*u)[6:8],
		(*u)[8:10],
		(*u)[10:16])
}

// New generates a new UUID.
func New() (*UUID, error) {
	uuid := make([]byte, 16)

	// Read random bytes
	_, err := RandRead(uuid)
	if err != nil {
		return nil, err
	}

	// Set version (4) in bits 12-15 of octet 6.
	uuid[6] = (uuid[6] & 0x0f) | 0x40

	// Set variant (RFC 4122) in bits 6-7 of octet 8.
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	u := UUID(uuid)
	return &u, nil
}

// MustNew generates a UUID and panics on error.
func MustNew() *UUID {
	u, err := New()
	if err != nil {
		panic(err)
	}
	return u
}
