package scpw

import (
	"io/fs"
	"os"
)

type Resource struct {
	fs.FileInfo
	Path string
}

func NewResource(filePath string) (*Resource, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	return &Resource{stat, filePath}, nil
}
