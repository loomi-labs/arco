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
		// Create a reader that concatenates the prefix, file content, and suffix
		cf.reader = io.MultiReader(
			strings.NewReader(cf.Prefix),
			cf.File,
			strings.NewReader(cf.Suffix),
		)
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
