package main

import (
	"io"

	"github.com/osbuild/images/pkg/osbuild"
)

var (
	CanChownInPath = canChownInPath
	RootCmd        = rootCmd
	Run            = run
)

func MockOsGetuid(new func() int) (restore func()) {
	saved := osGetuid
	osGetuid = new
	return func() {
		osGetuid = saved
	}
}

func MockOsbuildRunOSBuild(f func(manifest []byte, store string, outputDirectory string, exports []string, checkpoints []string, extraEnv []string, result bool, errorWriter io.Writer) (*osbuild.Result, error)) (restore func()) {
	saved := osbuildRunOSBuild
	osbuildRunOSBuild = f
	return func() {
		osbuildRunOSBuild = saved
	}
}
