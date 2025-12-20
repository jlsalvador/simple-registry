package cmd

import (
	"fmt"

	"github.com/jlsalvador/simple-registry/internal/version"
)

func Help(short bool) error {
	if !short {
		fmt.Printf(`simple-registry v%s
A lightweight OCI-compatible container registry with RBAC support.
Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
`, version.AppVersion)
	}

	fmt.Print(`
Usage:
  simple-registry -genhash
    Generate a password hash and exit.

  simple-registry [options]
    Start the registry server.

  simple-registry -version
    Print the version and exit.

Options:
  -rbacdir string
    Directory containing RBAC YAML definitions.

  -adminname string
    Administrator username.
    (ignored if -rbacdir is set).
  -adminpwd string
    Administrator password.
    (use with care)
    (ignored if -adminpwd-file is set).
    (ignored if -rbacdir is set).
  -adminpwd-file string
    File containing the administrator password.
    (ignored if -rbacdir is set).

  -addr string
    Listening address (default "0.0.0.0:5000")
  -datadir string
    Data directory (default "./data")
  -cert string
    TLS certificate (enables HTTPS).
  -key string
    TLS key.

`)

	if !short {
		fmt.Print(`Example:
  simple-registry \
    -adminname admin \
    -adminpwd secret \
    -datadir /var/lib/registry \
    -cert cert.pem \
    -key key.pem
`)
	}

	return nil
}
