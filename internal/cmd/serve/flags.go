package serve

import (
	"flag"
	"os"
	"strings"

	"github.com/jlsalvador/simple-registry/internal/cmd"
	cliFlag "github.com/jlsalvador/simple-registry/pkg/cli/flag"
	"github.com/jlsalvador/simple-registry/pkg/common"
)

type Flags struct {
	Addr    string
	DataDir string

	CfgDir cliFlag.StringSlice

	AdminName    string
	AdminPwd     string
	AdminPwdFile string

	CertFile string
	KeyFile  string
}

func parseFlags() (flags Flags, err error) {
	flagSet := flag.NewFlagSet("", flag.ExitOnError)
	flagSet.StringVar(&flags.Addr, "addr", common.GetEnv(cmd.ENV_PREFIX+"ADDR", "0.0.0.0:5000"), "Listening address")
	flagSet.StringVar(&flags.DataDir, "datadir", common.GetEnv(cmd.ENV_PREFIX+"DATADIR", "./data"), "Data directory")

	flagSet.Var(&flags.CfgDir, "cfgdir", "Directory with YAML configuration files\nCould be specified multiple times")

	flagSet.StringVar(&flags.AdminName, "adminname", common.GetEnv(cmd.ENV_PREFIX+"ADMINNAME", "admin"), "Administrator name\nIgnored if -cfgdir is set")
	flagSet.StringVar(&flags.AdminPwd, "adminpwd", common.GetEnv(cmd.ENV_PREFIX+"ADMINPWD", ""), "Administrator password\nLeaked by procfs! use adminpwdfile instead\nIgnored if -adminpwdfile is set\nIgnored if -cfgdir is set")
	flagSet.StringVar(&flags.AdminPwdFile, "adminpwdfile", common.GetEnv(cmd.ENV_PREFIX+"ADMINPWDFILE", ""), "Fetch administrator password from file\nIgnored if -cfgdir is set")

	flagSet.StringVar(&flags.CertFile, "certfile", common.GetEnv(cmd.ENV_PREFIX+"CERTFILE", ""), "TLS certificate file\nEnables HTTPS")
	flagSet.StringVar(&flags.KeyFile, "keyfile", common.GetEnv(cmd.ENV_PREFIX+"KEYFILE", ""), "TLS key file")

	if err = flagSet.Parse(os.Args[2:]); err != nil {
		return
	}

	if envVal, ok := os.LookupEnv(cmd.ENV_PREFIX + "CFGDIR"); len(flags.CfgDir) == 0 && ok {
		dirs := strings.SplitSeq(envVal, ",")
		for d := range dirs {
			flags.CfgDir = append(flags.CfgDir, strings.TrimSpace(d))
		}
	}

	return
}
