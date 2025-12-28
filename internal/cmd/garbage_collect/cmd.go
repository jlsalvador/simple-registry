package garbagecollect

import "github.com/jlsalvador/simple-registry/internal/config"

const CmdName = "garbage-collect"
const CmdHelp = "Removes blobs when they are no longer referenced by a manifest."

func CmdFn() error {
	flags, err := parseFlags()
	if err != nil {
		return err
	}

	var cfg *config.Config
	if len(flags.CfgDir) > 0 {
		cfg, err = config.NewFromYamlDir(
			flags.CfgDir,
			flags.DataDir,
		)
	} else {
		cfg, err = config.New(
			"",
			"-",
			"",
			flags.DataDir,
		)
	}

	return garbageCollect(
		*cfg,
		flags.DryRun,
		flags.LastAccess,
	)
}
