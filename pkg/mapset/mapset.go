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

// Package mapset provides a set data structure implemented using Go's map.
package mapset

import (
	"fmt"
	"sort"
	"strings"
)

type MapSet[T comparable] map[T]struct{}

func NewMapSet[T comparable]() MapSet[T] {
	return MapSet[T]{}
}

func (s MapSet[T]) Add(elements ...T) MapSet[T] {
	for _, e := range elements {
		s[e] = struct{}{}
	}
	return s
}

func (s MapSet[T]) Contains(d T) bool {
	_, ok := s[d]
	return ok
}

func (s MapSet[T]) Equal(other MapSet[T]) bool {
	if len(s) != len(other) {
		return false
	}

	for key := range s {
		if _, ok := other[key]; !ok {
			return false
		}
	}

	return true
}

func (s MapSet[T]) String() string {
	items := make([]string, 0, len(s))
	for k := range s {
		items = append(items, fmt.Sprint(k))
	}

	sort.Strings(items)

	return "{" + strings.Join(items, ", ") + "}"
}
