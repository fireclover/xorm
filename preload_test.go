// Copyright 2023 The Xorm Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xorm

import (
	"sort"
	"testing"

	"xorm.io/builder"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// https://gitea.com/xorm/xorm/issues/2240

type Employee struct {
	Id           int64
	Name         string
	BuddyId      *int64
	ManagerId    *int64
	Buddy        *Employee   `xorm:"belongs_to(buddy_id)"`
	Apprentice   *Employee   `xorm:"has_one(buddy_id)"`
	Manager      *Employee   `xorm:"belongs_to(manager_id)"`
	Subordinates []*Employee `xorm:"has_many(manager_id)"`
	Indications  []*Employee `xorm:"many_to_many(employee_indication, indicator_id, indicated_id)"`
	IndicatedBy  []*Employee `xorm:"many_to_many(employee_indication, indicated_id, indicator_id)"`
}

func TestPreload(t *testing.T) {
	engine, err := NewEngine("sqlite3", ":memory:")
	require.NoError(t, err)

	const sql = `
create table employee (
    id integer primary key autoincrement,
    name text not null,
    buddy_id integer references employee(id) check (buddy_id <> id) unique,
    manager_id integer references employee(id) check (manager_id <> id)
);

create table employee_indication (
    indicator_id integer not null references employee(id),
    indicated_id integer not null references employee(id),
    primary key (indicator_id, indicated_id),
    check (indicator_id <> indicated_id)
);

insert into employee (name) values ('John'), ('Bob');
insert into employee (name,manager_id) values ('Alice',1), ('Riya',2);
insert into employee (name,manager_id,buddy_id) values ('Emilie',1,3), ('Cynthia',2,4);
insert into employee_indication values (1,2), (1,3), (2,3), (2,4), (2,5), (3,5), (3,6);

-- John manages Alice and Emilie
-- Bob manages Riya and Cynthia
-- Alice is buddy of Emilie
-- Riya is buddy of Cynthia
-- John indicated Bob and Alice
-- Bob indicated Alice, Riya and Emilie
-- Alice indicated Emilie and Cynthia
`
	_, err = engine.Exec(sql)
	require.NoError(t, err)

	var employee Employee
	_, err = engine.Preloads(
		engine.Preload("Indications.Buddy").Cols("name"),
		engine.Preload("Indications").Cols("id"),
	).Cols("name").Where(builder.Eq{"id": 2}).Get(&employee)
	require.NoError(t, err)

	// order is not preserved when preloading, so we compensate for it in the test
	sort.Slice(employee.Indications, func(i, j int) bool {
		return employee.Indications[i].Id < employee.Indications[j].Id
	})

	assert.Equal(t, Employee{
		Id:   2,
		Name: "Bob",
		Indications: []*Employee{
			{Id: 3},
			{Id: 4},
			{
				Id:      5,
				BuddyId: &[]int64{3}[0],
				Buddy: &Employee{
					Id:   3,
					Name: "Alice",
				},
			},
		},
	}, employee)

	var employees []*Employee
	err = engine.Preloads(
		// 1. preload the names of all subordinates of this employee's manager
		engine.Preload("Manager.Subordinates").Cols("name"),
		// 2. preload the name of the buddy of each employee indicated by this employee
		engine.Preload("Indications.Buddy").Cols("name"),
		// 3. preload the names of all employees who indicated this employee's apprentice, except non-subordinates
		engine.Preload("Apprentice.IndicatedBy").Cols("name").Where(builder.NotNull{"manager_id"}),
		// 4. preload the names of:
		// 	all employees who don't have a maanger and were indicated by:
		engine.Preload("Subordinates.IndicatedBy.Indications").Where(builder.IsNull{"manager_id"}).Cols("name"),
		// 	employees whose name is 4 letters long and indicated:
		engine.Preload("Subordinates.IndicatedBy").Where(builder.Like{"name", "____"}),
		// 	this employee's subordinates who don't have a buddy
		engine.Preload("Subordinates").Where(builder.IsNull{"buddy_id"}),
		// 0. find the names of all employees
	).Cols("name").Find(&employees)
	require.NoError(t, err)

	// order is not preserved when preloading, so we compensate for it in the test
	for k := 2; k < 6; k++ {
		sort.Slice(employees[k].Manager.Subordinates, func(i, j int) bool {
			return employees[k].Manager.Subordinates[i].Id < employees[k].Manager.Subordinates[j].Id
		})
	}
	sort.Slice(employees[2].Indications, func(i, j int) bool {
		return employees[2].Indications[i].Id < employees[2].Indications[j].Id
	})

	expected := []*Employee{
		{
			Id:   1,
			Name: "John",
			Subordinates: []*Employee{
				{
					Id:        3,
					Name:      "Alice",
					ManagerId: &[]int64{1}[0],
					IndicatedBy: []*Employee{
						{
							Id:   1,
							Name: "John",
							Indications: []*Employee{
								{
									Id:   2,
									Name: "Bob",
								},
							},
						},
					},
				},
			},
		},
		{
			Id:   2,
			Name: "Bob",
			Indications: []*Employee{
				{
					Id:      5,
					BuddyId: &[]int64{3}[0],
					Buddy: &Employee{
						Id:   3,
						Name: "Alice",
					},
				},
			},
		},
		{
			Id:        3,
			Name:      "Alice",
			ManagerId: &[]int64{1}[0],
			Manager: &Employee{
				Id: 1,
				Subordinates: []*Employee{
					{
						Id:        3,
						Name:      "Alice",
						ManagerId: &[]int64{1}[0],
					},
					{
						Id:        5,
						Name:      "Emilie",
						ManagerId: &[]int64{1}[0],
					},
				},
			},
			Apprentice: &Employee{
				Id:      5,
				BuddyId: &[]int64{3}[0],
				IndicatedBy: []*Employee{
					{
						Id:   3,
						Name: "Alice",
					},
				},
			},
			Indications: []*Employee{
				{
					Id:      5,
					BuddyId: &[]int64{3}[0],
					Buddy: &Employee{
						Id:   3,
						Name: "Alice",
					},
				},
				{
					Id:      6,
					BuddyId: &[]int64{4}[0],
					Buddy: &Employee{
						Id:   4,
						Name: "Riya",
					},
				},
			},
		},
		{
			Id:        4,
			Name:      "Riya",
			ManagerId: &[]int64{2}[0],
			Manager: &Employee{
				Id: 2,
				Subordinates: []*Employee{
					{
						Id:        4,
						Name:      "Riya",
						ManagerId: &[]int64{2}[0],
					},
					{
						Id:        6,
						Name:      "Cynthia",
						ManagerId: &[]int64{2}[0],
					},
				},
			},
			Apprentice: &Employee{
				Id:      6,
				BuddyId: &[]int64{4}[0],
				IndicatedBy: []*Employee{
					{
						Id:   3,
						Name: "Alice",
					},
				},
			},
		},
		{
			Id:        5,
			Name:      "Emilie",
			ManagerId: &[]int64{1}[0],
			Manager: &Employee{
				Id: 1,
				Subordinates: []*Employee{
					{
						Id:        3,
						Name:      "Alice",
						ManagerId: &[]int64{1}[0],
					},
					{
						Id:        5,
						Name:      "Emilie",
						ManagerId: &[]int64{1}[0],
					},
				},
			},
		},
		{
			Id:        6,
			Name:      "Cynthia",
			ManagerId: &[]int64{2}[0],
			Manager: &Employee{
				Id: 2,
				Subordinates: []*Employee{
					{
						Id:        4,
						Name:      "Riya",
						ManagerId: &[]int64{2}[0],
					},
					{
						Id:        6,
						Name:      "Cynthia",
						ManagerId: &[]int64{2}[0],
					},
				},
			},
		},
	}
	assert.Equal(t, expected, employees)
}
