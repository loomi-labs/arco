package util

import (
	"io"
	"io/fs"
	"strings"
)

// CustomFS is a file system where all files are read with a prefix and suffix.
type CustomFS struct {
	FS     fs.FS
	Prefix string
	Suffix string
}

func (cfs *CustomFS) Open(name string) (fs.File, error) {
	file, err := cfs.FS.Open(name)
	if err != nil {
		return nil, err
	}
	return &CustomFile{
		File:   file,
		Prefix: cfs.Prefix,
		Suffix: cfs.Suffix,
	}, nil
}

type CustomFile struct {
	fs.File
	Prefix string
	Suffix string
	reader io.Reader
}

func (cf *CustomFile) Read(p []byte) (int, error) {
	if cf.reader == nil {
		content, err := io.ReadAll(cf.File)
		if err != nil {
			return 0, err
		}
		modifiedContent := cf.Prefix + string(content) + cf.Suffix
		cf.reader = strings.NewReader(modifiedContent)
	}
	return cf.reader.Read(p)
}

func (cf *CustomFile) ReadDir(n int) ([]fs.DirEntry, error) {
	dirFile, ok := cf.File.(fs.ReadDirFile)
	if !ok {
		return nil, &fs.PathError{Op: "ReadDir", Path: "", Err: fs.ErrInvalid}
	}
	return dirFile.ReadDir(n)
}
