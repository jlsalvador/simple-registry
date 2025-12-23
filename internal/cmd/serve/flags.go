package serve

import (
	"flag"
	"os"

	cliFlag "github.com/jlsalvador/simple-registry/pkg/cli/flag"
)

type Flags struct {
	Addr    string
	DataDir string

	CfgDir []string

	AdminName    string
	AdminPwd     string
	AdminPwdFile string

	CertFile string
	KeyFile  string
}

func parseFlags() (flags Flags, err error) {
	flagSet := flag.NewFlagSet("", flag.ExitOnError)
	addr := flagSet.String("addr", "0.0.0.0:5000", "Listening address")
	dataDir := flagSet.String("datadir", "./data", "Data directory")

	cfgDir := cliFlag.FlagValueStringSlice{}
	flagSet.Var(&cfgDir, "cfgdir", "Directory with YAML configuration files\nCould be specified multiple times")

	adminName := flagSet.String("adminname", "admin", "Administrator name\nIgnored if -cfgdir is set")
	adminPwd := flagSet.String("adminpwd", "", "Administrator password\nLeaked by procfs! use adminpwdfile instead\nIgnored if -adminpwdfile is set\nIgnored if -cfgdir is set")
	adminPwdFile := flagSet.String("adminpwdfile", "", "Fetch administrator password from file\nIgnored if -cfgdir is set")

	certFile := flagSet.String("certfile", "", "TLS certificate file\nEnables HTTPS")
	keyFile := flagSet.String("keyfile", "", "TLS key file")

	if err = flagSet.Parse(os.Args[2:]); err != nil {
		return
	}

	flags.Addr = *addr
	flags.DataDir = *dataDir
	flags.CfgDir = cfgDir.Slice
	flags.AdminName = *adminName
	flags.AdminPwd = *adminPwd
	flags.AdminPwdFile = *adminPwdFile
	flags.CertFile = *certFile
	flags.KeyFile = *keyFile

	return
}
