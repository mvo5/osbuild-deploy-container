package progress_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/bootc-image-builder/bib/internal/progress"
)

func TestPlainProgress(t *testing.T) {
	var buf bytes.Buffer
	restore := progress.MockOsStderr(&buf)
	defer restore()

	// plain progress never generates progress output
	pbar, err := progress.NewPlainProgressBar()
	assert.NoError(t, err)
	err = pbar.SetProgress(0, "set-progress", 1, 100)
	assert.NoError(t, err)
	assert.Equal(t, "", buf.String())

	// but it shows the messages
	pbar.SetPulseMsgf("pulse")
	assert.Equal(t, "pulse\n", buf.String())
	buf.Reset()

	pbar.SetMessagef("message")
	assert.Equal(t, "message\n", buf.String())
	buf.Reset()

	pbar.Start()
	assert.Equal(t, "", buf.String())
	pbar.Stop()
	assert.Equal(t, "", buf.String())
}
