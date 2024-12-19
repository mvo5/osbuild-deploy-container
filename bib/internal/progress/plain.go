package progress

import (
	"fmt"
	"io"
)

type plainProgressBar struct {
	w io.Writer
}

// NewPlainProgressBar starts a new "plain" progressbar that will just
// prints message but does not show any progress.
func NewPlainProgressBar() (ProgressBar, error) {
	b := &plainProgressBar{w: osStderr}
	return b, nil
}

func (b *plainProgressBar) SetPulseMsgf(msg string, args ...interface{}) {
	fmt.Fprintf(b.w, msg, args...)
}

func (b *plainProgressBar) SetMessagef(msg string, args ...interface{}) {
	fmt.Fprintf(b.w, msg, args...)
}

func (b *plainProgressBar) Start() {
}

func (b *plainProgressBar) Stop() {
}

func (b *plainProgressBar) SetProgress(subLevel int, msg string, done int, total int) error {
	return nil
}
