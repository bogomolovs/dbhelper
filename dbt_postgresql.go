// Copyright 2014 Sergii Bogomolov. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package dbhelper helps to interact with sql.DB by generating, preparing and
// executing queries. It marshals Go structs to and from databases and uses
// database/sql.
//
// Source code and project home:
// https://github.com/biggunsv/dbhelper
//
package dbhelper

import (
	"fmt"
)

// Postgresql database type.
type Postgresql struct {
}

// Returns placeholder generator.
func (dbt Postgresql) placeholder() placeholder {
	return &pgsqlPlaceholder{0}
}

// Postfix needed for Postgresql to return last inserted id.
func (dbt Postgresql) insertPostfix(tbl *dbTable) string {
	return fmt.Sprintf("RETURNING %s", tbl.idField.column)
}

// Custom insert query for Postgresql databse is needed to return last inserted record id.
func (dbt Postgresql) insert(tbl *dbTable, params map[string]interface{}) (int64, error) {
	var id int64
	_, err := tbl.insertQuery.Query(&id, params)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// Placeholder format: "$n".
type pgsqlPlaceholder struct {
	n int
}

// Returns next placeholder.
func (ph *pgsqlPlaceholder) next() string {
	ph.n++
	return fmt.Sprintf("$%d", ph.n)
}
