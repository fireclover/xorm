// Copyright 2022 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"fmt"
	"strings"

	"xorm.io/builder"
	"xorm.io/xorm/dialects"
	"xorm.io/xorm/internal/utils"
)

// Join The joinOP should be one of INNER, LEFT OUTER, CROSS etc - this will be prepended to JOIN
func (statement *Statement) Join(joinOP string, joinTable interface{}, condition interface{}, args ...interface{}) *Statement {
	statement.joins = append(statement.joins, join{
		op:        joinOP,
		table:     joinTable,
		condition: condition,
		args:      args,
	})
	return statement
}

func (statement *Statement) writeJoins(w builder.Writer) error {
	for _, join := range statement.joins {
		if err := statement.writeJoin(w, join); err != nil {
			return err
		}
	}
	return nil
}

func (statement *Statement) writeJoin(buf builder.Writer, join join) error {
	// write join operator
	if _, err := fmt.Fprintf(buf, " %v JOIN ", join.op); err != nil {
		return err
	}

	// write table or sub query
	switch tp := join.table.(type) {
	case builder.Builder:
		if err := tp.WriteTo(buf); err != nil {
			return err
		}
	case *builder.Builder:
		if err := tp.WriteTo(buf); err != nil {
			return err
		}
	default:
		tbName := dialects.FullTableName(statement.dialect, statement.tagParser.GetTableMapper(), join.table, true)
		if !utils.IsSubQuery(tbName) {
			var buf strings.Builder
			_ = statement.dialect.Quoter().QuoteTo(&buf, tbName)
			tbName = buf.String()
		} else {
			tbName = statement.ReplaceQuote(tbName)
		}
		if _, err := fmt.Fprint(buf, tbName); err != nil {
			return err
		}
	}

	// write alias FIXME
	/*fields := strings.Split(tp.TableName(), ".")
	aliasName := statement.dialect.Quoter().Trim(fields[len(fields)-1])
	aliasName = schemas.CommonQuoter.Trim(aliasName)
	if _, err := fmt.Fprint(buf, " ", statement.quote(aliasName)); err != nil {
		return err
	}*/

	// write condition
	if _, err := fmt.Fprint(buf, " ON "); err != nil {
		return err
	}

	switch condTp := join.condition.(type) {
	case string:
		if _, err := fmt.Fprint(buf, condTp); err != nil {
			return err
		}
		buf.Append(join.args...)
	case builder.Cond:
		if err := condTp.WriteTo(buf); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported join condition type: %v", condTp)
	}

	return nil
}
