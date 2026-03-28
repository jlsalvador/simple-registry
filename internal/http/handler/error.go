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

package handler

import (
	"reflect"
	"runtime"

	"github.com/jlsalvador/simple-registry/internal/version"
	"github.com/jlsalvador/simple-registry/pkg/log"
)

type ErrorOCI struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var (
	ErrorBlobUnknown         = ErrorOCI{"BLOB_UNKNOWN", "blob unknown to registry"}
	ErrorBlobUploadInvalid   = ErrorOCI{"BLOB_UPLOAD_INVALID", "blob upload invalid"}
	ErrorBlobUploadUnknown   = ErrorOCI{"BLOB_UPLOAD_UNKNOWN", "blob upload unknown to registry"}
	ErrorDigestInvalid       = ErrorOCI{"DIGEST_INVALID", "provided digest did not match uploaded content"}
	ErrorManifestBlobUnknown = ErrorOCI{"MANIFEST_BLOB_UNKNOWN", "manifest references a manifest or blob unknown to registry"}
	ErrorManifestInvalid     = ErrorOCI{"MANIFEST_INVALID", "manifest invalid"}
	ErrorManifestUnknown     = ErrorOCI{"MANIFEST_UNKNOWN", "manifest unknown to registry"}
	ErrorNameInvalid         = ErrorOCI{"NAME_INVALID", "invalid repository name"}
	ErrorNameUnknown         = ErrorOCI{"NAME_UNKNOWN", "repository name not known to registry"}
	ErrorSizeInvalid         = ErrorOCI{"SIZE_INVALID", "provided length did not match content length"}
	ErrorUnauthorized        = ErrorOCI{"UNAUTHORIZED", "authentication required"}
	ErrorDenied              = ErrorOCI{"DENIED", "requested access to the resource is denied"}
	ErrorUnsupported         = ErrorOCI{"UNSUPPORTED", "the operation is unsupported"}
	ErrorTooManyRequests     = ErrorOCI{"TOOMANYREQUESTS", "too many requests"}
)

func LogError(err error) {
	const MaxStackDepth = 50

	if err == nil {
		return
	}

	stack := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(1, stack)

	log.Error(
		"service.name", version.AppName,
		"service.version", version.AppVersion,
		"event.dataset", "http.access",
		"error.message", err.Error(),
		"error.stack_trace", stack[:length],
		"error.type", reflect.TypeOf(err),
	).Print()
}
