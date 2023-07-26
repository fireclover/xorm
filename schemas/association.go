package schemas

import (
	"fmt"
	"reflect"
	"xorm.io/builder"
)

// Association is the representation of an association
type Association struct {
	OwnTable  *Table
	OwnColumn *Column
	OwnPkType reflect.Type
	RefTable  *Table
	RefPkType reflect.Type
	JoinTable *Table // many_to_many
	SourceCol string // belongs_to, many_to_many
	TargetCol string // has_one, has_many, many_to_many
}

// NewJoinSlice creates a slice to hold the intermediate result of a many-to-many association query
func (a *Association) NewJoinSlice() reflect.Value {
	return reflect.New(reflect.SliceOf(a.JoinTable.Type))
}

// MakeJoinMap creates a map to hold the intermediate result of a many-to-many association
func (a *Association) MakeJoinMap() reflect.Value {
	return reflect.MakeMap(reflect.MapOf(a.RefPkType, reflect.SliceOf(a.OwnPkType)))
}

// MakeRefMap creates a map to hold the result of an association query
func (a *Association) MakeRefMap() reflect.Value {
	return reflect.MakeMap(reflect.MapOf(a.RefPkType, reflect.PtrTo(a.RefTable.Type)))
}

// ValidateOwnMap validates the type of the owner map (parent of an association)
func (a *Association) ValidateOwnMap(ownMap reflect.Value) error {
	if ownMap.Type() != reflect.MapOf(a.OwnPkType, reflect.PtrTo(a.OwnTable.Type)) {
		return fmt.Errorf("wrong map type: %v", ownMap.Type())
	}
	return nil
}

// GetCond gets a where condition to use in an association query
func (a *Association) GetCond(ownMap reflect.Value) builder.Cond {
	if a.JoinTable != nil {
		return a.condManyToMany(ownMap)
	}
	if len(a.SourceCol) > 0 {
		return a.condBelongsTo(ownMap)
	}
	return a.condHasOneOrMany(ownMap)
}

// condBelongsTo gets a where condition to use in a belongs-to association query
func (a *Association) condBelongsTo(ownMap reflect.Value) builder.Cond {
	pkMap := make(map[interface{}]bool)
	fkCol := a.OwnTable.GetColumn(a.SourceCol)
	iter := ownMap.MapRange()
	for iter.Next() {
		structPtr := iter.Value()
		fk, _ := fkCol.ValueOfV(&structPtr)
		if fk.Type().Kind() == reflect.Ptr {
			if fk.IsNil() {
				continue
			}
			*fk = fk.Elem()
		}
		// don't check fk.IsZero(), because it might be a valid fk value
		pkMap[fk.Interface()] = true
	}
	pks := make([]interface{}, 0, len(pkMap))
	for pk := range pkMap {
		pks = append(pks, pk)
	}
	return builder.In(a.RefTable.PrimaryKeys[0], pks)
}

// condHasOneOrMany gets a where condition to use in a has-one or has-many association query
func (a *Association) condHasOneOrMany(ownMap reflect.Value) builder.Cond {
	var pks []interface{}
	iter := ownMap.MapRange()
	for iter.Next() {
		pks = append(pks, iter.Key().Interface())
	}
	return builder.In(a.TargetCol, pks)
}

// condHasOneOrMany gets a where condition to use in a many-to-many association query
func (a *Association) condManyToMany(ownMap reflect.Value) builder.Cond {
	var pks []interface{}
	iter := ownMap.MapRange()
	for iter.Next() {
		pks = append(pks, iter.Key().Interface())
	}
	return builder.In(a.SourceCol, pks)
}

// Link links the owner (parent) values with the referenced association values
func (a *Association) Link(ownMap, refMap, pruneMap, joinMap reflect.Value) {
	if a.JoinTable != nil {
		a.linkManyToMany(ownMap, refMap, pruneMap, joinMap)
	} else if len(a.SourceCol) > 0 {
		a.linkBelongsTo(ownMap, refMap, pruneMap)
	} else {
		a.linkHasOneOrMany(ownMap, refMap, pruneMap)
	}
}

// linkBelongsTo links the owner (parent) values with the referenced belongs-to association values
func (a *Association) linkBelongsTo(ownMap, refMap, pruneMap reflect.Value) {
	fkCol := a.OwnTable.GetColumn(a.SourceCol)
	iter := ownMap.MapRange()
	for iter.Next() {
		structPtr := iter.Value()
		fk, _ := fkCol.ValueOfV(&structPtr)
		if fk.Type().Kind() == reflect.Ptr {
			if fk.IsNil() {
				continue
			}
			*fk = fk.Elem()
		}
		// don't check fk.IsZero(), because it might be a valid fk value
		refStructPtr := refMap.MapIndex(*fk)
		if refStructPtr.IsValid() {
			refField, _ := a.OwnColumn.ValueOfV(&structPtr)
			refField.Set(refStructPtr)
			if pruneMap.IsValid() {
				pruneMap.SetMapIndex(iter.Key(), reflect.Value{}) // do not prune this key
			}
		}
	}
}

// linkBelongsTo links the owner (parent) values with the referenced has-one or has-many association values
func (a *Association) linkHasOneOrMany(ownMap, refMap, pruneMap reflect.Value) {
	hasMany := a.OwnColumn.FieldType.Kind() == reflect.Slice
	fkCol := a.RefTable.GetColumn(a.TargetCol)
	iter := refMap.MapRange()
	for iter.Next() {
		refStructPtr := iter.Value()
		fk, _ := fkCol.ValueOfV(&refStructPtr)
		if fk.Type().Kind() == reflect.Ptr {
			if fk.IsNil() {
				continue
			}
			*fk = fk.Elem()
		}
		// don't check fk.IsZero(), because it might be a valid fk value
		structPtr := ownMap.MapIndex(*fk) // structPtr should be valid at this point
		refField, _ := a.OwnColumn.ValueOfV(&structPtr)
		if hasMany {
			refField.Set(reflect.Append(*refField, refStructPtr))
		} else {
			refField.Set(refStructPtr)
		}
		if pruneMap.IsValid() {
			pruneMap.SetMapIndex(*fk, reflect.Value{}) // do not prune this key
		}
	}
}

// linkManyToMany links the owner (parent) values with the referenced many-to-many association values
func (a *Association) linkManyToMany(ownMap, refMap, pruneMap, joinMap reflect.Value) {
	iter := refMap.MapRange()
	for iter.Next() {
		refStructPtr := iter.Value()
		refPk := iter.Key()                // refPk should not be a pointer
		pkSlice := joinMap.MapIndex(refPk) // pkSlice should be valid at this point
		for i := 0; i < pkSlice.Len(); i++ {
			pk := pkSlice.Index(i)
			structPtr := ownMap.MapIndex(pk) // structPtr should be valid at this point
			refField, _ := a.OwnColumn.ValueOfV(&structPtr)
			refField.Set(reflect.Append(*refField, refStructPtr))
			if pruneMap.IsValid() {
				pruneMap.SetMapIndex(pk, reflect.Value{}) // do not prune this key
			}
		}
	}
}
