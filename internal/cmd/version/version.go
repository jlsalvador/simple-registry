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
