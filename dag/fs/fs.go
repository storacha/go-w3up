package fs

import (
	"io/fs"
	"os"
)

type Opener interface {
	Open() (fs.File, error)
}

type OsEntry struct {
	Path     string
	FileInfo fs.FileInfo
}

func (e *OsEntry) Open() (fs.File, error) {
	return os.Open(e.Path)
}

func (e *OsEntry) Name() string {
	return e.FileInfo.Name()
}

func (e *OsEntry) IsDir() bool {
	return e.FileInfo.IsDir()
}

func (e *OsEntry) Type() fs.FileMode {
	return e.FileInfo.Mode().Type()
}

func (e *OsEntry) Info() (fs.FileInfo, error) {
	return e.FileInfo, nil
}

type OsFs struct {
	Root string
}

func (f *OsFs) Open(name string) (fs.File, error) {
	return os.Open(name)
}