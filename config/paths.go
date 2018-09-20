package config

import (
	"path/filepath"
	"runtime"
)

func GetProjectSourceRootPath() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..")
}

func GetCurrentSourceFileDirPath() string {
	_, filename, _, _ := runtime.Caller(1)
	return filepath.Dir(filename)
}

func GetProjectSourceTmpPath() string {
	return filepath.Join(GetProjectSourceRootPath(), "_tmp")
}
