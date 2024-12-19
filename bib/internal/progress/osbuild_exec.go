package progress

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/osbuild/images/pkg/osbuild"
)

// XXX: merge variant back into images/pkg/osbuild/osbuild-exec.go
func RunOSBuild(pb ProgressBar, manifest []byte, store, outputDirectory string, exports, extraEnv []string) error {
	// To keep maximum compatibility keep the old behavior to run osbuild
	// directly and show all messages unless we have a "real" progress bar.
	//
	// This should ensure that e.g. "podman bootc" keeps working as it
	// is currently expecting the raw osbuild output. Once we double
	// checked with them we can remove the runOSBuildNoProgress() and
	// just run with the new runOSBuildWithProgress() helper.
	switch pb.(type) {
	case *terminalProgressBar, *debugProgressBar:
		return runOSBuildWithProgress(pb, manifest, store, outputDirectory, exports, extraEnv)
	default:
		return runOSBuildNoProgress(pb, manifest, store, outputDirectory, exports, extraEnv)
	}
}

func runOSBuildNoProgress(pb ProgressBar, manifest []byte, store, outputDirectory string, exports, extraEnv []string) error {
	_, err := osbuild.RunOSBuild(manifest, store, outputDirectory, exports, nil, extraEnv, false, os.Stderr)
	return err
}

func runOSBuildWithProgress(pb ProgressBar, manifest []byte, store, outputDirectory string, exports, extraEnv []string) error {
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
			pb.SetMessagef(st.Message)
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("error running osbuild: %w\nLog:\n%s", err, strings.Join(tracesMsgs, "\n"))
	}

	return nil
}
