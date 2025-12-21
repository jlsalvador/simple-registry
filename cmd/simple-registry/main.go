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

package main

import (
	"flag"
	"fmt"
	"os"
	"slices"

	cmdGenHash "github.com/jlsalvador/simple-registry/internal/cmd/generate_hash"
	cmdServe "github.com/jlsalvador/simple-registry/internal/cmd/serve"
	cmdVersion "github.com/jlsalvador/simple-registry/internal/cmd/version"
	"github.com/jlsalvador/simple-registry/internal/version"
)

type Cmd struct {
	Name string
	Help string
	Fn   func() error
}

var cmds = []Cmd{
	{cmdGenHash.CmdName, cmdGenHash.CmdHelp, cmdGenHash.CmdFn},
	{cmdServe.CmdName, cmdServe.CmdHelp, cmdServe.CmdFn},
	{cmdVersion.CmdName, cmdVersion.CmdHelp, cmdVersion.CmdFn},
}

func help() {
	fmt.Printf("Usage: %s [command]\n\nCommands:\n", version.AppName)
	for _, cmd := range cmds {
		fmt.Printf("  %s\n        %s\n", cmd.Name, cmd.Help)
	}
	fmt.Println()
}

func main() {
	cmdMain := flag.NewFlagSet(version.AppName, flag.ExitOnError)
	cmdMain.Usage = help
	cmdMain.Parse(os.Args[1:])

	if len(os.Args) < 2 {
		help()
		os.Exit(1)
	}

	var err error
	if i := slices.IndexFunc(cmds, func(cmd Cmd) bool {
		return cmd.Name == os.Args[1]
	}); i >= 0 {
		err = cmds[i].Fn()
	} else {
		err = fmt.Errorf("unknown command: %s\n", os.Args[1])
		help()
	}

	if err != nil {
		panic(err)
	}
}
