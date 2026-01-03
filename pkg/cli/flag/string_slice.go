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

// Package flag extends the functionality of the standard flag package by adding
// support for a custom string slice type.
//
// Usage:
//
//	// program -myslice val1 -myslice val2
//	var mySlice flag.StringSlice
//	flag.Var(&mySlice, "myslice", "Could be specified multiple times")
//	flag.Parse()
//	fmt.Println(mySlice)
//	// Output: [val1 val2]
package flag

import (
	"strings"
)

type StringSlice []string

func (s *StringSlice) String() string {
	return strings.Join(*s, ", ")
}

func (s *StringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}
