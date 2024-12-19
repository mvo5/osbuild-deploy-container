package progress

import (
	"fmt"
	"io"
	"os"
)

var (
	osStderr io.Writer = os.Stderr
)

func cursorUp(i int) string {
	return fmt.Sprintf("%s[%dA", ESC, i)
}

// ProgressBar is an interface for progress reporting when there is
// an arbitrary amount of sub-progress information (like osbuild)
type ProgressBar interface {
	// SetProgress sets the progress details at the given "level".
	// Levels should start with "0" and increase as the nesting
	// gets deeper.
	//
	// Note that reducing depth is currently not supported, once
	// a sub-progress is added it cannot be removed/hidden
	// (but if required it can be added, its a SMOP)
	SetProgress(level int, msg string, done int, total int) error

	// The high-level message that is displayed in a spinner
	// that contains the current top level step, for bib this
	// is really just "Manifest generation step" and
	// "Image generation step". We could map this to a three-level
	// progress as well but we spend 90% of the time in the
	// "Image generation step" so the UI looks a bit odd.
	SetPulseMsgf(fmt string, args ...interface{})

	// A high level message with the last operation status.
	// For us this usually comes from the stages and has information
	// like "Starting module org.osbuild.selinux"
	SetMessagef(fmt string, args ...interface{})

	// Start will start rendering the progress information
	Start()

	// Stop will stop rendering the progress information, the
	// screen is not cleared, the last few lines will be visible
	Stop()
}

// New creates a new progressbar based on the requested type
func New(typ string) (ProgressBar, error) {
	switch typ {
	// XXX: autoseelct based on PS1 value (i.e. use term in
	// interactive shells only?)
	case "", "plain":
		return NewPlainProgressBar()
	case "term":
		return NewTerminalProgressBar()
	case "debug":
		return NewDebugProgressBar()
	default:
		return nil, fmt.Errorf("unknown progress type: %q", typ)
	}
}
