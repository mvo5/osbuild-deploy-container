package progress

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/osbuild/images/pkg/osbuild"
)

var (
	osStderr io.Writer = os.Stderr
)

// ProgressBar is an interfacs for progress reporting when there is
// an arbitrary amount of sub-progress information (like osbuild)
type ProgressBar interface {
	// SetProgress sets the progress details at the given "level".
	// Levels should start with "0" and increase as the nesting
	// gets deeper.
	SetProgress(level int, msg string, done int, total int) error
	// The high-level message that is displayed in a spinner
	// (e.g. "Building image foo")
	SetPulseMsg(fmt string, args ...interface{})
	// A high level message with the last high level status
	// (e.g. "Started downloading")
	SetMessage(fmt string, args ...interface{})
	Start() error
	Stop() error
}

// New creates a new progressbar based on the requested type
func New(typ string) (ProgressBar, error) {
	switch typ {
	// XXX: autoseelct based on TERM value?
	case "", "plain":
		return NewPlainProgressBar()
	case "term":
		return NewTermProgressBar()
	case "debug":
		return NewDebugProgressBar()
	default:
		return nil, fmt.Errorf("unknown progress type: %q", typ)
	}
}

type termProgressBar struct {
	// the first line is the spinner
	spinnerMsg string
	spinnerPos int
	shutdownCh chan interface{}

	// progress/subprocess
	subProgress []osbuild.Progress

	// last line is always the last message
	msg string

	out io.Writer
}

// NewProgressBar creates a new default pb3 based progressbar suitable for
// most terminals.
func NewTermProgressBar() (ProgressBar, error) {
	ppb := &termProgressBar{
		out: osStderr,
	}
	return ppb, nil
}

func (ppb *termProgressBar) SetProgress(subLevel int, msg string, done int, total int) error {
	// auto-add as needed, requires sublevels to get added in order
	// i.e. adding 0 and then 2 will fail
	switch {
	case subLevel == len(ppb.subProgress):
		ppb.subProgress = append(ppb.subProgress, osbuild.Progress{
			Done:    done,
			Total:   total,
			Message: msg,
		})
	case subLevel > len(ppb.subProgress):
		return fmt.Errorf("subprogress added out of order, have %v sublevels but want level %v", len(ppb.subProgress), subLevel)
	}
	apb := &ppb.subProgress[subLevel]
	apb.Done = done + 1
	apb.Total = total + 1
	apb.Message = msg
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

func (ppb *termProgressBar) SetPulseMsg(msg string, args ...interface{}) {
	ppb.spinnerMsg = shorten(fmt.Sprintf(msg, args...))
}

func (ppb *termProgressBar) SetMessage(msg string, args ...interface{}) {
	ppb.msg = shorten(fmt.Sprintf(msg, args...))
}

var (
	ESC        = "\x1b"
	ERASE_LINE = ESC + "[2K"

	SPINNER = []string{"|", "/", "-", "\\"}
)

func cursorUp(i int) string {
	return fmt.Sprintf("%s[%dA", ESC, i)
}

func (ppb *termProgressBar) render() {
	for {
		select {
		case <-ppb.shutdownCh:
			return
		case <-time.After(200 * time.Millisecond):
			// break
		}
		var renderedLines int
		fmt.Fprintf(ppb.out, "%s[%s] %s\n", ERASE_LINE, SPINNER[ppb.spinnerPos], ppb.spinnerMsg)
		renderedLines++
		for _, prog := range ppb.subProgress {
			fmt.Fprintf(ppb.out, "%s[%d/%d] %s\n", ERASE_LINE, prog.Done, prog.Total, prog.Message)
			renderedLines++
		}
		if ppb.msg != "" {
			fmt.Fprintf(ppb.out, "%sMessage: %s\n", ERASE_LINE, ppb.msg)
			renderedLines++
		}
		ppb.spinnerPos = (ppb.spinnerPos + 1) % len(SPINNER)
		fmt.Fprintf(ppb.out, cursorUp(renderedLines))
	}
}

func (ppb *termProgressBar) Start() error {
	// spinner already running
	if ppb.shutdownCh != nil {
		return nil
	}
	ppb.shutdownCh = make(chan interface{})
	go ppb.render()

	return nil
}

func (ppb *termProgressBar) Stop() (err error) {
	if ppb.shutdownCh == nil {
		return nil
	}
	close(ppb.shutdownCh)
	ppb.shutdownCh = nil
	return nil
}

type plainProgressBar struct {
	w io.Writer
}

// NewPlainProgressBar starts a new "plain" progressbar that will just
// prints message but does not show any progress.
func NewPlainProgressBar() (ProgressBar, error) {
	np := &plainProgressBar{w: osStderr}
	return np, nil
}

func (np *plainProgressBar) SetPulseMsg(msg string, args ...interface{}) {
	fmt.Fprintf(np.w, msg, args...)
}

func (np *plainProgressBar) SetMessage(msg string, args ...interface{}) {
	fmt.Fprintf(np.w, msg, args...)
}

func (np *plainProgressBar) Start() (err error) {
	return nil
}

func (np *plainProgressBar) Stop() (err error) {
	return nil
}

func (np *plainProgressBar) SetProgress(subLevel int, msg string, done int, total int) error {
	return nil
}

type debugProgressBar struct {
	w io.Writer
}

// NewDebugProgressBar will create a progressbar aimed to debug the
// lower level osbuild/images message. It will never clear the screen
// so "glitches/weird" messages from the lower-layers can be inspected
// easier.
func NewDebugProgressBar() (ProgressBar, error) {
	np := &debugProgressBar{w: osStderr}
	return np, nil
}

func (np *debugProgressBar) SetPulseMsg(msg string, args ...interface{}) {
	fmt.Fprintf(np.w, "pulse: ")
	fmt.Fprintf(np.w, msg, args...)
	fmt.Fprintf(np.w, "\n")
}

func (np *debugProgressBar) SetMessage(msg string, args ...interface{}) {
	fmt.Fprintf(np.w, "msg: ")
	fmt.Fprintf(np.w, msg, args...)
	fmt.Fprintf(np.w, "\n")
}

func (np *debugProgressBar) Start() (err error) {
	fmt.Fprintf(np.w, "Start progressbar\n")
	return nil
}

func (np *debugProgressBar) Stop() (err error) {
	fmt.Fprintf(np.w, "Stop progressbar\n")
	return nil
}

func (np *debugProgressBar) SetProgress(subLevel int, msg string, done int, total int) error {
	fmt.Fprintf(np.w, "%s[%v / %v] %s", strings.Repeat("  ", subLevel), done, total, msg)
	fmt.Fprintf(np.w, "\n")
	return nil
}

// XXX: merge variant back into images/pkg/osbuild/osbuild-exec.go
func RunOSBuild(pb ProgressBar, manifest []byte, store, outputDirectory string, exports, extraEnv []string) error {
	switch pb.(type) {
	case *termProgressBar, *debugProgressBar:
		return runOSBuildNew(pb, manifest, store, outputDirectory, exports, extraEnv)
	default:
		return runOSBuildOld(pb, manifest, store, outputDirectory, exports, extraEnv)
	}
}

func runOSBuildOld(pb ProgressBar, manifest []byte, store, outputDirectory string, exports, extraEnv []string) error {
	_, err := osbuild.RunOSBuild(manifest, store, outputDirectory, exports, nil, extraEnv, false, os.Stderr)
	return err
}

func runOSBuildNew(pb ProgressBar, manifest []byte, store, outputDirectory string, exports, extraEnv []string) error {
	rp, wp, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("cannot create pipe for osbuild: %w", err)
	}
	defer rp.Close()
	defer wp.Close()

	cmd := exec.Command(
		"osbuild",
		"--store", store,
		"--output-directory", outputDirectory,
		"--monitor=JSONSeqMonitor",
		"--monitor-fd=3",
		"-",
	)
	for _, export := range exports {
		cmd.Args = append(cmd.Args, "--export", export)
	}

	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdin = bytes.NewBuffer(manifest)
	cmd.Stderr = os.Stderr
	// we could use "--json" here and would get the build-result
	// exported here
	cmd.Stdout = nil
	cmd.ExtraFiles = []*os.File{wp}

	osbuildStatus := osbuild.NewStatusScanner(rp)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error starting osbuild: %v", err)
	}
	wp.Close()

	var tracesMsgs []string
	for {
		st, err := osbuildStatus.Status()
		if err != nil {
			return fmt.Errorf("error reading osbuild status: %w", err)
		}
		if st == nil {
			break
		}
		i := 0
		for p := st.Progress; p != nil; p = p.SubProgress {
			// XXX: osbuild gives us bad progress messages
			if err := pb.SetProgress(i, p.Message, p.Done, p.Total); err != nil {
				logrus.Warnf("cannot set progress: %v", err)
			}
			i++
		}
		// keep the messages/traces for better error reporting
		if st.Message != "" {
			tracesMsgs = append(tracesMsgs, st.Message)
		}
		if st.Trace != "" {
			tracesMsgs = append(tracesMsgs, st.Trace)
		}
		// forward to user
		if st.Message != "" {
			pb.SetMessage(st.Message)
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("error running osbuild: %w\nLog:\n%s", err, strings.Join(tracesMsgs, "\n"))
	}

	return nil
}
