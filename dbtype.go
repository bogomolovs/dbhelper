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

// Holds information specific for different database types.
type DbType interface {
	// Placeholders are different for different database types.
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
