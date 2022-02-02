// Copyright (c) 2022 Jean-Francois Smigielski
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"errors"
	"github.com/jfsmig/cui/monitor"
	"log"
	"os"
	"path/filepath"
)

type dirListing struct{}

type fileItem struct {
	filename string
}

func (fi *fileItem) PrimaryKey() string       { return fi.filename }
func (fi *fileItem) GetKeys() []string        { return []string{} }
func (fi *fileItem) GetValue(k string) string { return "" }

func (dl *dirListing) FetchAll(query string) ([]monitor.MonitoredItem, error) {
	var out []monitor.MonitoredItem
	if query == "" {
		return out, errors.New("Empty query")
	}
	if !filepath.IsAbs(query) {
		return out, errors.New("Relative query path")
	}

	entries, err := os.ReadDir(query)
	for _, entry := range entries {
		out = append(out, &fileItem{entry.Name()})
	}
	return out, err
}

func main() {
	if err := monitor.Monitor(&dirListing{}); err != nil {
		log.Fatalln(err)
	}
}
