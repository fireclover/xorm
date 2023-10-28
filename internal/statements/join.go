// Copyright 2022 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"fmt"
	"strings"

	"xorm.io/builder"
	"xorm.io/xorm/v2/dialects"
	"xorm.io/xorm/v2/internal/utils"
	"xorm.io/xorm/v2/schemas"
)

// Join The joinOP should be one of INNER, LEFT OUTER, CROSS etc - this will be prepended to JOIN
func (statement *Statement) Join(joinOP string, joinTable any, condition any, args ...any) *Statement {
	statement.joins = append(statement.joins, join{
		op:        joinOP,
		table:     joinTable,
		condition: condition,
		args:      args,
	})
	return statement
}

func (statement *Statement) writeJoins(w *builder.BytesWriter) error {
	for _, join := range statement.joins {
		if err := statement.writeJoin(w, join); err != nil {
			return err
		}
	}
	return nil
}

func (statement *Statement) writeJoin(buf *builder.BytesWriter, join join) error {
	// write join operator
	if _, err := fmt.Fprint(buf, " ", join.op, " JOIN"); err != nil {
		return err
	}

	// write join table or subquery
	switch tp := join.table.(type) {
	case builder.Builder:
		if _, err := fmt.Fprintf(buf, " ("); err != nil {
			return err
		}
		if err := tp.WriteTo(statement.QuoteReplacer(buf)); err != nil {
			return err
		}

		fields := strings.Split(tp.TableName(), ".")
		aliasName := statement.dialect.Quoter().Trim(fields[len(fields)-1])
		aliasName = schemas.CommonQuoter.Trim(aliasName)

		if _, err := fmt.Fprintf(buf, ") %s", statement.quote(aliasName)); err != nil {
			return err
		}
	case *builder.Builder:
		if _, err := fmt.Fprintf(buf, " ("); err != nil {
			return err
		}
		if err := tp.WriteTo(statement.QuoteReplacer(buf)); err != nil {
			return err
		}

		fields := strings.Split(tp.TableName(), ".")
		aliasName := statement.dialect.Quoter().Trim(fields[len(fields)-1])
		aliasName = schemas.CommonQuoter.Trim(aliasName)

		if _, err := fmt.Fprintf(buf, ") %s", statement.quote(aliasName)); err != nil {
			return err
		}
	default:
		tbName := dialects.FullTableName(statement.dialect, statement.tagParser.GetTableMapper(), join.table, true)
		if !utils.IsSubQuery(tbName) {
			var sb strings.Builder
			if err := statement.dialect.Quoter().QuoteTo(&sb, tbName); err != nil {
				return err
			}
			tbName = sb.String()
		} else {
			tbName = statement.ReplaceQuote(tbName)
		}
		if _, err := fmt.Fprint(buf, " ", tbName); err != nil {
			return err
		}
	}

	// write on condition
	if _, err := fmt.Fprint(buf, " ON "); err != nil {
		return err
	}

	switch condTp := join.condition.(type) {
	case string:
		if _, err := fmt.Fprint(buf, statement.ReplaceQuote(condTp)); err != nil {
			return err
		}
	case builder.Cond:
		if err := condTp.WriteTo(statement.QuoteReplacer(buf)); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported join condition type: %v", condTp)
	}
	buf.Append(join.args...)

	return nil
}
