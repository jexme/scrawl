// Package windowerr reports a window that could not be opened. A GUI-subsystem exe has no
// console, so without this the program would just exit and the user would see nothing.
package windowerr

import (
	"fmt"
	"os"

	"github.com/jexme/scrawl/internal/i18n"
)

// Report prints "could not open the window: <err>" to stderr and shows it to the user
// (a message box on Windows, a notification elsewhere). It does nothing for a nil error.
func Report(err error) {
	if err == nil {
		return
	}
	msg := fmt.Sprintf(i18n.T("could not open the window: %v"), err)
	fmt.Fprintln(os.Stderr, msg)
	show("Scrawl", msg)
}
