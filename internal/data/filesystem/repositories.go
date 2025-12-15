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
	"slices"
)

func (s *FilesystemDataStorage) RepositoriesList() ([]string, error) {
	reposDir := filepath.Join(s.base, "repositories")
	var respos []string

	err := filepath.WalkDir(reposDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		// Skip root
		if path == reposDir {
			return nil
		}

		// Skip internal registry directories
		if slices.Contains([]string{
			"_manifests",
			"_layers",
			"_uploads",
			"_links",
		}, d.Name()) {
			return filepath.SkipDir
		}

		// If the directory contains a _manifests directory, it's a repository
		if _, err := os.Stat(filepath.Join(path, "_manifests")); err == nil {

			// Get relative path as respoitory name
			rel, err := filepath.Rel(reposDir, path)
			if err != nil {
				return err
			}
			respos = append(respos, rel)

			// Repository already added, skip childrens
			return filepath.SkipDir
		}

		return nil
	})

	return respos, err
}
