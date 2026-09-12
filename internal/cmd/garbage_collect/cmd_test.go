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

package garbagecollect

import (
	"os"
	"testing"
	"time"

	cliFlag "github.com/jlsalvador/simple-registry/pkg/cli/flag"
)

func TestParseFlags(t *testing.T) {
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	tests := []struct {
		name     string
		args     []string
		expected Flags
	}{
		{
			name: "default values",
			args: []string{"registry", "garbage-collect"},
			expected: Flags{
				DataDir:        "./data",
				DryRun:         false,
				DeleteUntagged: false,
				LastAccess:     24 * time.Hour,
			},
		},
		{
			name: "custom datadir and dry run",
			args: []string{"registry", "garbage-collect", "-datadir", "/tmp/data", "-dryrun"},
			expected: Flags{
				DataDir:        "/tmp/data",
				DryRun:         true,
				DeleteUntagged: false,
				LastAccess:     24 * time.Hour,
			},
		},
		{
			name: "with delete untagged and last access",
			args: []string{"registry", "garbage-collect", "-delete-untagged", "-last-access", "1h"},
			expected: Flags{
				DataDir:        "./data",
				DryRun:         false,
				DeleteUntagged: true,
				LastAccess:     1 * time.Hour,
			},
		},
		{
			name: "with cfgdir",
			args: []string{"registry", "garbage-collect", "-cfgdir", "/etc/config"},
			expected: Flags{
				DataDir:        "./data",
				CfgDir:         cliFlag.StringSlice{"/etc/config"},
				DryRun:         false,
				DeleteUntagged: false,
				LastAccess:     24 * time.Hour,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args
			flags, err := parseFlags()
			if err != nil {
				t.Fatalf("parseFlags() error = %v", err)
			}
			if flags.DataDir != tt.expected.DataDir {
				t.Errorf("DataDir = %v, want %v", flags.DataDir, tt.expected.DataDir)
			}
			if flags.DryRun != tt.expected.DryRun {
				t.Errorf("DryRun = %v, want %v", flags.DryRun, tt.expected.DryRun)
			}
			if flags.DeleteUntagged != tt.expected.DeleteUntagged {
				t.Errorf("DeleteUntagged = %v, want %v", flags.DeleteUntagged, tt.expected.DeleteUntagged)
			}
			if flags.LastAccess != tt.expected.LastAccess {
				t.Errorf("LastAccess = %v, want %v", flags.LastAccess, tt.expected.LastAccess)
			}
			if len(flags.CfgDir) != len(tt.expected.CfgDir) {
				t.Errorf("CfgDir length = %v, want %v", len(flags.CfgDir), len(tt.expected.CfgDir))
			} else {
				for i, v := range flags.CfgDir {
					if v != tt.expected.CfgDir[i] {
						t.Errorf("CfgDir[%d] = %v, want %v", i, v, tt.expected.CfgDir[i])
					}
				}
			}
		})
	}
}

func TestCmdFn(t *testing.T) {
	// Test CmdFn with dry run to avoid actual deletions.
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Create a temporary directory for data.
	tmpDir := t.TempDir()

	// Create the basic directory structure.
	repoDir := tmpDir + "/repositories"
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatal(err)
	}

	os.Args = []string{"registry", "garbage-collect", "-datadir", tmpDir, "-dryrun"}

	err := CmdFn()
	if err != nil {
		t.Errorf("CmdFn() error = %v", err)
	}
}
