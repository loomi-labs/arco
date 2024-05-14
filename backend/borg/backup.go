package borg

import "github.com/google/uuid"

type BackupSet struct {
	Id          string      `json:"id"`
	Name        string      `json:"name"`
	Prefix      string      `json:"prefix"`
	Directories []Directory `json:"directories"`
	Schedule    Schedule    `json:"schedule"`
}

type Directory struct {
	Path    string `json:"path"`
	IsAdded bool   `json:"isAdded"`
}

type Schedule struct {
	HasPeriodicBackups bool   `json:"hasPeriodicBackups"`
	PeriodicBackupTime string `json:"periodicBackupTime"`
}

func NewBackupSet(name, prefix string, directories []string) *BackupSet {
	var dirs []Directory
	for _, dir := range directories {
		dirs = append(dirs, Directory{
			Path:    dir,
			IsAdded: false,
		})
	}
	return &BackupSet{
		Id:          uuid.New().String(),
		Name:        name,
		Prefix:      prefix,
		Directories: dirs,
		Schedule: Schedule{
			HasPeriodicBackups: true,
			PeriodicBackupTime: "00:00",
		},
	}
}

func (b *BackupSet) Save() {

}

func (b *BackupSet) AddDirectory(newDir Directory) {
	// Add directory to the list of directories
	// If it already exists, set IsAdded to true
	for i, dir := range b.Directories {
		if dir.Path == newDir.Path {
			b.Directories[i].IsAdded = true
			return
		}
	}
	b.Directories = append(b.Directories, newDir)
}
