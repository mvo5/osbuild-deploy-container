package progress_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/bootc-image-builder/bib/internal/progress"
)

func TestDebugProgress(t *testing.T) {
	var buf bytes.Buffer
	restore := progress.MockOsStderr(&buf)
	defer restore()

	pbar, err := progress.NewDebugProgressBar()
	assert.NoError(t, err)
	err = pbar.SetProgress(0, "set-progress-msg", 1, 100)
	assert.NoError(t, err)
	assert.Equal(t, "[1 / 100] set-progress-msg\n", buf.String())
	buf.Reset()

	pbar.SetPulseMsgf("pulse-msg")
	assert.Equal(t, "pulse: pulse-msg\n", buf.String())
	buf.Reset()

	pbar.SetMessagef("some-message")
	assert.Equal(t, "msg: some-message\n", buf.String())
	buf.Reset()

	pbar.Start()
	assert.Equal(t, "Start progressbar\n", buf.String())
	buf.Reset()

	pbar.Stop()
	assert.Equal(t, "Stop progressbar\n", buf.String())
	buf.Reset()
}
