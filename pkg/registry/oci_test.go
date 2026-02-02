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

func TestNewImageIndexManifest(t *testing.T) {
	const want = "application/vnd.oci.image.index.v1+json"

	manifest := registry.NewImageIndexManifest()

	if manifest.SchemaVersion != 2 {
		t.Errorf("Expected SchemaVersion to be 2, got %d", manifest.SchemaVersion)
	}

	if manifest.MediaType != want {
		t.Errorf("Expected MediaType to be %s, got %s", want, manifest.MediaType)
	}
}
