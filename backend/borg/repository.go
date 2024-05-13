package borg

import "github.com/google/uuid"

type Repo struct {
	Id          string      `json:"id"`
	Name        string      `json:"name"`
	Prefix      string      `json:"prefix"`
	Directories []Directory `json:"directories"`
}

func NewRepo(name, prefix string, directories []string) *Repo {
	var dirs []Directory
	for _, dir := range directories {
		dirs = append(dirs, Directory{
			Path:    dir,
			IsAdded: false,
		})
	}
	return &Repo{
		Id:          uuid.New().String(),
		Name:        name,
		Prefix:      prefix,
		Directories: dirs,
	}
}

type Directory struct {
	Path    string `json:"path"`
	IsAdded bool   `json:"isAdded"`
}

func (r *Repo) AddDirectory(newDir Directory) {
	// Add directory to the list of directories
	// If it already exists, set IsAdded to true
	for i, dir := range r.Directories {
		if dir.Path == newDir.Path {
			r.Directories[i].IsAdded = true
			return
		}
	}
	r.Directories = append(r.Directories, newDir)
}
