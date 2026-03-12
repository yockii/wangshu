package utils

import (
	"os"

	"github.com/charmbracelet/x/term"
)

func IsInteractiveTerminal() bool {
	b := term.IsTerminal(os.Stdin.Fd())
	return b
}
