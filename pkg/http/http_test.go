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
	"net/url"
	"reflect"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/http"
	httpErrors "github.com/jlsalvador/simple-registry/pkg/http/errors"
)

func TestParseRequestContentRange(t *testing.T) {
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

func TestPaginateString(t *testing.T) {
	items := []string{"a", "b", "c", "d", "e"}

	tcs := []struct {
		name     string
		last     string
		n        string
		expected []string
	}{
		{
			name:     "no parameters returns all items",
			last:     "",
			n:        "",
			expected: []string{"a", "b", "c", "d", "e"},
		},
		{
			name:     "filter with last skips items up to that value",
			last:     "b",
			n:        "",
			expected: []string{"c", "d", "e"}, // skips a, b
		},
		{
			name:     "filter with n limits the number of items",
			last:     "",
			n:        "2",
			expected: []string{"a", "b"},
		},
		{
			name:     "filter with last and n combined",
			last:     "a",
			n:        "2",
			expected: []string{"b", "c"},
		},
		{
			name:     "last not found returns all items (then applies n)",
			last:     "nonexistent",
			n:        "2",
			expected: []string{"a", "b"},
		},
		{
			name:     "n greater than length returns items after last",
			last:     "c",
			n:        "10",
			expected: []string{"d", "e"},
		},
		{
			name:     "invalid n returns all items after last",
			last:     "b",
			n:        "invalid",
			expected: []string{"c", "d", "e"},
		},
		{
			name:     "negative n returns all items after last",
			last:     "b",
			n:        "-1",
			expected: []string{"c", "d", "e"},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			u, _ := url.Parse("http://example.com")
			q := u.Query()
			if tc.last != "" {
				q.Set("last", tc.last)
			}
			if tc.n != "" {
				q.Set("n", tc.n)
			}
			u.RawQuery = q.Encode()

			req := &netHttp.Request{URL: u}
			got := http.PaginateString(items, req)

			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("PaginateString() = %v, want %v", got, tc.expected)
			}
		})
	}
}
