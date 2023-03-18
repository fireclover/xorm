// Copyright 2021 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package convert

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInterface2Interface_RawBytesWithSameAddr(t *testing.T) {

	kases := []string{
		"testOneTwoThird",
		"testTwo",
		"testThird",
	}

	targetResult := make(map[string]interface{}, len(kases))
	//  sql.Rows nextLocked() used same addr []driver.Value
	var rawBytes sql.RawBytes = make([]byte, 0, 1000)
	sameRawBytes := &rawBytes
	for _, val := range kases {
		t.Run(val, func(t *testing.T) {
			sameBytes := *sameRawBytes
			sameBytes = sameBytes[:0]
			for _, b := range []byte(val) {
				sameBytes = append(sameBytes, b)
			}
			*sameRawBytes = sameBytes

			target, err := Interface2Interface(time.UTC, sameRawBytes)
			assert.NoError(t, err)
			targetResult[val] = target
		})
	}

	for val, target := range targetResult {
		t.Run(val, func(t *testing.T) {
			assert.EqualValues(t, val, target)
		})
	}
}
