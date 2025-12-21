package generatehash

import (
	"fmt"
	"os"

	"github.com/jlsalvador/simple-registry/pkg/log"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

const CmdName = "genhash"
const CmdHelp = "Generate a hash for the given password and exit"

func isPiped() bool {
	return !log.IsTerminal(os.Stdin)
}

func CmdFn() error {
	var pwd []byte
	var err error
	if isPiped() {
		pwd, err = os.ReadFile(os.Stdin.Name())
	} else {
		fmt.Print("Enter password (no echo): ")
		pwd, err = term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Print("\n\n")
	}
	if err != nil {
		return fmt.Errorf("failed to read from stdin: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword(pwd, bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	fmt.Printf("Hash:\n%s\n", hash)

	return nil
}
