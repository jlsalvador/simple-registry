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

package filesystem

import (
	"os"
	"path/filepath"
)

func (s *FilesystemDataStorage) TagsList(repo string) ([]string, error) {
	tagsDir := filepath.Join(s.base, "repositories", repo, "_manifests", "tags")

	entries, err := os.ReadDir(tagsDir)
	if err != nil {
		return nil, err
	}

	var tags []string
	for _, entry := range entries {
		if entry.IsDir() {
			tags = append(tags, entry.Name())
		}
	}

	if len(tags) == 0 {
		return nil, nil
	}

	return tags, nil
}
