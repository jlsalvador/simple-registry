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

// Package digest provides functions to parse, validate, and generate digests.
package digest

import (
	"errors"
	"strings"

	"github.com/jlsalvador/simple-registry/pkg/hasher"
)

var (
	ErrInvalidDigestFormat  = errors.New("invalid digest format")
	ErrEmptyAlgorithm       = errors.New("empty algorithm")
	ErrEmptyHash            = errors.New("empty hash")
	ErrUnsupportedAlgorithm = errors.New("unsupported algorithm")
)

// Parse splits a digest into algorithm and hex hash.
//
// Example:
//
//	Parse("sha256:abcd...") == "sha256", "abcd...", nil
func Parse(digest string) (algo, hex string, err error) {
	parts := strings.SplitN(digest, ":", 2)
	if len(parts) != 2 {
		return "", "", ErrInvalidDigestFormat
	}
	algo = parts[0]
	hex = parts[1]

	if algo == "" {
		return "", "", ErrEmptyAlgorithm
	}
	if hex == "" {
		return "", "", ErrEmptyHash
	}

	return algo, hex, nil
}

// NewHasher returns a new Hasher for the given algorithm.
//
// Supported algorithms:
//   - sha256
//   - sha512.
func NewHasher(algo string) (hasher.Hasher, error) {
	switch algo {
	case "sha256":
		return hasher.NewSha256(), nil
	case "sha512":
		return hasher.NewSha512(), nil
	default:
		return nil, ErrUnsupportedAlgorithm
	}
}
