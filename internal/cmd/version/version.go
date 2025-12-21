package version

import (
	"fmt"

	"github.com/jlsalvador/simple-registry/internal/version"
)

const CmdName = "version"
const CmdHelp = "Print the version and exit"

func CmdFn() error {
	fmt.Printf("%s v%s\n", version.AppName, version.AppVersion)
	return nil
}
