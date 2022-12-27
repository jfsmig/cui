// Copyright (c) 2022-2023 Jean-Francois Smigielski
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jfsmig/cui"
)

func main() {
	if err := cui.Monitor(&directorySource{}, "/var/log"); err != nil {
		log.Fatalln(err)
	}
}

type directorySource struct{}

type fileItem struct {
	Path  string      `json:"path"`
	Size  int64       `json:"size"`
	Mode  fs.FileMode `json:"mode"`
	CTime time.Time   `json:"ctime"`
}

func (fi *fileItem) GetPrimaryKey() string { return "path" }

func (fi *fileItem) GetKeys() []string { return []string{"path", "size", "mode", "ctime"} }

func (fi *fileItem) GetValue(k string) string {
	switch k {
	case "path":
		return fi.Path
	case "size":
		return strconv.FormatInt(fi.Size, 10)
	case "mode":
		return fi.Mode.String()
	case "ctime":
		return fi.CTime.String()
	default:
		return "-"
	}
}

func (fi *fileItem) GetDetail() string {
	var builder strings.Builder
	encoder := json.NewEncoder(&builder)
	encoder.SetIndent("", " ")
	encoder.Encode(*fi)
	return builder.String()
}

func (dl *directorySource) FetchAll(query string) ([]cui.MonitoredItem, error) {
	var out []cui.MonitoredItem
	if query == "" {
		return out, errors.New("Empty query")
	}
	if !filepath.IsAbs(query) {
		return out, errors.New("Relative query path")
	}

	entries, err := os.ReadDir(query)
	for _, entry := range entries {
		info, _ := entry.Info()
		out = append(out, &fileItem{
			entry.Name(),
			info.Size(),
			info.Mode(),
			info.ModTime(),
		})
	}
	return out, err
}
