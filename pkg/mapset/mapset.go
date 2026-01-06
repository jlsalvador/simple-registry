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
