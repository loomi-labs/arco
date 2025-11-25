package types

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// UnixTime wraps time.Time to provide custom unmarshalling of Unix timestamps.
type UnixTime time.Time

// UnmarshalJSON converts a float64 Unix timestamp to a time.Time object.
func (jt *UnixTime) UnmarshalJSON(b []byte) error {
	var timestamp *float64

	// Unmarshal the JSON number into a *float64.
	if err := json.Unmarshal(b, &timestamp); err != nil {
		return err
	}

	if timestamp != nil {
		sec, dec := math.Modf(*timestamp)
		t := time.Unix(int64(sec), int64(dec*1e9))
		*jt = UnixTime(t)
	}

	return nil
}

// StringTime wraps time.Time to provide custom unmarshalling of string timestamps.
type StringTime time.Time

// UnmarshalJSON converts a string timestamp to a time.Time object.
func (st *StringTime) UnmarshalJSON(b []byte) error {
	var timestamp *string

	// Unmarshal the JSON string into a *string.
	if err := json.Unmarshal(b, &timestamp); err != nil {
		return err
	}

	if timestamp != nil {
		t, err := time.Parse("2006-01-02T15:04:05.000000", *timestamp)
		if err != nil {
			return err
		}
		*st = StringTime(t)
	}

	return nil
}

type Type struct {
	Type string `json:"type"`
}

type JSONType string

const (
	ArchiveProgressType JSONType = "archive_progress"
	ProgressMessageType JSONType = "progress_message"
	ProgressPercentType JSONType = "progress_percent"
	FileStatusType      JSONType = "file_status"
	LogMessageType      JSONType = "log_message"
)

type ArchiveProgress struct {
	OriginalSize     int64    `json:"original_size,omitempty"`
	CompressedSize   int64    `json:"compressed_size,omitempty"`
	DeduplicatedSize int64    `json:"deduplicated_size,omitempty"`
	NFiles           int      `json:"nfiles,omitempty"`
	Path             string   `json:"path,omitempty"`
	Time             UnixTime `json:"time,omitempty"`
	Finished         bool     `json:"finished,omitempty"`
}

func (ap ArchiveProgress) String() string {
	return fmt.Sprintf("ArchiveProgress{OriginalSize: %v, CompressedSize: %v, DeduplicatedSize: %v, NFiles: %v, Path: %v, Time: %v, Finished: %v}",
		ap.OriginalSize, ap.CompressedSize, ap.DeduplicatedSize, ap.NFiles, ap.Path, ap.Time, ap.Finished)
}

type ProgressMessage struct {
	Operation int      `json:"operation"`
	MsgID     string   `json:"msgid,omitempty"`
	Finished  bool     `json:"finished"`
	Message   string   `json:"message,omitempty"`
	Time      UnixTime `json:"time,omitempty"`
}

func (pm ProgressMessage) String() string {
	return fmt.Sprintf("ProgressMessage{Operation: %v, MsgID: %v, Finished: %v, Message: %v, Time: %v}",
		pm.Operation, pm.MsgID, pm.Finished, pm.Message, pm.Time)
}

type ProgressPercent struct {
	Operation int      `json:"operation"`
	MsgID     string   `json:"msgid,omitempty"`
	Finished  bool     `json:"finished"`
	Message   string   `json:"message,omitempty"`
	Current   int      `json:"current,omitempty"`
	Info      []string `json:"info,omitempty"`
	Total     int      `json:"total,omitempty"`
	Time      UnixTime `json:"time,omitempty"`
}

func (pp ProgressPercent) String() string {
	return fmt.Sprintf("ProgressPercent{Operation: %v, MsgID: %v, Finished: %v, Message: %v, Current: %v, Info: %v, Total: %v, Time: %v}",
		pp.Operation, pp.MsgID, pp.Finished, pp.Message, pp.Current, pp.Info, pp.Total, pp.Time)
}

type FileStatus struct {
	Status string `json:"status"`
	Path   string `json:"path"`
}

func (fs FileStatus) String() string {
	return fmt.Sprintf("FileStatus{Status: %v, Path: %v}", fs.Status, fs.Path)
}

type LogMessage struct {
	Time      UnixTime `json:"time"`
	LevelName string   `json:"levelname"`
	Name      string   `json:"name"`
	Message   string   `json:"message"`
	MsgID     string   `json:"msgid,omitempty"`
}

func (lm LogMessage) String() string {
	return fmt.Sprintf("LogMessage{Time: %v, LevelName: %v, Name: %v, Message: %v, MsgID: %v}",
		lm.Time, lm.LevelName, lm.Name, lm.Message, lm.MsgID)
}

func decodeStreamedJSON(scanner *bufio.Scanner, ch chan<- interface{}) {
	for scanner.Scan() {
		data := scanner.Text()

		// Let's try to find out what type of message we have
		var typeMsg Type
		decoder := json.NewDecoder(strings.NewReader(data))
		err := decoder.Decode(&typeMsg)
		if err != nil {
			// Continue if we can't decode the JSON
			continue
		}

		switch JSONType(typeMsg.Type) {
		case ArchiveProgressType:
			var archiveProgress ArchiveProgress
			decoder = json.NewDecoder(strings.NewReader(data))
			err = decoder.Decode(&archiveProgress)
			if err != nil {
				continue
			}
			ch <- archiveProgress
		case ProgressMessageType:
			var progressMessage ProgressMessage
			decoder = json.NewDecoder(strings.NewReader(data))
			err = decoder.Decode(&progressMessage)
			if err != nil {
				continue
			}
			ch <- progressMessage
		case ProgressPercentType:
			var progressPercent ProgressPercent
			decoder = json.NewDecoder(strings.NewReader(data))
			err = decoder.Decode(&progressPercent)
			if err != nil {
				continue
			}
			ch <- progressPercent
		case FileStatusType:
			var fileStatus FileStatus
			decoder = json.NewDecoder(strings.NewReader(data))
			err = decoder.Decode(&fileStatus)
			if err != nil {
				continue
			}
			ch <- fileStatus
		case LogMessageType:
			var logMessage LogMessage
			decoder = json.NewDecoder(strings.NewReader(data))
			err = decoder.Decode(&logMessage)
			if err != nil {
				continue
			}
			ch <- logMessage
		}
	}
}

type ArchiveList struct {
	Archive  string     `json:"archive"`
	Barchive string     `json:"barchive"`
	ID       string     `json:"id"`
	Name     string     `json:"name"`
	Start    StringTime `json:"start"`
	End      StringTime `json:"end"`
}

type Limits struct {
	MaxArchiveSize float64 `json:"max_archive_size"`
}

type ArchiveStats struct {
	CompressedSize   int `json:"compressed_size"`
	DeduplicatedSize int `json:"deduplicated_size"`
	NFiles           int `json:"nfiles"`
	OriginalSize     int `json:"original_size"`
}

type ArchiveInfo struct {
	ChunkerParams []interface{} `json:"chunker_params"`
	CommandLine   []string      `json:"command_line"`
	Comment       string        `json:"comment"`
	Duration      UnixTime      `json:"duration"`
	End           StringTime    `json:"end"`
	Hostname      string        `json:"hostname"`
	ID            string        `json:"id"`
	Limits        Limits        `json:"limits"`
	Name          string        `json:"name"`
	Start         StringTime    `json:"start"`
	Stats         ArchiveStats  `json:"stats"`
	Username      string        `json:"username"`
}

type Encryption struct {
	Mode string `json:"mode"`
}

type Repository struct {
	ID           string `json:"id"`
	LastModified string `json:"last_modified"`
	Location     string `json:"location"`
}

type Stats struct {
	TotalChunks       int `json:"total_chunks"`        // Number of chunks
	TotalSize         int `json:"total_size"`          // Total uncompressed size of all chunks multiplied with their reference counts
	TotalCSize        int `json:"total_csize"`         // Total compressed and encrypted size of all chunks multiplied with their reference counts
	TotalUniqueChunks int `json:"total_unique_chunks"` // Number of unique chunks
	UniqueSize        int `json:"unique_size"`         // Uncompressed size of all chunks
	UniqueCSize       int `json:"unique_csize"`        // Compressed and encrypted size of all chunks
}

type CheckResult struct {
	Status    *Status      // Command execution status
	ErrorLogs []LogMessage // Captured ERROR messages only
}

type Cache struct {
	Path  string `json:"path"`
	Stats Stats  `json:"stats"`
}

type InfoResponse struct {
	Archives    []ArchiveInfo `json:"archives"`
	Cache       Cache         `json:"cache"`
	Encryption  Encryption    `json:"encryption"`
	Repository  Repository    `json:"repository"`
	SecurityDir string        `json:"security_dir"`
}

type BackupProgress struct {
	TotalFiles     int `json:"totalFiles"`
	ProcessedFiles int `json:"processedFiles"`
}

type ListResponse struct {
	Archives   []ArchiveList `json:"archives"`
	Encryption Encryption    `json:"encryption"`
	Repository Repository    `json:"repository"`
}

type PruneResult struct {
	IsDryRun      bool
	PruneArchives []*PruneArchive
	KeepArchives  []*KeepArchive
}

type PruneArchive struct {
	Name string
}

type KeepArchive struct {
	Name   string
	Reason string
}
