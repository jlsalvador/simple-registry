package garbagecollect

import (
	"fmt"

	"github.com/jlsalvador/simple-registry/internal/config"
	"github.com/jlsalvador/simple-registry/internal/version"
	"github.com/jlsalvador/simple-registry/pkg/log"
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

	deletedBlobs, deletedManifests, seenBlobs, seenManifests, err := GarbageCollect(
		*cfg,
		flags.DryRun,
		flags.LastAccess,
		flags.DeleteUntagged,
	)
	if err != nil {
		return err
	}

	nManifestsDeleted := 0
	for digest := range deletedManifests {
		nManifestsDeleted++
		if flags.DryRun {
			log.Debug(
				"service.name", version.AppName,
				"service.version", version.AppVersion,
				"event.dataset", "cmd.garbage_collect",
				"message", fmt.Sprintf("manifest eligible for deletion: %s", digest),
			).Print()
		} else {
			log.Debug(
				"service.name", version.AppName,
				"service.version", version.AppVersion,
				"event.dataset", "cmd.garbage_collect",
				"message", fmt.Sprintf("manifest deleted: %s", digest),
			).Print()
		}
	}

	nBlobsDeleted := 0
	for digest := range deletedBlobs {
		nBlobsDeleted++
		if flags.DryRun {
			log.Debug(
				"service.name", version.AppName,
				"service.version", version.AppVersion,
				"event.dataset", "cmd.garbage_collect",
				"message", fmt.Sprintf("blob eligible for deletion: %s", digest),
			).Print()
		} else {
			log.Debug(
				"service.name", version.AppName,
				"service.version", version.AppVersion,
				"event.dataset", "cmd.garbage_collect",
				"message", fmt.Sprintf("blob deleted: %s", digest),
			).Print()
		}
	}
	if flags.DryRun {
		log.Info(
			"service.name", version.AppName,
			"service.version", version.AppVersion,
			"event.dataset", "cmd.garbage_collect",
			"message", fmt.Sprintf(
				"%d manifests marked, %d blobs marked, %d manifests eligible for deletion, %d blobs eligible for deletion",
				len(seenManifests), len(seenBlobs), nManifestsDeleted, nBlobsDeleted,
			),
		).Print()

	} else {
		log.Info(
			"service.name", version.AppName,
			"service.version", version.AppVersion,
			"event.dataset", "cmd.garbage_collect",
			"message", fmt.Sprintf(
				"%d manifests marked, %d blobs marked, %d manifests deleted, %d blobs deleted",
				len(seenManifests), len(seenBlobs), nManifestsDeleted, nBlobsDeleted,
			),
		).Print()
	}

	return nil
}
