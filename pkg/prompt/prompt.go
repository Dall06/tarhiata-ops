package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Confirm solicita confirmación interactiva al usuario (s/N). Retorna true si el usuario responde afirmativamente.
func Confirm(promptMsg string, autoYes bool) (bool, error) {
	if autoYes || os.Getenv("TARHIATA_AUTO_YES") == "true" {
		return true, nil
	}
	return ConfirmWithReader(os.Stdin, promptMsg)
}

// ConfirmWithReader lee la confirmación desde un io.Reader explícito (útil para pruebas unitarias).
func ConfirmWithReader(r io.Reader, promptMsg string) (bool, error) {
	fmt.Printf("%s (s/N): ", promptMsg)
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, err
		}
		return false, nil
	}
	val := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return val == "s" || val == "si" || val == "y" || val == "yes", nil
}
