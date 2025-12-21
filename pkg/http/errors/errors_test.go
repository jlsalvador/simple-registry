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

package errors_test

import (
	"testing"

	httpErrors "github.com/jlsalvador/simple-registry/pkg/http/errors"
)

func TestHttpError(t *testing.T) {
	tcs := []struct {
		err  httpErrors.HttpError
		want string
	}{
		{httpErrors.ErrBadRequest, "Bad Request"},
		{httpErrors.ErrUnauthorized, "Unauthorized"},
		{httpErrors.ErrRequestedRangeNotSatisfiable, "Requested Range Not Satisfiable"},
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
