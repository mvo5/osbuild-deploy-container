package progress

import (
	"fmt"
	"io"
	"strings"
)

type debugProgressBar struct {
	w io.Writer
}

// NewDebugProgressBar will create a progressbar aimed to debug the
// lower level osbuild/images message. It will never clear the screen
// so "glitches/weird" messages from the lower-layers can be inspected
// easier.
func NewDebugProgressBar() (ProgressBar, error) {
	b := &debugProgressBar{w: osStderr}
	return b, nil
}

func (b *debugProgressBar) SetPulseMsgf(msg string, args ...interface{}) {
	fmt.Fprintf(b.w, "pulse: ")
	fmt.Fprintf(b.w, msg, args...)
	fmt.Fprintf(b.w, "\n")
}

func (b *debugProgressBar) SetMessagef(msg string, args ...interface{}) {
	fmt.Fprintf(b.w, "msg: ")
	fmt.Fprintf(b.w, msg, args...)
	fmt.Fprintf(b.w, "\n")
}

func (b *debugProgressBar) Start() {
	fmt.Fprintf(b.w, "Start progressbar\n")
}

func (b *debugProgressBar) Stop() {
	fmt.Fprintf(b.w, "Stop progressbar\n")
}

func (b *debugProgressBar) SetProgress(subLevel int, msg string, done int, total int) error {
	fmt.Fprintf(b.w, "%s[%v / %v] %s", strings.Repeat("  ", subLevel), done, total, msg)
	fmt.Fprintf(b.w, "\n")
	return nil
}
