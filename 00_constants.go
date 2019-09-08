package main

import (
	"path"
	"path/filepath"
	"runtime"
)

const (
	TEST_PROJECT_BUNDLE = "projectx.fcpbundle"
	TEST_PROJECT_UUID   = "3B60E5BE-C5CA-4D1B-A5C9-55E0F819A286"
)

var (
	buildDir, testFCPXBundlePath = func() (string, string) {
		_, b, _, _ := runtime.Caller(0)
		rundir := filepath.Dir(b)
		project_path := path.Join(rundir, TEST_PROJECT_BUNDLE)
		return rundir, project_path
	}()
)
