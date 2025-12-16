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

package digest_test

import (
	"errors"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/digest"
)

func TestParse(t *testing.T) {
	tcs := []struct {
		name         string
		digest       string
		expectedAlgo string
		expectedHash string
		expectedErr  error
	}{
		{
			name:         "valid sha256 digest",
			digest:       "sha256:64ec88ca00b268e5ba1a35678a1b5316d212f4f366b2477232534a8aeca37f3c",
			expectedAlgo: "sha256",
			expectedHash: "64ec88ca00b268e5ba1a35678a1b5316d212f4f366b2477232534a8aeca37f3c",
			expectedErr:  nil,
		},
		{
			name:         "valid sha512 digest",
			digest:       "sha512:b7f783baed8297f0db917462184ff4f08e69c2d5e5f79a942600f9725f58ce1f29c18139bf80b06c0fff2bdd34738452ecf40c488c22a7e3d80cdf6f9c1c0d47",
			expectedAlgo: "sha512",
			expectedHash: "b7f783baed8297f0db917462184ff4f08e69c2d5e5f79a942600f9725f58ce1f29c18139bf80b06c0fff2bdd34738452ecf40c488c22a7e3d80cdf6f9c1c0d47",
			expectedErr:  nil,
		},
		{
			name:         "invalid digest format",
			digest:       "sha256abcd...",
			expectedAlgo: "",
			expectedHash: "",
			expectedErr:  digest.ErrInvalidDigestFormat,
		},
		{
			name:         "unsupported algorithm",
			digest:       "md5:3e25960a79dbc69b674cd4ec67a72c62",
			expectedAlgo: "md5",
			expectedHash: "3e25960a79dbc69b674cd4ec67a72c62",
			expectedErr:  nil,
		},
		{
			name:         "empty hash",
			digest:       "sha256:",
			expectedAlgo: "",
			expectedHash: "",
			expectedErr:  digest.ErrEmptyHash,
		},
		{
			name:         "empty algorithm",
			digest:       ":64ec88ca00b268e5ba1a35678a1b5316d212f4f366b2477232534a8aeca37f3c",
			expectedAlgo: "",
			expectedHash: "",
			expectedErr:  digest.ErrEmptyAlgorithm,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			algo, hash, err := digest.Parse(tc.digest)
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("expected error %v, got %v", tc.expectedErr, err)
			}
			if algo != tc.expectedAlgo {
				t.Errorf("expected algorithm %s, got %s", tc.expectedAlgo, algo)
			}
			if hash != tc.expectedHash {
				t.Errorf("expected hash %s, got %s", tc.expectedHash, hash)
			}
		})
	}
}

func TestNewHasher(t *testing.T) {
	tcs := []struct {
		name    string
		wantErr error
	}{
		{
			name:    "sha256",
			wantErr: nil,
		},
		{
			name:    "sha512",
			wantErr: nil,
		},
		{
			name:    "md5",
			wantErr: digest.ErrUnsupportedAlgorithm,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := digest.NewHasher(tc.name)
			if (err != nil && tc.wantErr == nil) || (err == nil && tc.wantErr != nil) {
				t.Errorf("error = %v; wantErr %v", err, tc.wantErr)
			}
		})
	}
}
