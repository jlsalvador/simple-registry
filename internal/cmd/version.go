package cmd

import "fmt"

var Version = "0.0.1765883021"

func ShowVersion() error {
	fmt.Printf("simple-registry v%s\n", Version)
	return nil
}
