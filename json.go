// Copyright 2019 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xorm

import (
	stdjson "encoding/json"
)

// JSONHandler represents an interface to handle json data
type JSONHandler interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
}

var (
	// DefaultJSONHandler default json handler
	DefaultJSONHandler JSONHandler = StdJSON{}
)

// StdJSON implements JSONInterface via encoding/json
type StdJSON struct{}

// Marshal implements JSONInterface
func (StdJSON) Marshal(v interface{}) ([]byte, error) {
	return stdjson.Marshal(v)
}

// Unmarshal implements JSONInterface
func (StdJSON) Unmarshal(data []byte, v interface{}) error {
	return stdjson.Unmarshal(data, v)
}
