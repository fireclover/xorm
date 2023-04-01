package xorm

import (
	"fmt"
	"reflect"
	"strings"
	"xorm.io/builder"
	"xorm.io/xorm/schemas"
)

type Preload struct {
	path    []string
	cols    []string
	cond    builder.Cond
	noPrune bool
}

func NewPreload(path string) *Preload {
	return &Preload{
		path: strings.Split(path, "."),
		cond: builder.NewCond(),
	}
}

func (p *Preload) Cols(cols ...string) *Preload {
	p.cols = append(p.cols, cols...)
	return p
}

func (p *Preload) Where(cond builder.Cond) *Preload {
	p.cond = p.cond.And(cond)
	return p
}

func (p *Preload) NoPrune() *Preload {
	p.noPrune = true
	return p
}

type PreloadNode struct {
	preload     *Preload
	children    map[string]*PreloadNode
	association *schemas.Association
	ExtraCols   []string
}

func NewPreloadNode() *PreloadNode {
	return &PreloadNode{
		children: make(map[string]*PreloadNode),
	}
}

func (pn *PreloadNode) Add(preload *Preload) error {
	return pn.add(preload, 0)
}

func (pn *PreloadNode) add(preload *Preload, index int) error {
	if index == len(preload.path) {
		if pn.preload != nil {
			return fmt.Errorf("preload: duplicated path: %s", strings.Join(preload.path, ","))
		}
		pn.preload = preload
		return nil
	}
	child, ok := pn.children[preload.path[index]]
	if !ok {
		child = NewPreloadNode()
		pn.children[preload.path[index]] = child
	}
	return child.add(preload, index+1)
}

func (pn *PreloadNode) Validate(table *schemas.Table) error {
	if pn.preload != nil {
		for _, col := range pn.preload.cols {
			if table.GetColumn(col) == nil {
				return fmt.Errorf("preload: missing col %s in table %s", col, table.Name)
			}
		}
	}
	for name, node := range pn.children {
		column := table.GetColumn(name)
		if column == nil {
			return fmt.Errorf("preload: missing field %s in struct %s", name, table.Type.Name())
		}
		if column.Association == nil {
			return fmt.Errorf("preload: missing association in field %s", name)
		}
		if column.Association.JoinTable == nil && len(column.Association.SourceCol) > 0 {
			pn.ExtraCols = append(pn.ExtraCols, column.Association.SourceCol)
		}
		if len(column.Association.TargetCol) > 0 {
			pn.ExtraCols = append(pn.ExtraCols, table.PrimaryKeys[0]) // pk might be added many times, but that's ok
		}
		if column.Association.JoinTable == nil && len(column.Association.TargetCol) > 0 {
			node.ExtraCols = append(node.ExtraCols, column.Association.TargetCol)
		}
		node.association = column.Association
		if err := node.Validate(column.Association.RefTable); err != nil {
			return err
		}
	}
	return nil
}

func (pn *PreloadNode) Compute(session *Session, ownMap reflect.Value) error {
	for _, node := range pn.children {
		if err := node.compute(session, ownMap, reflect.Value{}); err != nil {
			return err
		}
	}
	return nil
}

func (pn *PreloadNode) compute(session *Session, ownMap, pruneMap reflect.Value) error {
	// non-root node: pn.association is not nil
	if err := pn.association.ValidateOwnMap(ownMap); err != nil {
		return err
	}

	var joinMap reflect.Value
	cond := pn.association.GetCond(ownMap)
	if pn.association.JoinTable != nil {
		var err error
		cond, joinMap, err = pn.preloadJoin(session, cond)
		if err != nil {
			return err
		}
	}

	refMap := pn.association.MakeRefMap()
	preloadSession := session.Engine().Cols(pn.ExtraCols...).Where(cond)
	if pn.preload != nil {
		preloadSession.Cols(pn.preload.cols...).Where(pn.preload.cond)
	}
	if err := preloadSession.Find(refMap.Interface()); err != nil {
		return err
	}

	var refPruneMap reflect.Value
	if len(pn.children) > 0 && !(pn.preload != nil && (len(pn.preload.cols) > 0 || pn.preload.noPrune)) {
		refPruneMap = reflect.MakeMap(reflect.MapOf(refMap.Type().Key(), reflect.TypeOf(true)))
		refIter := refMap.MapRange()
		for refIter.Next() {
			refPruneMap.SetMapIndex(refIter.Key(), reflect.ValueOf(true))
		}
	}

	for _, node := range pn.children {
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

	pn.association.Link(ownMap, refMap, pruneMap, joinMap)
	return nil
}

func (pn *PreloadNode) preloadJoin(session *Session, cond builder.Cond) (builder.Cond, reflect.Value, error) {
	joinSlicePtr := pn.association.MakeJoinSlice()
	if err := session.Engine().
		Table(pn.association.JoinTable.Name).Where(cond).
		Cols(pn.association.SourceCol, pn.association.TargetCol).
		Find(joinSlicePtr.Interface()); err != nil {
		return nil, reflect.Value{}, err
	}
	joinSlice := joinSlicePtr.Elem()

	joinMap := pn.association.MakeJoinMap()
	for i := 0; i < joinSlice.Len(); i++ {
		entry := joinSlice.Index(i)
		pkSlice := joinMap.MapIndex(entry.Field(1))
		if !pkSlice.IsValid() {
			pkSlice = reflect.MakeSlice(reflect.SliceOf(pn.association.OwnPkType), 0, 0)
		}
		joinMap.SetMapIndex(entry.Field(1), reflect.Append(pkSlice, entry.Field(0)))
	}

	var refPks []interface{}
	iter := joinMap.MapRange()
	joinMap.MapKeys()
	for iter.Next() {
		refPks = append(refPks, iter.Key().Interface())
	}
	cond = builder.In(pn.association.RefTable.PrimaryKeys[0], refPks)
	return cond, joinMap, nil
}
