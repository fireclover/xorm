// Copyright 2023 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"fmt"

	"xorm.io/builder"
)

func (statement *Statement) writeString(str string) func(w *builder.BytesWriter) error {
	return func(w *builder.BytesWriter) error {
		if _, err := fmt.Fprint(w, str); err != nil {
			return err
		}
		return nil
	}
}

func (statement *Statement) writeSpace(w *builder.BytesWriter) error {
	return statement.writeString(" ")(w)
}

func (statement *Statement) groupWriteFns(writeFuncs ...func(*builder.BytesWriter) error) func(*builder.BytesWriter) error {
	return func(bw *builder.BytesWriter) error {
		return statement.writeMultiple(bw, writeFuncs...)
	}
}

func (statement *Statement) writeMultiple(buf *builder.BytesWriter, writeFuncs ...func(*builder.BytesWriter) error) (err error) {
	for _, fn := range writeFuncs {
		if err = fn(buf); err != nil {
			return
		}
	}
	return
}
