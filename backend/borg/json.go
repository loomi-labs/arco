package borg

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// JSONTime wraps time.Time to provide custom unmashalling of Unix timestamps.
type JSONTime time.Time

// UnmarshalJSON converts a float64 Unix timestamp to a time.Time object.
func (jt *JSONTime) UnmarshalJSON(b []byte) error {
	var timestamp *float64

	// Unmarshal the JSON number into a *float64.
	if err := json.Unmarshal(b, &timestamp); err != nil {
		return err
	}

	if timestamp != nil {
		sec, dec := math.Modf(*timestamp)
		t := time.Unix(int64(sec), int64(dec*1e9))
		*jt = JSONTime(t)
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
	Time             JSONTime `json:"time,omitempty"`
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
	Time      JSONTime `json:"time,omitempty"`
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
	Time      JSONTime `json:"time,omitempty"`
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
	Time      JSONTime `json:"time"`
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
