package cmd

import (
	"fmt"

	"github.com/jlsalvador/simple-registry/internal/version"
)

func Help(short bool) error {
	if !short {
		fmt.Printf(`%s v%s
A lightweight OCI-compatible container registry with RBAC support.
Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>
`, version.AppName, version.AppVersion)
	}

	fmt.Printf(`
Usage:
  %s -genhash
    Generate a password hash and exit.

  %s [options]
    Start the registry server.

  %s -version
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

`, version.AppName, version.AppName, version.AppName)

	if !short {
		fmt.Printf(`Example:
  %s \
    -adminname admin \
    -adminpwd secret \
    -datadir /var/lib/registry \
    -cert cert.pem \
    -key key.pem
`, version.AppName)
	}

	return nil
}
