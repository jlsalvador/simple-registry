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

package yaml

import "time"

type CommonManifest struct {
	ApiVersion string `json:"apiVersion" yaml:"apiVersion"`
	Kind       string `json:"kind" yaml:"kind"`
	Metadata   struct {
		Name string `json:"name" yaml:"name"`
	} `json:"metadata" yaml:"metadata"`
}

type TokenManifest struct {
	CommonManifest

	Spec struct {
		Value     string    `json:"value" yaml:"value"`
		ExpiresAt time.Time `json:"expiresAt" yaml:"expiresAt"` // RFC3339 timestamp.
		Username  string    `json:"username" yaml:"username"`
	} `json:"spec" yaml:"spec"`
}

type UserManifest struct {
	CommonManifest

	Spec struct {
		PasswordHash string   `json:"passwordHash,omitempty" yaml:"passwordHash,omitempty"` // bcrypt hashed password.
		Groups       []string `json:"groups" yaml:"groups"`
	} `json:"spec" yaml:"spec"`
}

type RoleManifest struct {
	CommonManifest

	Spec struct {
		Resources []string `json:"resources" yaml:"resources"` // "catalog", "blobs", "manifests", "tags", or "*".
		Verbs     []string `json:"verbs" yaml:"verbs"`         // "HEAD", "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "TRACE", or "*".
	} `json:"spec" yaml:"spec"`
}

type RoleBindingManifest struct {
	CommonManifest

	Spec struct {
		Subjects []struct {
			Kind string `json:"kind" yaml:"kind"` // "User" or "Group".
			Name string `json:"name" yaml:"name"`
		} `json:"subjects" yaml:"subjects"`
		RoleRef struct {
			Name string `json:"name" yaml:"name"`
		} `json:"roleRef" yaml:"roleRef"`
		Scopes []string `json:"scopes" yaml:"scopes"` // Regular expressions matching the repository path."
	} `json:"spec" yaml:"spec"`
}
