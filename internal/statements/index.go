// Copyright 2023 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"strings"

	"xorm.io/builder"
)

type ErrInvalidIndexHintOperator struct {
	Op string
}

func (e ErrInvalidIndexHintOperator) Error() string {
	return "invalid index hint operator: " + e.Op
}

func (statement *Statement) IndexHint(op, indexName string) error {
	op = strings.ToUpper(op)
	statement.indexHints = append(statement.indexHints, indexHint{
		op:        op,
		indexName: indexName,
	})
	return nil
}

func (statement *Statement) writeIndexHints(w *builder.BytesWriter) error {
	switch statement.dialect.URI().DBType {
	case "mysql":
		return statement.writeIndexHintsMySQL(w)
	default:
		return ErrNotImplemented
	}
}

func (statement *Statement) writeIndexHintsMySQL(w *builder.BytesWriter) error {
	for _, hint := range statement.indexHints {
		if hint.op != "USE" && hint.op != "FORCE" && hint.op != "IGNORE" {
			return ErrInvalidIndexHintOperator{Op: hint.op}
		}
		if err := statement.writeStrings(hint.op, " INDEX(", hint.indexName, ")")(w); err != nil {
			return err
		}
	}
	return nil
}
