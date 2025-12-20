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

package yamlscheme

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TestErrorReader is a mock reader that always returns an error.
type TestErrorReader struct{}

// Read implements the io.Reader interface. It always returns an error.
func (e *TestErrorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("forced read error")
}

func TestCompleteCoverage(t *testing.T) {
	// Reset manifests registry.
	registry = make(map[CommonManifest]func() any)

	t.Run("panic on duplicate register", func(t *testing.T) {
		defer func() { recover() }()
		Register[struct{}]("v1", "Test")
		Register[struct{}]("v1", "Test")
	})

	t.Run("error on initial decode", func(t *testing.T) {
		_, err := DecodeAll(&TestErrorReader{})
		if err == nil {
			t.Error("expected error from ErrorReader")
		}
	})

	t.Run("error on missing apiVersion or kind", func(t *testing.T) {
		// kind is miss.
		yaml1 := "apiVersion: v1"
		_, err := DecodeAll(strings.NewReader(yaml1))
		if err == nil || err.Error() != "missing apiVersion or kind" {
			t.Errorf("expected missing fields error, got %v", err)
		}

		// Empty object.
		yaml2 := "{}"
		_, err = DecodeAll(strings.NewReader(yaml2))
		if err == nil {
			t.Error("expected error for empty object")
		}
	})

	t.Run("error on unregistered type", func(t *testing.T) {
		yamlData := "apiVersion: v1\nkind: Unknown"
		_, err := DecodeAll(strings.NewReader(yamlData))
		if err == nil || !strings.Contains(err.Error(), "unregistered type") {
			t.Errorf("expected unregistered error, got %v", err)
		}
	})

	t.Run("error on final unmarshal (type mismatch)", func(t *testing.T) {
		type StrictObj struct {
			Number int `yaml:"number"`
		}
		Register[StrictObj]("v1", "Strict")

		yamlData := "apiVersion: v1\nkind: Strict\nnumber: [not an int]"
		_, err := DecodeAll(strings.NewReader(yamlData))
		if err == nil {
			t.Error("expected error in final decoding due to type mismatch")
		}
	})

	t.Run("success full flow", func(t *testing.T) {
		type Valid struct {
			ApiVersion string `yaml:"apiVersion"`
			Kind       string `yaml:"kind"`
			Data       string `yaml:"data"`
		}
		Register[Valid]("v1", "Valid")

		yamlData := "apiVersion: v1\nkind: Valid\ndata: Hello World!"
		res, err := DecodeAll(strings.NewReader(yamlData))
		if err != nil || len(res) != 1 {
			t.Fatalf("expected success, got err: %v, len: %d", err, len(res))
		}
	})
}

func TestMarshalUnmarshalErrors(t *testing.T) {
	// Mock yaml.Marshal and yaml.Unmarshal functions.
	oldMarshal := yamlMarshal
	oldUnmarshal := yamlUnmarshal
	defer func() {
		yamlMarshal = oldMarshal
		yamlUnmarshal = oldUnmarshal
	}()

	// tests cannot be run in parallel because they modify global variables.

	t.Run("force marshal error", func(t *testing.T) {
		yamlMarshal = func(v any) ([]byte, error) {
			return nil, fmt.Errorf("forced marshal error")
		}

		yamlData := "apiVersion: v1\nkind: Pod"
		_, err := DecodeAll(strings.NewReader(yamlData))
		if err == nil || err.Error() != "forced marshal error" {
			t.Errorf("expected forced marshal error, got %v", err)
		}
	})

	t.Run("force Unmarshal error", func(t *testing.T) {
		// yamlMarshal must be ok for this test.
		yamlMarshal = oldMarshal
		yamlUnmarshal = func(data []byte, v any) error {
			return fmt.Errorf("forced unmarshal error")
		}

		yamlData := "apiVersion: v1\nkind: Pod"
		_, err := DecodeAll(strings.NewReader(yamlData))
		if err == nil || err.Error() != "forced unmarshal error" {
			t.Errorf("expected forced unmarshal error, got %v", err)
		}
	})
}
