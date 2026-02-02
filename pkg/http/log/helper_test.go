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

package log_test

import (
	"net/http"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/http/log"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{
			name:     "X-Forwarded-For simple",
			headers:  map[string]string{"X-Forwarded-For": "10.0.0.1"},
			remote:   "192.168.1.1:1234",
			expected: "10.0.0.1",
		},
		{
			name:     "X-Forwarded-For multiple IPs (first is client)",
			headers:  map[string]string{"X-Forwarded-For": "10.0.0.1, 10.0.0.2"},
			remote:   "192.168.1.1:1234",
			expected: "10.0.0.1",
		},
		{
			name:     "X-Real-IP fallback",
			headers:  map[string]string{"X-Real-IP": "10.0.0.2"},
			remote:   "192.168.1.1:1234",
			expected: "10.0.0.2",
		},
		{
			name:     "X-Forwarded-For priority over X-Real-IP",
			headers:  map[string]string{"X-Forwarded-For": "10.0.0.1", "X-Real-IP": "10.0.0.2"},
			remote:   "192.168.1.1:1234",
			expected: "10.0.0.1",
		},
		{
			name:     "RemoteAddr fallback",
			headers:  map[string]string{},
			remote:   "192.168.1.50:1234",
			expected: "192.168.1.50",
		},
		{
			name:     "RemoteAddr IPv6",
			headers:  map[string]string{},
			remote:   "[2001:db8::1]:1234",
			expected: "2001:db8::1",
		},
		{
			name:     "RemoteAddr IPv6 with Zone",
			headers:  map[string]string{},
			remote:   "[fe80::1%eth0]:1234",
			expected: "fe80::1",
		},
		{
			name:     "Garbage in Header falls back to Remote",
			headers:  map[string]string{"X-Real-IP": "not-an-ip"},
			remote:   "127.0.0.1:5555",
			expected: "127.0.0.1",
		},
		{
			name:     "Empty",
			headers:  map[string]string{},
			remote:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remote
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			if got := log.GetClientIP(req); got != tt.expected {
				t.Errorf("GetClientIP() = %v, want %v", got, tt.expected)
			}
		})
	}
}
