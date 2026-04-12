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

package generatehash

import (
	"fmt"
	"io"
	"os"

	cliTerm "github.com/jlsalvador/simple-registry/pkg/cli/term"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

const CmdName = "genhash"
const CmdHelp = "Generate a hash for the given password and exit"

var IsTerminal = cliTerm.IsTerminal
var ReadPassword = term.ReadPassword

func CmdFn() error {
	var pwd []byte
	var err error
	if !IsTerminal(os.Stdin) {
		pwd, err = io.ReadAll(os.Stdin)
	} else {
		fmt.Fprint(os.Stderr, "Enter password (no echo): ")
		pwd, err = ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr, "")
	}
	if err != nil {
		return fmt.Errorf("failed to read from stdin: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword(pwd, bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "%s\n", hash)

	return nil
}
