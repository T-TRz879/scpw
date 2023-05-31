package scpw

import (
	"github.com/google/uuid"
	"io/fs"
	"math/rand"
	"os"
	"path"
)

const (
	TmpRoot = "/tmp/trz"
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

func NewTestResource(dir bool) *Resource {
	var err error
	_, err = os.Stat(TmpRoot)
	if err != nil {
		err := os.Mkdir(TmpRoot, os.FileMode(0777))
		if err != nil {
			panic(err)
		}
	}
	entryPath := path.Join(TmpRoot, uuid.NewString())
	_, err = os.Stat(entryPath)
	if err == nil {
		err = os.Remove(entryPath)
		if err != nil {
			panic(err)
		}
	}
	if dir {
		err = os.Mkdir(entryPath, os.FileMode(0777))
		if err != nil {
			panic(err)
		}
	} else {
		_, err = os.Create(entryPath)
		if err != nil {
			panic(err)
		}
		b := make([]byte, 1000)
		rand.Read(b)
		err := os.WriteFile(entryPath, b, os.FileMode(0777))
		if err != nil {
			panic(err)
		}
	}
	stat, err := os.Stat(entryPath)
	if err != nil {
		panic(err)
	}
	return &Resource{stat, entryPath}
}
