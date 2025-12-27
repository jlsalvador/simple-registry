package garbagecollect

const CmdName = "garbage-collect"
const CmdHelp = "Removes blobs when they are no longer referenced by a manifest."

func CmdFn() error {
	_, err := parseFlags()
	if err != nil {
		return err
	}

	return nil
}
