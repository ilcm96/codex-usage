package output

import "golang.org/x/term"

func IsTerminal(fd uintptr) bool {
	return term.IsTerminal(int(fd))
}
