package progress_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/bootc-image-builder/bib/internal/progress"
)

func TestProgressNew(t *testing.T) {
	for _, tc := range []struct {
		typ         string
		expected    interface{}
		expectedErr string
	}{
		{"term", &progress.TerminalProgressBar{}, ""},
		{"debug", &progress.DebugProgressBar{}, ""},
		{"plain", &progress.PlainProgressBar{}, ""},
		// unknown progress type
		{"bad", nil, `unknown progress type: "bad"`},
	} {
		pb, err := progress.New(tc.typ)
		if tc.expectedErr == "" {
			assert.NoError(t, err)
			assert.Equal(t, reflect.TypeOf(pb), reflect.TypeOf(tc.expected), fmt.Sprintf("[%v] %T not the expected %T", tc.typ, pb, tc.expected))
		} else {
			assert.EqualError(t, err, tc.expectedErr)
		}
	}
}
