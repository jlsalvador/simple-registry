package garbagecollect

import (
	"flag"
	"os"
	"time"

	cliFlag "github.com/jlsalvador/simple-registry/pkg/cli/flag"
)

type Flags struct {
	DataDir string
	CfgDir  []string

	DryRun         bool
	DeleteUntagged bool
	LastAccess     time.Duration
}

func parseFlags() (flags Flags, err error) {
	flagSet := flag.NewFlagSet("", flag.ExitOnError)

	dataDir := flagSet.String("datadir", "./data", "Data directory")
	cfgDir := cliFlag.FlagValueStringSlice{}
	flagSet.Var(&cfgDir, "cfgdir", "Directory with YAML configuration files\nCould be specified multiple times")

	deleteUntagged := flagSet.Bool("delete-untagged", false, "If set, the command will delete manifests that are not currently referenced by a tag.")
	dryRun := flagSet.Bool("dryrun", false, "If set, the command will not actually remove any blobs.")
	lastAccess := flagSet.String("last-access", "24h", "The time since the last access to a file before it is considered garbage.\nFormat: 1h, 2m, 3s, etc. Default: 24h.")

	if err = flagSet.Parse(os.Args[2:]); err != nil {
		return
	}

	flags.DataDir = *dataDir
	flags.CfgDir = cfgDir.Slice
	flags.DeleteUntagged = *deleteUntagged
	flags.DryRun = *dryRun
	flags.LastAccess, err = time.ParseDuration(*lastAccess)
	if err != nil {
		return
	}

	return
}
