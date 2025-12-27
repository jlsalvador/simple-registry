package garbagecollect

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jlsalvador/simple-registry/internal/cmd"
	cliFlag "github.com/jlsalvador/simple-registry/pkg/cli/flag"
	"github.com/jlsalvador/simple-registry/pkg/common"
)

type Flags struct {
	DataDir string
	CfgDir  cliFlag.StringSlice

	DryRun         bool
	DeleteUntagged bool
	LastAccess     time.Duration
}

func parseFlags() (flags Flags, err error) {
	flagSet := flag.NewFlagSet("", flag.ExitOnError)

	flagSet.StringVar(&flags.DataDir, "datadir", common.GetEnv(cmd.ENV_PREFIX+"DATADIR", "./data"), "Data directory")
	flagSet.Var(&flags.CfgDir, "cfgdir", "Directory with YAML configuration files\nCould be specified multiple times")

	var deleteUntagged string
	flagSet.StringVar(&deleteUntagged, "delete-untagged", common.GetEnv(cmd.ENV_PREFIX+"DELETE_UNTAGGED", "false"), "If set, the command will delete manifests that are not currently referenced by a tag.")
	var dryRun string
	flagSet.StringVar(&dryRun, "dryrun", common.GetEnv(cmd.ENV_PREFIX+"DRYRUN", "false"), "If set, the command will not actually remove any blobs.")
	var lastAccess string
	flagSet.StringVar(&lastAccess, "last-access", common.GetEnv(cmd.ENV_PREFIX+"LAST_ACCESS", "24h"), "The time since the last access to a file before it is considered garbage.\nFormat: 1h, 2m, 3s, etc. Default: 24h.")

	if err = flagSet.Parse(os.Args[2:]); err != nil {
		return
	}

	flags.DeleteUntagged, err = strconv.ParseBool(deleteUntagged)
	if err != nil {
		return
	}

	flags.DryRun, err = strconv.ParseBool(dryRun)
	if err != nil {
		return
	}

	flags.LastAccess, err = time.ParseDuration(lastAccess)
	if err != nil {
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
