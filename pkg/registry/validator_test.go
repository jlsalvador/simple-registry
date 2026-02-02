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

package registry_test

import (
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/registry"
)

func TestRegExprName(t *testing.T) {
	validNames := []string{
		"ubuntu",
		"library/ubuntu",
		"my-repo/my-image",
		"registry.example.com/namespace/image",
		"a0",
		"test123",
		"multi/level/deep/image",
		"name_with_underscore",
		"name__double__underscore",
		"hyphen-image",
		"triple---hyphen",
		"many-------hyphens",
		"dot.image",
		"complex-name_123.test/sub_path",
	}

	invalidNames := []string{
		"",
		"UPPERCASE",
		"Image",
		"-starting-hyphen",
		"ending-hyphen-",
		"_starting_underscore",
		".starting.dot",
		"double..dot",
		"name/",
		"/name",
		"//double",
		"special@char",
		"space name",
		"name#tag",
	}

	for _, name := range validNames {
		if !registry.RegExprName.MatchString(name) {
			t.Errorf("expected valid name: %s", name)
		}
	}

	for _, name := range invalidNames {
		if registry.RegExprName.MatchString(name) {
			t.Errorf("expected invalid name: %s", name)
		}
	}
}

func TestRegExprTag(t *testing.T) {
	validTags := []string{
		"latest",
		"v1.0.0",
		"1.2.3",
		"develop",
		"feature_branch",
		"release-candidate",
		"build.123",
		"a",
		"A",
		"_tag",
		"tag_with_underscores",
		"tag-with-hyphens",
		"tag.with.dots",
		"MixedCase123",
		"0123456789",
	}

	invalidTags := []string{
		"",
		"-starting-hyphen",
		".starting.dot",
		"tag with spaces",
		"tag@special",
		"tag#hash",
		"tag/slash",
		"this-is-a-very-long-tag-name-that-exceeds-the-maximum-allowed-length-of-128-characters-and-should-not-be-valid-according-to-the-regex-pattern-defined",
	}

	for _, tag := range validTags {
		if !registry.RegExprTag.MatchString(tag) {
			t.Errorf("expected valid tag: %s", tag)
		}
	}

	for _, tag := range invalidTags {
		if registry.RegExprTag.MatchString(tag) {
			t.Errorf("expected invalid tag: %s", tag)
		}
	}
}

func TestRegExprDigest(t *testing.T) {
	validDigests := []string{
		"sha256:abcdef1234567890",
		"sha512:1234567890abcdef",
		"md5:abc123",
		"sha256:a1b2c3d4e5f6",
		"algo:digest123",
		"sha256:ABCDEF123456",
		"test+algo:digest",
		"test.algo:digest",
		"test_algo:digest",
		"test-algo:digest",
	}

	invalidDigests := []string{
		"",
		"noalgorithm",
		":noalgo",
		"algo:",
		"SHA256:digest", // Upper case algo.
		"algo:digest with spaces",
		"algo:digest@special",
		"algo::double",
		"algo#digest",
	}

	for _, digest := range validDigests {
		if !registry.RegExprDigest.MatchString(digest) {
			t.Errorf("expected valid digest: %s", digest)
		}
	}

	for _, digest := range invalidDigests {
		if registry.RegExprDigest.MatchString(digest) {
			t.Errorf("expected invalid digest: %s", digest)
		}
	}
}

func TestRegExprUUID(t *testing.T) {
	validUUIDs := []string{
		"123e4567-e89b-12d3-a456-426614174000",
		"550e8400-e29b-41d4-a716-446655440000",
		"00000000-0000-0000-0000-000000000000",
		"FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF",
		"abcdef12-3456-7890-abcd-ef1234567890",
		"ABCDEF12-3456-7890-ABCD-EF1234567890",
		"aBcDeF12-3456-7890-AbCd-Ef1234567890",
	}

	invalidUUIDs := []string{
		"",                        // Empty.
		"not-a-uuid",              // Not a UUID.
		"123e4567-e89b-12d3-a456", // Too short.
		"123e4567-e89b-12d3-a456-426614174000-extra", // Too long.
		"123e4567e89b12d3a456426614174000",           // Without dashes.
		"123e4567-e89b-12d3-a456-42661417400g",       // Invalid chars.
		"123e4567_e89b_12d3_a456_426614174000",       // Lower dashes.
		"zzz-e89b-12d3-a456-426614174000",            // Short groups.
	}

	for _, uuid := range validUUIDs {
		if !registry.RegExprUUID.MatchString(uuid) {
			t.Errorf("expected valid UUID: %s", uuid)
		}
	}

	for _, uuid := range invalidUUIDs {
		if registry.RegExprUUID.MatchString(uuid) {
			t.Errorf("expected invalid UUID: %s", uuid)
		}
	}
}
