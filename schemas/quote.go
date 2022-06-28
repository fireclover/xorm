// Copyright 2020 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package schemas

import (
	"fmt"
	"regexp"
	"strings"
)

// Quoter represents a quoter to the SQL table name and column name
type Quoter struct {
	prefix     byte
	suffix     byte
	isReserved func(string) bool
	re         *regexp.Regexp
}

var regexCharsToEscape = map[byte]struct{}{
	'[': {}, ']': {}, '(': {}, ')': {}, '{': {}, '}': {}, '*': {}, '+': {}, '?': {}, '|': {}, '^': {}, '$': {}, '.': {}, '\\': {}, '`': {},
}

func escapedRegexBytes(in byte) []byte {
	if _, ok := regexCharsToEscape[in]; !ok {
		return []byte{in}
	}
	return []byte{'\\', in}
}

func NewQuoter(prefix byte, suffix byte, isReserved func(string) bool) (Quoter, error) {
	regexPrefix := escapedRegexBytes(prefix)
	regexSuffix := escapedRegexBytes(suffix)

	regex := fmt.Sprintf(`(?i)^\s*([^.\s]+|\x60[^.\s]+\x60|%s[^.\s]+%s)(?:\s*\.\s*([^.\s]+|\x60[^.\s]+\x60|%s[^.\s]+%s))?\s*?(?:\s+as\s+([^.\s]+|\x60[^.\s]+\x60|%s[^.\s]+%s))?(?:\s+(use|force)\s+index\s+\(([^.\s]+|\x60[^.\s]+\x60|%s[^.\s]+%s)\))?\s*$`,
		regexPrefix, regexSuffix, regexPrefix, regexSuffix, regexPrefix, regexSuffix, regexPrefix, regexSuffix,
	)
	re, err := regexp.Compile(regex)
	if err != nil {
		return Quoter{}, err
	}

	return Quoter{
		prefix:     prefix,
		suffix:     suffix,
		isReserved: isReserved,
		re:         re,
	}, nil
}

func (q *Quoter) SetIsReserved(isReserved func(string) bool) {
	q.isReserved = isReserved
}

var (
	// AlwaysNoReserve always think it's not a reverse word
	AlwaysNoReserve = func(string) bool { return false }

	// AlwaysReserve always reverse the word
	AlwaysReserve = func(string) bool { return true }

	// CommonQuoteMark represents the common quote mark
	CommonQuoteMark byte = '`'

	// CommonQuoter represents a common quoter
	CommonQuoter Quoter
)

func init() {
	var err error
	CommonQuoter, err = NewQuoter(CommonQuoteMark, CommonQuoteMark, AlwaysReserve)
	if err != nil {
		panic(err)
	}
}

// IsEmpty return true if no prefix and suffix
func (q Quoter) IsEmpty() bool {
	return q.prefix == 0 && q.suffix == 0
}

// Quote quote a string
func (q Quoter) Quote(s string) string {
	var buf strings.Builder
	_ = q.QuoteTo(&buf, s)
	return buf.String()
}

// Trim removes quotes from s
func (q Quoter) Trim(s string) string {
	if len(s) < 2 {
		return s
	}

	var buf strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case i == 0 && s[i] == q.prefix:
		case i == len(s)-1 && s[i] == q.suffix:
		case s[i] == q.suffix && s[i+1] == '.':
		case s[i] == q.prefix && s[i-1] == '.':
		default:
			buf.WriteByte(s[i])
		}
	}
	return buf.String()
}

// Join joins a slice with quoters
func (q Quoter) Join(a []string, sep string) string {
	var b strings.Builder
	_ = q.JoinWrite(&b, a, sep)
	return b.String()
}

// JoinWrite writes quoted content to a builder
func (q Quoter) JoinWrite(b *strings.Builder, a []string, sep string) error {
	if len(a) == 0 {
		return nil
	}

	n := len(sep) * (len(a) - 1)
	for i := 0; i < len(a); i++ {
		n += len(a[i])
	}

	b.Grow(n)
	for i, s := range a {
		if i > 0 {
			if _, err := b.WriteString(sep); err != nil {
				return err
			}
		}
		if err := q.QuoteTo(b, strings.TrimSpace(s)); err != nil {
			return err
		}
	}
	return nil
}

func (q Quoter) quoteWordTo(buf *strings.Builder, word string) error {
	var realWord = word
	if (word[0] == CommonQuoteMark && word[len(word)-1] == CommonQuoteMark) ||
		(word[0] == q.prefix && word[len(word)-1] == q.suffix) {
		realWord = word[1 : len(word)-1]
	}

	if q.IsEmpty() {
		_, err := buf.WriteString(realWord)
		return err
	}

	isReserved := q.isReserved(realWord)
	if isReserved && realWord != "*" {
		if err := buf.WriteByte(q.prefix); err != nil {
			return err
		}
	}
	if _, err := buf.WriteString(realWord); err != nil {
		return err
	}
	if isReserved && realWord != "*" {
		return buf.WriteByte(q.suffix)
	}

	return nil
}

// QuoteTo quotes the table or column names. i.e. if the quotes are [ and ]
//   name -> [name]
//   `name` -> [name]
//   [name] -> [name]
//   schema.name -> [schema].[name]
//   `schema`.`name` -> [schema].[name]
//   `schema`.name -> [schema].[name]
//   schema.`name` -> [schema].[name]
//   [schema].name -> [schema].[name]
//   schema.[name] -> [schema].[name]
//   name AS a  ->  [name] AS a
//   schema.name AS a  ->  [schema].[name] AS a
func (q Quoter) QuoteTo(buf *strings.Builder, value string) (err error) {
	matches := q.re.FindStringSubmatch(value)
	if len(matches) != 6 {
		return fmt.Errorf("unable to determine quoting for %q", value)
	}

	schema := matches[1]
	table := matches[2]
	alias := matches[3]
	indexMethod := matches[4]
	index := matches[5]
	if table == "" {
		table = schema
		schema = ""
	}

	if schema != "" {
		if err = q.quoteWordTo(buf, schema); err != nil {
			return err
		}
		buf.WriteByte('.')
	}
	if err = q.quoteWordTo(buf, table); err != nil {
		return err
	}
	if alias != "" {
		buf.WriteString(" AS ")
		if err = q.quoteWordTo(buf, alias); err != nil {
			return err
		}
	}
	if index != "" {
		_, err = fmt.Fprintf(buf, " %s index (", indexMethod)
		if err != nil {
			return err
		}
		if err = q.quoteWordTo(buf, index); err != nil {
			return err
		}
		buf.WriteByte(')')
	}

	return nil
}

// Strings quotes a slice of string
func (q Quoter) Strings(s []string) []string {
	var res = make([]string, 0, len(s))
	for _, a := range s {
		res = append(res, q.Quote(a))
	}
	return res
}

// Replace replaces common quote(`) as the quotes on the sql
func (q Quoter) Replace(sql string) string {
	if q.IsEmpty() {
		return sql
	}

	var buf strings.Builder
	buf.Grow(len(sql))

	var beginSingleQuote bool
	for i := 0; i < len(sql); i++ {
		if !beginSingleQuote && sql[i] == CommonQuoteMark {
			var j = i + 1
			for ; j < len(sql); j++ {
				if sql[j] == CommonQuoteMark {
					break
				}
			}
			word := sql[i+1 : j]
			isReserved := q.isReserved(word)
			if isReserved {
				buf.WriteByte(q.prefix)
			}
			buf.WriteString(word)
			if isReserved {
				buf.WriteByte(q.suffix)
			}
			i = j
		} else {
			if sql[i] == '\'' {
				beginSingleQuote = !beginSingleQuote
			}
			buf.WriteByte(sql[i])
		}
	}
	return buf.String()
}
