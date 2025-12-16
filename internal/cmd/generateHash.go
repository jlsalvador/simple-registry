package cmd

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

func isPiped() bool {
	return !term.IsTerminal(int(os.Stdin.Fd()))
}

func GenerateHash() error {
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
