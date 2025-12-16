package cmd

import "fmt"

var Version = "0.0.1765883021"

func Help() error {
	fmt.Printf(`simple-registry v%s
A lightweight OCI-compatible container registry with RBAC support.
Copyright 2025 José Luis Salvador Rufo <salvador.joseluis@gmail.com>

Usage:
  simple-registry -genhash
      Generate a password hash and exit.

  simple-registry [options]
      Start the registry server.

  simple-registry -version
      Print the version and exit.

Options:
  Required:
	-adminname string
        Administrator username.
	-adminpwd string
		Administrator password
		(use with care, ignored if -adminpwd-file is set).
    -adminpwd-file string
        File containing the administrator password
		(takes precedence over -adminpwd).

  Optional flags:
	-addr string
        Listening address (default "0.0.0.0:5000")
	-datadir string
        Data directory (default "./data")
	-cert string
        TLS certificate (enables HTTPS).
	-key string
        TLS key.
	-rbacdir string
        Directory containing RBAC YAML definitions.

Examples:
  simple-registry \
    -adminname admin \
    -adminpwd secret \
    -datadir /var/lib/registry \
    -cert cert.pem \
    -key key.pem
`, Version)
	return nil
}
