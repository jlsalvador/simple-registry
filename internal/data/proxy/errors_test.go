// Copyright 2026 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
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

package proxy_test

import (
	"testing"

	"github.com/jlsalvador/simple-registry/internal/data/proxy"
)

func TestErrors_Sentinels(t *testing.T) {
	if proxy.ErrDataStorageNotInitialized == nil {
		t.Error("ErrDataStorageNotInitialized should not be nil")
	}
	if proxy.ErrUpstreamError == nil {
		t.Error("ErrUpstreamError should not be nil")
	}
}
