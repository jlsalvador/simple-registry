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

package http_test

import (
	"errors"
	netHttp "net/http"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/http"
	httpErrors "github.com/jlsalvador/simple-registry/pkg/http/errors"
)

func TestParseRequestContentRange(t *testing.T) {
	// Test cases for ParseRequestContentRange function
	tcs := []struct {
		name   string
		header map[string][]string
		start  int64
		end    int64
		err    error
	}{
		{
			name: "valid",
			header: map[string][]string{
				"Content-Range": {"0-1023"},
			},
			start: 0,
			end:   1023,
			err:   nil,
		},
		{
			name: "invalid key",
			header: map[string][]string{
				"Range": {"0-1023"},
			},
			start: -1,
			end:   -1,
			err:   httpErrors.ErrRequestedRangeNotSatisfiable,
		},
		{
			name: "invalid value",
			header: map[string][]string{
				"Content-Range": {"1023"},
			},
			start: -1,
			end:   -1,
			err:   httpErrors.ErrRequestedRangeNotSatisfiable,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := &netHttp.Request{}
			r.Header = tc.header

			start, end, err := http.ParseRequestContentRange(r)

			if !errors.Is(err, tc.err) {
				t.Errorf("expected error %v, got %v", tc.err, err)
			}
			if start != tc.start {
				t.Errorf("expected start %d, got %d", tc.start, start)
			}
			if end != tc.end {
				t.Errorf("expected end %d, got %d", tc.end, end)
			}
		})
	}
}
