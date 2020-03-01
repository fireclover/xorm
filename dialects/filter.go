// Copyright 2019 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dialects

import (
	"fmt"
	"strings"

	"xorm.io/xorm/schemas"
)

// Filter is an interface to filter SQL
type Filter interface {
	Do(sql string) string
}

// QuoteFilter filter SQL replace ` to database's own quote character
type QuoteFilter struct {
	quoter schemas.Quoter
}

func (s *QuoteFilter) Do(sql string) string {
	if s.quoter.IsEmpty() {
		return sql
	}

	var buf strings.Builder
	buf.Grow(len(sql))

	var beginSingleQuote bool
	var prefix = true
	for i := 0; i < len(sql); i++ {
		if !beginSingleQuote && sql[i] == '`' {
			if prefix {
				buf.WriteByte(s.quoter.Prefix)
			} else {
				buf.WriteByte(s.quoter.Suffix)
			}
			prefix = !prefix
		} else {
			if sql[i] == '\'' {
				beginSingleQuote = !beginSingleQuote
			}
			buf.WriteByte(sql[i])
		}
	}
	return buf.String()

}

// SeqFilter filter SQL replace ?, ? ... to $1, $2 ...
type SeqFilter struct {
	Prefix string
	Start  int
}

func convertQuestionMark(sql, prefix string, start int) string {
	var buf strings.Builder
	var beginSingleQuote bool
	var index = start
	for _, c := range sql {
		if !beginSingleQuote && c == '?' {
			buf.WriteString(fmt.Sprintf("%s%v", prefix, index))
			index++
		} else {
			if c == '\'' {
				beginSingleQuote = !beginSingleQuote
			}
			buf.WriteRune(c)
		}
	}
	return buf.String()

}

func (s *SeqFilter) Do(sql string) string {
	return convertQuestionMark(sql, s.Prefix, s.Start)
}
