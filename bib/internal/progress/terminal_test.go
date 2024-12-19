package progress_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/bootc-image-builder/bib/internal/progress"
)

func TestTermProgress(t *testing.T) {
	var buf bytes.Buffer
	restore := progress.MockOsStderr(&buf)
	defer restore()

	pbar, err := progress.NewTerminalProgressBar()
	assert.NoError(t, err)

	pbar.Start()
	pbar.SetPulseMsgf("pulse-msg")
	pbar.SetMessagef("some-message")
	err = pbar.SetProgress(0, "set-progress-msg", 0, 5)
	assert.NoError(t, err)
	pbar.Stop()
	assert.NoError(t, pbar.(*progress.TerminalProgressBar).Err())

	assert.Contains(t, buf.String(), "[1 / 6] set-progress-msg")
	assert.Contains(t, buf.String(), "[|] pulse-msg\n")
	assert.Contains(t, buf.String(), "Message: some-message\n")
	// check shutdown
	assert.Contains(t, buf.String(), progress.CURSOR_SHOW)
}
