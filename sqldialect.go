// Copyright 2015 Sergii Bogomolov. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package dbhelper helps to interact with sql.DB by generating, preparing and
// executing queries. It marshals Go structs to and from databases and uses
// database/sql.
//
// Source code and project home:
// https://github.com/bogomolovs/dbhelper
//
package dbhelper

import (
	"fmt"
)

// Holds information specific for different database dialects.
type SqlDialect interface {
	// Placeholders are different for different database dialects.
	placeholder() placeholder
}

// Postfix for insert statement. Sometimes needed to get last inserted id.
type hasInsertPostfix interface {
	// Sometimes needed to last inserted id.
	insertPostfix(tbl *dbTable) string
}

// Actions after execution of insert query. Sometimes needed to get last inserted id.
type hasCustomInsert interface {
	// Sometimes needed to last inserted id.
	insert(tbl *dbTable, params map[string]interface{}) (int64, error)
}

// Placeholder interface.
type placeholder interface {
	next() string
}

// Placeholder format: "?".
type standardPlaceholder struct {
}

// Returns next placeholder.
func (ph *standardPlaceholder) next() string {
	return "?"
}

//
// Postgresql
//

// Postgresql SQL dialect.
type Postgresql struct {
}

// Returns placeholder generator.
func (sqld Postgresql) placeholder() placeholder {
	return &pgsqlPlaceholder{0}
}

// Postfix needed for Postgresql to return last inserted id.
func (sqld Postgresql) insertPostfix(tbl *dbTable) string {
	return fmt.Sprintf("RETURNING %s", tbl.idField.column)
}

// Custom insert query for Postgresql databse is needed to return last inserted record id.
func (sqld Postgresql) insert(tbl *dbTable, params map[string]interface{}) (int64, error) {
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

//
// MySQL
//

// MySql SQL dialect.
type MySql struct {
}

// Returns placeholder generator.
func (sqld MySql) placeholder() placeholder {
	return &standardPlaceholder{}
}

//
// Sqlite
//

// Sqlite SQL dialect.
type Sqlite struct {
}

// Returns placeholder generator.
func (sqld Sqlite) placeholder() placeholder {
	return &standardPlaceholder{}
}
