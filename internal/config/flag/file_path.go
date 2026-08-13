package flag

import (
	"flag"
	"fmt"
)

type FilePath struct {
	Path string
}

func NewFilePathFlag(flagName string, flagUsage string, defaultURL string) *FilePath {
	filePath := FilePath{
		Path: defaultURL,
	}
	flag.Var(&filePath, flagName, flagUsage)
	return &filePath
}

func (filePath *FilePath) String() string {
	return filePath.Path
}

func (filePath *FilePath) Set(value string) error {
	if value == "" {
		return fmt.Errorf("file path should not be empty")
	}
	filePath.Path = value
	return nil
}
