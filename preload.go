// Copyright 2023 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xorm

import (
	"fmt"
	"reflect"
	"strings"

	"xorm.io/builder"
	"xorm.io/xorm/v2/schemas"
)

// Preload is the representation of an association preload
type Preload struct {
	path []string
	cols []string
	cond builder.Cond
}

// NewPreload creates a new preload with the specified path
func NewPreload(path string) *Preload {
	return &Preload{
		path: strings.Split(path, "."), // list of association names composing the path
		cond: builder.NewCond(),
	}
}

// Cols sets column selection for this preload
func (p *Preload) Cols(cols ...string) *Preload {
	p.cols = append(p.cols, cols...)
	return p
}

// Where sets the where condition for this preload
func (p *Preload) Where(cond builder.Cond) *Preload {
	p.cond = p.cond.And(cond)
	return p
}

// PreloadTreeNode is a tree node for the association preloads
type PreloadTreeNode struct {
	preload     *Preload
	children    map[string]*PreloadTreeNode
	association *schemas.Association
	extraCols   []string
}

// NewPreloadTeeeNode creates a new preload tree node
func NewPreloadTeeeNode() *PreloadTreeNode {
	return &PreloadTreeNode{
		children: make(map[string]*PreloadTreeNode),
	}
}

// Add adds a node to the preload tree
func (node *PreloadTreeNode) Add(preload *Preload) error {
	return node.add(preload, 0)
}

// add adds a node to the preload tree in a recursion level
func (node *PreloadTreeNode) add(preload *Preload, level int) error {
	if level == len(preload.path) {
		if node.preload != nil {
			return fmt.Errorf("preload: duplicated path: %s", strings.Join(preload.path, ","))
		}
		node.preload = preload
		return nil
	}
	child, ok := node.children[preload.path[level]]
	if !ok {
		child = NewPreloadTeeeNode()
		node.children[preload.path[level]] = child
	}
	return child.add(preload, level+1)
}

// Validate validates a preload tree node against a table schema and sets the association
func (node *PreloadTreeNode) Validate(table *schemas.Table) error {
	if node.preload != nil {
		for _, col := range node.preload.cols {
			if table.GetColumn(col) == nil {
				return fmt.Errorf("preload: missing col %s in table %s", col, table.Name)
			}
		}
	}
	for name, child := range node.children {
		column := table.GetColumn(name)
		if column == nil {
			return fmt.Errorf("preload: missing field %s in struct %s", name, table.Type.Name())
		}
		if column.Association == nil {
			return fmt.Errorf("preload: missing association in field %s", name)
		}
		if column.Association.JoinTable == nil && len(column.Association.SourceCol) > 0 {
			node.extraCols = append(node.extraCols, column.Association.SourceCol)
		}
		if len(column.Association.TargetCol) > 0 {
			node.extraCols = append(node.extraCols, table.PrimaryKeys[0]) // pk might be added many times, but that's ok
		}
		if column.Association.JoinTable == nil && len(column.Association.TargetCol) > 0 {
			child.extraCols = append(child.extraCols, column.Association.TargetCol)
		}
		child.association = column.Association
		if err := child.Validate(column.Association.RefTable); err != nil {
			return err
		}
	}
	return nil
}

// Compute preloads the associations contained in the preload tree
func (node *PreloadTreeNode) Compute(session *Session, ownMap reflect.Value) error {
	for _, node := range node.children {
		if err := node.compute(session, ownMap, reflect.Value{}); err != nil {
			return err
		}
	}
	return nil
}

// compute preloads the association contained in a preload tree node
func (node *PreloadTreeNode) compute(session *Session, ownMap, pruneMap reflect.Value) error {
	// non-root node: association is not nil
	if err := node.association.ValidateOwnMap(ownMap); err != nil {
		return err
	}

	var joinMap reflect.Value
	cond := node.association.GetCond(ownMap)
	if node.association.JoinTable != nil {
		var err error
		cond, joinMap, err = node.preloadJoin(session, cond)
		if err != nil {
			return err
		}
	}

	refMap := node.association.MakeRefMap()
	preloadSession := session.Engine().Where(cond)
	if node.preload != nil {
		if len(node.preload.cols) > 0 {
			preloadSession.Cols(node.extraCols...).Cols(node.preload.cols...)
		}
		preloadSession.Where(node.preload.cond)
	} else {
		preloadSession.Cols(node.extraCols...)
	}
	if err := preloadSession.Find(refMap.Interface()); err != nil {
		return err
	}

	var refPruneMap reflect.Value
	if len(node.children) > 0 && !(node.preload != nil && len(node.preload.cols) > 0) {
		refPruneMap = reflect.MakeMap(reflect.MapOf(refMap.Type().Key(), reflect.TypeOf(true)))
		refIter := refMap.MapRange()
		for refIter.Next() {
			refPruneMap.SetMapIndex(refIter.Key(), reflect.ValueOf(true))
		}
	}

	for _, node := range node.children {
		if err := node.compute(session, refMap, refPruneMap); err != nil {
			return err
		}
	}

	if refPruneMap.IsValid() {
		pruneIter := refPruneMap.MapRange()
		for pruneIter.Next() {
			refMap.SetMapIndex(pruneIter.Key(), reflect.Value{})
		}
	}

	node.association.Link(ownMap, refMap, pruneMap, joinMap)
	return nil
}

// preloadJoin obtains a join condition and a join map for a many-to-many association
func (node *PreloadTreeNode) preloadJoin(session *Session, cond builder.Cond) (builder.Cond, reflect.Value, error) {
	joinSlicePtr := node.association.NewJoinSlice()
	if err := session.Engine().
		Table(node.association.JoinTable.Name).Where(cond).
		Cols(node.association.SourceCol, node.association.TargetCol).
		Find(joinSlicePtr.Interface()); err != nil {
		return nil, reflect.Value{}, err
	}
	joinSlice := joinSlicePtr.Elem()

	joinMap := node.association.MakeJoinMap()
	for i := 0; i < joinSlice.Len(); i++ {
		entry := joinSlice.Index(i)
		pkSlice := joinMap.MapIndex(entry.Field(1))
		if !pkSlice.IsValid() {
			pkSlice = reflect.MakeSlice(reflect.SliceOf(node.association.OwnPkType), 0, 0)
		}
		joinMap.SetMapIndex(entry.Field(1), reflect.Append(pkSlice, entry.Field(0)))
	}

	var refPks []interface{}
	iter := joinMap.MapRange()
	joinMap.MapKeys()
	for iter.Next() {
		refPks = append(refPks, iter.Key().Interface())
	}
	cond = builder.In(node.association.RefTable.PrimaryKeys[0], refPks)
	return cond, joinMap, nil
}
