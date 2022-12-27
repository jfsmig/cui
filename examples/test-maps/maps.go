// Copyright (c) 2022-2023 Jean-Francois Smigielski
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log"
	"math/rand"
	"strconv"
	"strings"

	"github.com/jfsmig/cui"
)

func main() {
	for i := 0; i < 8192; i++ {
		items = append(items, generateRandomItem())
	}

	if err := cui.Monitor(&staticMapsSource{}, "/var/log"); err != nil {
		log.Fatalln(err)
	}
}

type staticMapsSource struct{}

type mapItem map[string]string

func (mi *mapItem) GetPrimaryKey() string { return mi.GetKeys()[0] }

func (mi *mapItem) GetKeys() []string {
	out := make([]string, 0)
	for k, _ := range *mi {
		out = append(out, k)
	}
	return out
}

func (mi *mapItem) GetValue(k string) string { return (*mi)[k] }

func (mi *mapItem) GetDetail() string {
	var builder strings.Builder
	encoder := json.NewEncoder(&builder)
	encoder.SetIndent("", " ")
	encoder.Encode(*mi)
	return builder.String()
}

func (dl *staticMapsSource) FetchAll(query string) ([]cui.MonitoredItem, error) {
	var out []cui.MonitoredItem
	if query == "" {
		return out, errors.New("Empty query")
	}

	h := md5.New()
	io.WriteString(h, query)
	checksum := h.Sum(nil)
	u0, _ := binary.ReadUvarint(bytes.NewReader(checksum[:8]))
	u1, _ := binary.ReadUvarint(bytes.NewReader(checksum[8:]))
	offset := (u0 ^ u1)

	max := 1024
	for i := 0; i < max; i++ {
		item := items[offset%uint64(len(items))]
		offset++
		out = append(out, item)
	}
	return out, nil
}

var keys = []string{
	"address",
	"critical_warning",
	"temperature",
	"spare",
	"spare_threshold",
	"lifetime_used",
	"units_read_hi",
	"units_read_lo",
	"units_written_hi",
	"units_written_lo",
	"host_read_cmds_hi",
	"host_read_cmds",
	"host_write_cmds_hi",
	"host_write_cmds_lo",
	"busy_mins_hi",
	"busy_mins_lo",
	"power_cycles_hi",
	"power_cycles_lo",
	"uptime_hi",
	"uptime_lo",
	"unsafe_shutdowns_hi",
	"unsafe_shutdowns_lo",
	"media_errors_hi",
	"media_errors_lo",
	"error_log_count_hi",
	"error_log_count_lo",
	"sav_arb",
	"def_arb",
	"cur_arb",
	"cap_arb",
	"sav_pwr_mgmt",
	"def_pwr_mgmt",
	"cur_pwr_mgmt",
	"cap_pwr_mgmt",
	"sav_temp_thresh",
	"def_temp_thresh",
	"cur_temp_thresh",
	"cap_temp_thresh",
	"sav_err_recov",
	"def_err_recov",
}

var items []cui.MonitoredItem

func generateRandomItem() cui.MonitoredItem {
	var item mapItem = make(map[string]string)
	// Poll 20 random keys among the 40
	for _, kIndex := range rand.Perm(len(keys))[:20] {
		item[keys[kIndex]] = strconv.FormatUint(rand.Uint64(), 16)
	}
	return &item
}
