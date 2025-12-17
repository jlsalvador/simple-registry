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

package rbac_test

import (
	"errors"
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/rbac"
)

func TestParseActions(t *testing.T) {
	allVerbs := []string{
		http.MethodHead,
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
		http.MethodConnect,
		http.MethodTrace,
	}
	slices.Sort(allVerbs)

	someVerbs := []string{http.MethodGet, http.MethodPost, http.MethodDelete}
	slices.Sort(someVerbs)

	tcs := []struct {
		name     string
		input    []string
		expected []string
		wantErr  error
	}{
		{
			name:     "valid actions",
			input:    []string{"get", "Post", " DeLeTe  "},
			expected: someVerbs,
			wantErr:  nil,
		},
		{
			name:     "wildcard action",
			input:    []string{"*"},
			expected: allVerbs,
			wantErr:  nil,
		},
		{
			name:     "invalid action",
			input:    []string{"Post", "gET", "PUT", "unknown"},
			expected: nil,
			wantErr:  rbac.ErrInvalidVerb,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			actions, err := rbac.ParseVerbs(tc.input)
			if tc.wantErr != nil && !errors.Is(err, tc.wantErr) {
				t.Errorf("ParseActions() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !reflect.DeepEqual(actions, tc.expected) {
				t.Errorf("ParseActions() got = %v, want %v", actions, tc.expected)
				return
			}
		})
	}
}
