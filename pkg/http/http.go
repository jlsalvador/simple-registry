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

package http

import (
	netHttp "net/http"
	"regexp"
	"strconv"

	httpErrors "github.com/jlsalvador/simple-registry/pkg/http/errors"
)

var regExpRange = regexp.MustCompile(`^([0-9]+)-([0-9]+)$`)

// Parse HTTP request header "Content-Range"
//
// Returns [ErrInvalidRange] on error.
func ParseRequestContentRange(r *netHttp.Request) (start int64, end int64, err error) {
	raw := r.Header.Get("Content-Range")
	if raw == "" {
		return -1, -1, httpErrors.ErrRequestedRangeNotSatisfiable
	}

	match := regExpRange.FindStringSubmatch(raw)
	if len(match) != 3 {
		return -1, -1, httpErrors.ErrRequestedRangeNotSatisfiable
	}

	start, _ = strconv.ParseInt(match[1], 10, 64) // ignore error because it's already validated by the regexp.
	end, _ = strconv.ParseInt(match[2], 10, 64)   // ignore error because it's already validated by the regexp.

	return start, end, nil
}
