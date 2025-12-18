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
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/http"
)

func TestHttpError(t *testing.T) {
	tcs := []struct {
		err  http.HttpError
		want string
	}{
		{http.ErrBadRequest, "Bad Request"},
		{http.ErrUnauthorized, "Unauthorized"},
		{http.ErrRequestedRangeNotSatisfiable, "Requested Range Not Satisfiable"},
	}

	for _, tc := range tcs {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()

			if got := tc.err.Error(); got != tc.want {
				t.Errorf("HttpError() = %v, want %v", got, tc.want)
			}
		})
	}
}
