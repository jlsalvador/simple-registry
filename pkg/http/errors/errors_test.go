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
	"errors"
	"net/http"
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
		{httpErrors.ErrNotFound, "Not Found"},
		{httpErrors.ErrRequestedRangeNotSatisfiable, "Requested Range Not Satisfiable"},
		{httpErrors.ErrInternalServerError, "Internal Server Error"},
	}

	for _, tc := range tcs {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()

			if got := tc.err.Error(); got != tc.want {
				t.Errorf("HttpError.Error() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestStatusCodeFromError(t *testing.T) {
	tcs := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "Returns status from BadRequest",
			err:  httpErrors.ErrBadRequest,
			want: http.StatusBadRequest,
		},
		{
			name: "Returns status from NotFound",
			err:  httpErrors.ErrNotFound,
			want: http.StatusNotFound,
		},
		{
			name: "Returns status from custom HttpError",
			err:  httpErrors.HttpError{Status: http.StatusTeapot},
			want: http.StatusTeapot,
		},
		{
			name: "Returns 500 for generic Go errors",
			err:  errors.New("generic database error"),
			want: http.StatusInternalServerError,
		},
		{
			name: "Returns 500 for nil error",
			err:  nil,
			want: http.StatusInternalServerError,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := httpErrors.StatusCodeFromError(tc.err); got != tc.want {
				t.Errorf("StatusCodeFromError() = %v, want %v", got, tc.want)
			}
		})
	}
}
