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

// Package yamlscheme provides a way to decode multiples YAML manifests
// (requires the fields "apiVersion" and "kind") at once.
package yamlscheme

import (
	"bytes"
	"fmt"
	"io"

	"github.com/goccy/go-yaml"
)

var (
	registry = map[CommonManifest]func() any{}
)

// Mock.
var (
	yamlMarshal   = yaml.Marshal
	yamlUnmarshal = yaml.Unmarshal
)

// CommonManifest represents a YAML manifest with the required fields.
type CommonManifest struct {
	ApiVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
}

// Register registers a manifest type to be used when decoding YAML manifests.
func Register[T any](apiVersion, kind string) {
	k := CommonManifest{apiVersion, kind}
	if _, exists := registry[k]; exists {
		panic(fmt.Sprintf("type already registered: %s/%s", apiVersion, kind))
	}

	registry[k] = func() any {
		var zero T
		return &zero
	}
}

func newObject(apiVersion, kind string) (any, bool) {
	f, ok := registry[CommonManifest{apiVersion, kind}]
	if !ok {
		return nil, false
	}
	return f(), true
}

// DecodeAll decodes all registered YAML manifests from the given reader.
func DecodeAll(r io.Reader) ([]any, error) {
	dec := yaml.NewDecoder(r)

	var result []any

	for {
		var raw any
		err := dec.Decode(&raw)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		data, err := yamlMarshal(raw)
		if err != nil {
			return nil, err
		}

		var m CommonManifest
		if err := yamlUnmarshal(data, &m); err != nil {
			return nil, err
		}

		if m.ApiVersion == "" || m.Kind == "" {
			return nil, fmt.Errorf("missing apiVersion or kind")
		}

		obj, ok := newObject(m.ApiVersion, m.Kind)
		if !ok {
			return nil, fmt.Errorf("unregistered type %s/%s", m.ApiVersion, m.Kind)
		}

		if err := yaml.NewDecoder(bytes.NewReader(data)).Decode(obj); err != nil {
			return nil, err
		}

		result = append(result, obj)
	}

	return result, nil
}
