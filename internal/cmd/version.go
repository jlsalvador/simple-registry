package cmd

import (
	"fmt"

	"github.com/jlsalvador/simple-registry/internal/version"
)

func ShowVersion() error {
	fmt.Printf("%s v%s\n", version.AppName, version.AppVersion)
	return nil
}
