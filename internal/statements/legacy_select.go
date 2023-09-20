// Copyright 2022 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"fmt"

	"xorm.io/builder"
)

// isUsingLegacy returns true if xorm uses legacy LIMIT OFFSET.
// It's only available in sqlserver and oracle, if param USE_LEGACY_LIMIT_OFFSET is set to "true"
func (statement *Statement) isUsingLegacyLimitOffset() bool {
	u, ok := statement.dialect.(interface{ UseLegacyLimitOffset() bool })
	return ok && u.UseLegacyLimitOffset()
}

func (statement *Statement) writeSelectWithFns(buf *builder.BytesWriter, writeFuncs ...func(*builder.BytesWriter) error) (err error) {
	for _, fn := range writeFuncs {
		if err = fn(buf); err != nil {
			return
		}
	}
	return
}

// write mssql legacy query sql
func (statement *Statement) writeMssqlLegacySelect(buf *builder.BytesWriter, columnStr string) error {
	writeFns := []func(*builder.BytesWriter) error{
		func(bw *builder.BytesWriter) (err error) {
			_, err = fmt.Fprintf(bw, "SELECT")
			return
		},
		func(bw *builder.BytesWriter) error { return statement.writeDistinct(bw) },
		func(bw *builder.BytesWriter) error { return statement.writeTop(bw) },
		statement.writeFrom,
		statement.writeWhereWithMssqlPagination,
		func(bw *builder.BytesWriter) error { return statement.writeGroupBy(bw) },
		func(bw *builder.BytesWriter) error { return statement.writeHaving(bw) },
		func(bw *builder.BytesWriter) error { return statement.writeOrderBys(bw) },
		func(bw *builder.BytesWriter) error { return statement.writeForUpdate(bw) },
	}
	return statement.writeSelectWithFns(buf, writeFns...)
}

func (statement *Statement) writeOracleLegacySelect(buf *builder.BytesWriter, columnStr string) error {
	writeFns := []func(*builder.BytesWriter) error{
		func(bw *builder.BytesWriter) error { return statement.writeSelectColumns(bw, columnStr) },
		statement.writeFrom,
		func(bw *builder.BytesWriter) error { return statement.writeOracleLimit(bw, columnStr) },
		func(bw *builder.BytesWriter) error { return statement.writeGroupBy(bw) },
		func(bw *builder.BytesWriter) error { return statement.writeHaving(bw) },
		func(bw *builder.BytesWriter) error { return statement.writeOrderBys(bw) },
		func(bw *builder.BytesWriter) error { return statement.writeForUpdate(bw) },
	}
	return statement.writeSelectWithFns(buf, writeFns...)
}
