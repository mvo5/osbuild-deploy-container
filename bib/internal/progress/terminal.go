package progress

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/sirupsen/logrus"
)

var (

	// This is only needed because pb.Pool require a real terminal.
	// It sets it into "raw-mode" but there is really no need for
	// this (see "func render()" below) so once this is fixed
	// upstream we should remove this.
	ESC         = "\x1b"
	ERASE_LINE  = ESC + "[2K"
	CURSOR_HIDE = ESC + "[?25l"
	CURSOR_SHOW = ESC + "[?25h"
)

type terminalProgressBar struct {
	spinnerPb   *pb.ProgressBar
	msgPb       *pb.ProgressBar
	subLevelPbs []*pb.ProgressBar

	shutdownCh chan bool

	out io.Writer
}

// NewTerminalProgressBar creates a new default pb3 based progressbar suitable for
// most terminals.
func NewTerminalProgressBar() (ProgressBar, error) {
	b := &terminalProgressBar{
		out: osStderr,
	}
	b.spinnerPb = pb.New(0)
	b.spinnerPb.SetTemplate(`[{{ (cycle . "|" "/" "-" "\\") }}] {{ string . "spinnerMsg" }}`)
	b.msgPb = pb.New(0)
	b.msgPb.SetTemplate(`Message: {{ string . "msg" }}`)
	return b, nil
}

func (b *terminalProgressBar) SetProgress(subLevel int, msg string, done int, total int) error {
	// auto-add as needed, requires sublevels to get added in order
	// i.e. adding 0 and then 2 will fail
	switch {
	case subLevel == len(b.subLevelPbs):
		apb := pb.New(0)
		progressBarTmpl := `[{{ counters . }}] {{ string . "prefix" }} {{ bar .}} {{ percent . }}`
		apb.SetTemplateString(progressBarTmpl)
		if err := apb.Err(); err != nil {
			return fmt.Errorf("error setting the progressbarTemplat: %w", err)
		}
		// workaround bug when running tests in tmt
		if apb.Width() == 0 {
			// this is pb.defaultBarWidth
			apb.SetWidth(100)
		}
		b.subLevelPbs = append(b.subLevelPbs, apb)
	case subLevel > len(b.subLevelPbs):
		return fmt.Errorf("sublevel added out of order, have %v sublevels but want level %v", len(b.subLevelPbs), subLevel)
	}
	apb := b.subLevelPbs[subLevel]
	apb.SetTotal(int64(total) + 1)
	apb.SetCurrent(int64(done) + 1)
	apb.Set("prefix", msg)
	return nil
}

func shorten(msg string) string {
	msg = strings.Replace(msg, "\n", " ", -1)
	// XXX: make this smarter
	if len(msg) > 60 {
		return msg[:60] + "..."
	}
	return msg
}

func (b *terminalProgressBar) SetPulseMsgf(msg string, args ...interface{}) {
	b.spinnerPb.Set("spinnerMsg", shorten(fmt.Sprintf(msg, args...)))
}

func (b *terminalProgressBar) SetMessagef(msg string, args ...interface{}) {
	b.msgPb.Set("msg", shorten(fmt.Sprintf(msg, args...)))
}

func (b *terminalProgressBar) render() {
	var renderedLines int
	fmt.Fprintf(b.out, "%s%s\n", ERASE_LINE, b.spinnerPb.String())
	renderedLines++
	for _, prog := range b.subLevelPbs {
		fmt.Fprintf(b.out, "%s%s\n", ERASE_LINE, prog.String())
		renderedLines++
	}
	fmt.Fprintf(b.out, "%s%s\n", ERASE_LINE, b.msgPb.String())
	renderedLines++
	fmt.Fprint(b.out, cursorUp(renderedLines))
}

// Workaround for the pb.Pool requiring "raw-mode" - see here how to avoid
// it. Once fixes upstream we should remove this.
func (b *terminalProgressBar) renderLoop() {
	for {
		select {
		case <-b.shutdownCh:
			b.render()
			// finally move cursor down again
			fmt.Fprint(b.out, CURSOR_SHOW)
			fmt.Fprint(b.out, strings.Repeat("\n", 2+len(b.subLevelPbs)))
			// close last to avoid race with b.out
			close(b.shutdownCh)
			return
		case <-time.After(200 * time.Millisecond):
			// break to redraw the screen
		}
		b.render()
	}
}

func (b *terminalProgressBar) Start() {
	// render() already running
	if b.shutdownCh != nil {
		return
	}
	fmt.Fprintf(b.out, "%s", CURSOR_HIDE)
	b.shutdownCh = make(chan bool)
	go b.renderLoop()
}

func (b *terminalProgressBar) Err() error {
	var errs []error
	if err := b.spinnerPb.Err(); err != nil {
		errs = append(errs, fmt.Errorf("error on spinner progressbar: %w", err))
	}
	if err := b.msgPb.Err(); err != nil {
		errs = append(errs, fmt.Errorf("error on spinner progressbar: %w", err))
	}
	for _, pb := range b.subLevelPbs {
		if err := pb.Err(); err != nil {
			errs = append(errs, fmt.Errorf("error on spinner progressbar: %w", err))
		}
	}
	return errors.Join(errs...)
}

func (b *terminalProgressBar) Stop() {
	if b.shutdownCh == nil {
		return
	}
	// request shutdown
	b.shutdownCh <- true
	// wait for ack
	select {
	case <-b.shutdownCh:
	// shudown complete
	case <-time.After(1 * time.Second):
		// I cannot think of how this could happen, i.e. why
		// closing would not work but lets be conservative -
		// without a timeout we hang here forever
		logrus.Warnf("no progress channel shutdown after 1sec")
	}
	b.shutdownCh = nil
	// This should never happen but be paranoid, this should
	// never happen but ensure we did not accumulate error while
	// running
	if err := b.Err(); err != nil {
		fmt.Fprintf(b.out, "error from pb.ProgressBar: %v", err)
	}
}
