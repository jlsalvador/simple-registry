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

package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jlsalvador/simple-registry/internal/version"
	"github.com/jlsalvador/simple-registry/pkg/log"
	"github.com/jlsalvador/simple-registry/pkg/yamlscheme"
)

func parseYamlDir(dirName string) (manifests []any, err error) {
	entries, err := os.ReadDir(dirName)
	if err != nil {
		return nil, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		filename := filepath.Join(dirName, name)
		f, err := os.Open(filename)
		if err != nil {
			log.Error(
				"filename", filename,
				"err", err,
			).Print()
		}
		defer f.Close()

		m, err := yamlscheme.DecodeAll(f)
		if err != nil {
			return nil, err
		}

		fullFilenamePath, err := filepath.Abs(filename)
		if err != nil {
			return nil, err
		}

		log.Debug(
			"service.name", version.AppName,
			"service.version", version.AppVersion,
			"event.dataset", "cmd.serve",
			"file.path", fullFilenamePath,
			"message", fmt.Sprintf("%d manifest(s) added", len(m)),
		).Print()

		manifests = append(manifests, m...)
	}

	return manifests, nil
}
