package generatehash

import (
	"fmt"
	"os"

	cliTerm "github.com/jlsalvador/simple-registry/pkg/cli/term"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

const CmdName = "genhash"
const CmdHelp = "Generate a hash for the given password and exit"

func isPiped() bool {
	return !cliTerm.IsTerminal(os.Stdin)
}

func CmdFn() error {
	var pwd []byte
	var err error
	if isPiped() {
		pwd, err = os.ReadFile(os.Stdin.Name())
	} else {
		fmt.Fprint(os.Stderr, "Enter password (no echo): ")
		pwd, err = term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr, "")
	}
	if err != nil {
		return fmt.Errorf("failed to read from stdin: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword(pwd, bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "%s\n", hash)

	return nil
}
