// Copyright 2019 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package statements

import (
	"fmt"
	"strings"

	"xorm.io/builder"
	"xorm.io/xorm/schemas"
)

// ErrUnsupportedExprType represents an error with unsupported express type
type ErrUnsupportedExprType struct {
	tp string
}

func (err ErrUnsupportedExprType) Error() string {
	return fmt.Sprintf("Unsupported expression type: %v", err.tp)
}

// Expr represents an SQL express
type Expr struct {
	ColName string
	Arg     interface{}
}

type exprParams []Expr

func (exprs exprParams) ColNames() []string {
	var cols = make([]string, 0, len(exprs))
	for _, expr := range exprs {
		cols = append(cols, expr.ColName)
	}
	return cols
}

func (exprs exprParams) addParam(colName string, arg interface{}) {
	exprs = append(exprs, Expr{colName, arg})
}

func (exprs exprParams) IsColExist(colName string) bool {
	for _, expr := range exprs {
		if strings.EqualFold(schemas.CommonQuoter.Trim(expr.ColName), schemas.CommonQuoter.Trim(colName)) {
			return true
		}
	}
	return false
}

func (exprs exprParams) getByName(colName string) (Expr, bool) {
	for _, expr := range exprs {
		if strings.EqualFold(expr.ColName, colName) {
			return expr, true
		}
	}
	return Expr{}, false
}

func (exprs exprParams) WriteArgs(w *builder.BytesWriter) error {
	for i, expr := range exprs {
		switch arg := expr.Arg.(type) {
		case *builder.Builder:
			if _, err := w.WriteString("("); err != nil {
				return err
			}
			if err := arg.WriteTo(w); err != nil {
				return err
			}
			if _, err := w.WriteString(")"); err != nil {
				return err
			}
		case string:
			if arg == "" {
				arg = "''"
			}
			if _, err := w.WriteString(fmt.Sprintf("%v", arg)); err != nil {
				return err
			}
		default:
			if _, err := w.WriteString("?"); err != nil {
				return err
			}
			w.Append(arg)
		}
		if i != len(exprs)-1 {
			if _, err := w.WriteString(","); err != nil {
				return err
			}
		}
	}
	return nil
}

func (exprs exprParams) writeNameArgs(w *builder.BytesWriter) error {
	for i, expr := range exprs {
		if _, err := w.WriteString(expr.ColName); err != nil {
			return err
		}
		if _, err := w.WriteString("="); err != nil {
			return err
		}

		switch arg := expr.Arg.(type) {
		case *builder.Builder:
			if _, err := w.WriteString("("); err != nil {
				return err
			}
			if err := arg.WriteTo(w); err != nil {
				return err
			}
			if _, err := w.WriteString("("); err != nil {
				return err
			}
		default:
			w.Append(expr.Arg)
		}

		if i+1 != len(exprs) {
			if _, err := w.WriteString(","); err != nil {
				return err
			}
		}
	}
	return nil
}
