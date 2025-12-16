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

	"github.com/jlsalvador/simple-registry/internal/cmd"
)

func main() {
	flag.Usage = func() {
		_ = cmd.Help(true)
	}

	showHelp := false
	flag.BoolVar(&showHelp, "help", false, "")
	flag.BoolVar(&showHelp, "h", false, "")

	genHash := flag.Bool("genhash", false, "Generate a hash for the given password and exit")

	rbacDir := flag.String("rbacdir", "", "Directory with YAML files for RBAC")

	adminName := flag.String("adminname", "", "Administrator name")
	adminPwd := flag.String("adminpwd", "", "Administrator password")
	adminPwdFile := flag.String("adminpwd-file", "", "Fetch administrator password from file")

	certFile := flag.String("cert", "", "TLS certificate")
	keyFile := flag.String("key", "", "TLS key")
	addr := flag.String("addr", "0.0.0.0:5000", "Listening address")
	dataDir := flag.String("datadir", "./data", "Data directory")

	showVersion := flag.Bool("version", false, "Print the version and exit.")

	flag.Parse()

	var err error
	switch {
	case *showVersion:
		err = cmd.ShowVersion()

	case *genHash:
		err = cmd.GenerateHash()

	case (*adminName != "" && *adminPwd != "") || *rbacDir != "":
		err = cmd.Serve(
			*addr,
			*dataDir,
			*adminName,
			*adminPwd,
			*adminPwdFile,
			*certFile,
			*keyFile,
			*rbacDir,
		)

	default:
		err = cmd.Help(false)
	}

	if err != nil {
		panic(err)
	}
}
