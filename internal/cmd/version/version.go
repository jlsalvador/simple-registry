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

package version

import (
	"fmt"
	"runtime/debug"

	"github.com/jlsalvador/simple-registry/internal/version"
)

const CmdName = "version"
const CmdHelp = "Print the version and exit"

func CmdFn() error {
	fmt.Printf("%s\tv%s\n", version.AppName, version.AppVersion)
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		fmt.Println(buildInfo.String())
	}
	return nil
}
