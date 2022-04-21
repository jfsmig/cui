// Copyright (c) 2022 Jean-Francois Smigielski
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jfsmig/cui"
	"log"
	"net/rpc"
	"strings"
	"time"
)

func main() {
	if err := cui.Monitor(&extenTitanObjectsSource{}, "127.0.0.1"); err != nil {
		log.Fatalln(err)
	}
}

type extenTitanObjectsSource struct{}

type ObjectItem map[string]interface{}

func (fi ObjectItem) GetPrimaryKey() string { return "key" }

func (fi ObjectItem) GetKeys() []string {
	out := make([]string, 0)
	for k, _ := range fi {
		out = append(out, k)
	}
	return out
}

func (fi ObjectItem) GetValue(k string) string { return fmt.Sprint(fi[k]) }

var builder = strings.Builder{}
var encoder = json.NewEncoder(&builder)

func (fi ObjectItem) GetDetail() string {
	builder.Reset()
	encoder.SetIndent("", " ")
	encoder.Encode(fi)
	return builder.String()
}

type NetRpcBaseRequest struct {
	Timeout time.Duration
	Fields  map[string]interface{}
}

type Empty struct {
	NetRpcBaseRequest
}

type TitanObjectsReply struct {
	Payload []byte
}

func (dl *extenTitanObjectsSource) FetchAll(query string) ([]cui.MonitoredItem, error) {
	var out []cui.MonitoredItem
	if query == "" {
		return out, errors.New("Empty query")
	}

	client, err := rpc.DialHTTP("tcp", query + ":2233")
	if err != nil {
		return out, fmt.Errorf("can't connect to systest agent: %w", err)
	}

	defer func() { _ = client.Close() }()

	reply := TitanObjectsReply{}
	err = client.Call("Sys.TitanObjects", Empty{}, &reply)
	if err != nil {
		return out, fmt.Errorf("Failed to query systest-agent: %w", err)
	}

	decoded := make([]ObjectItem, 0)
	decoder := json.NewDecoder(bytes.NewReader(reply.Payload))
	err = decoder.Decode(&decoded)
	if err != nil {
		return out, fmt.Errorf("Format error: not an array of maps: %w", err)
	}
	for _, obj := range decoded {
		out = append(out, obj)
	}
	return out, nil
}