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

package route_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jlsalvador/simple-registry/pkg/http/route"
)

func TestHeadResponseWriter_Write(t *testing.T) {
	tests := []struct {
		name          string
		data          []byte
		expectedLen   int
		expectedError error
	}{
		{
			name:          "writes empty byte slice",
			data:          []byte{},
			expectedLen:   0,
			expectedError: nil,
		},
		{
			name:          "writes small byte slice",
			data:          []byte("Hello, World!"),
			expectedLen:   13,
			expectedError: nil,
		},
		{
			name:          "writes large byte slice",
			data:          make([]byte, 1024*1024), // 1 MB
			expectedLen:   1024 * 1024,
			expectedError: nil,
		},
		{
			name:          "writes nil byte slice",
			data:          nil,
			expectedLen:   0,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test ResponseWriter.
			recorder := httptest.NewRecorder()

			// Wrap it with HeadResponseWriter.
			headWriter := &route.HeadResponseWriter{
				ResponseWriter: recorder,
			}

			// Call Write.
			n, err := headWriter.Write(tt.data)

			// Check the return values.
			if n != tt.expectedLen {
				t.Errorf("Write() returned length = %d, expected %d", n, tt.expectedLen)
			}

			if err != tt.expectedError {
				t.Errorf("Write() returned error = %v, expected %v", err, tt.expectedError)
			}

			// Verify that the underlying ResponseWriter did NOT receive the data.
			if recorder.Body.Len() != 0 {
				t.Errorf("HeadResponseWriter should discard body, but got %d bytes", recorder.Body.Len())
			}
		})
	}
}

func TestHeadResponseWriter_PreservesStatusCode(t *testing.T) {
	recorder := httptest.NewRecorder()
	headWriter := &route.HeadResponseWriter{
		ResponseWriter: recorder,
	}

	// Write a status code.
	headWriter.WriteHeader(http.StatusNotFound)

	// Verify status code is preserved.
	if recorder.Code != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d", http.StatusNotFound, recorder.Code)
	}
}

func TestHeadResponseWriter_PreservesHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	headWriter := &route.HeadResponseWriter{
		ResponseWriter: recorder,
	}

	// Set some headers.
	headWriter.Header().Set("Content-Type", "application/json")
	headWriter.Header().Set("X-Custom-Header", "test-value")

	// Verify headers are preserved.
	if ct := recorder.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", ct)
	}

	if custom := recorder.Header().Get("X-Custom-Header"); custom != "test-value" {
		t.Errorf("expected X-Custom-Header 'test-value', got '%s'", custom)
	}
}

func TestHeadResponseWriter_MultipleWrites(t *testing.T) {
	recorder := httptest.NewRecorder()
	headWriter := &route.HeadResponseWriter{
		ResponseWriter: recorder,
	}

	// Perform multiple writes.
	data1 := []byte("First write")
	data2 := []byte("Second write")
	data3 := []byte("Third write")

	n1, err1 := headWriter.Write(data1)
	n2, err2 := headWriter.Write(data2)
	n3, err3 := headWriter.Write(data3)

	// Verify each write returns correct length.
	if n1 != len(data1) || err1 != nil {
		t.Errorf("first write failed: n=%d, err=%v", n1, err1)
	}
	if n2 != len(data2) || err2 != nil {
		t.Errorf("second write failed: n=%d, err=%v", n2, err2)
	}
	if n3 != len(data3) || err3 != nil {
		t.Errorf("third write failed: n=%d, err=%v", n3, err3)
	}

	// Verify no data was actually written.
	if recorder.Body.Len() != 0 {
		t.Errorf("expected no body data, got %d bytes", recorder.Body.Len())
	}
}

func BenchmarkHeadResponseWriter_Write(b *testing.B) {
	recorder := httptest.NewRecorder()
	headWriter := &route.HeadResponseWriter{
		ResponseWriter: recorder,
	}
	data := []byte("This is some test data for benchmarking")

	for b.Loop() {
		headWriter.Write(data)
	}
}
