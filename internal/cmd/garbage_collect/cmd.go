package garbagecollect

import (
	"fmt"

	"github.com/jlsalvador/simple-registry/internal/config"
)

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

	deletedBlobs, _, seenBlobs, _, err := GarbageCollect(
		*cfg,
		flags.DryRun,
		flags.LastAccess,
		flags.DeleteUntagged,
	)
	if err != nil {
		return err
	}

	nDeleted := 0
	for digest := range deletedBlobs {
		nDeleted++
		if flags.DryRun {
			fmt.Printf("blob eligible for deletion: %s\n", digest)
		} else {
			fmt.Printf("blob deleted: %s\n", digest)
		}
	}
	if flags.DryRun {
		fmt.Printf("%d blobs marked, %d blobs eligible for deletion", len(seenBlobs), nDeleted)
	} else {
		fmt.Printf("%d blobs marked, %d blobs deleted\n", len(seenBlobs), nDeleted)
	}

	return nil
}
