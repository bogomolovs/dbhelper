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
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Stores field data.
type dbField struct {
	// Field index in the structure.
	index []int

	// Name of the column in database.
	column string

	// Autoincremented field.
	auto bool

	// This field is an identifier ow the row.
	id bool

	// This field stores a timestamp of time when the record was created.
	created bool

	// This field stores a timestamp of time when the record was modified.
	modified bool
}

// Stores information about database table.
type dbTable struct {
	dbHelper   *DbHelper
	structType reflect.Type
	name       string

	fields        map[string]*dbField
	idField       *dbField
	createdField  *dbField
	modifiedField *dbField

	numField     int
	numFieldAuto int

	insertQuery *Pstmt
	updateQuery *Pstmt
	deleteQuery *Pstmt
}

// Returns pointer to new database table structure.
func (dbh *DbHelper) newDbTable(t reflect.Type, name string) (*dbTable, error) {
	if t.Kind() != reflect.Struct {
		return nil, errors.New(fmt.Sprintf("dbhelper: type '%v' is not a structure", t))
	}

	// number of fields
	num := t.NumField()

	// new database table structure
	tbl := &dbTable{
		dbHelper:   dbh,
		structType: t,
		name:       name,
		fields:     make(map[string]*dbField),
	}

	// check all fields in the structure
	for i := 0; i < num; i++ {
		// parse field to include embedded structures
		fields, err := tbl.parseField(t.Field(i))
		if err != nil {
			return nil, err
		}

		// add fields to table
		for _, f := range fields {
			// check that column name is unique
			if _, ok := tbl.fields[f.column]; ok {
				return nil, errors.New(
					fmt.Sprintf("dbhelper: attempt to define several fields with the same column name '%s' in structure type '%v'",
						f.column, t))
			}

			// add field to table
			tbl.numField++
			tbl.fields[f.column] = f

			// increase number of auto incremented fields
			if f.auto {
				tbl.numFieldAuto++
			}

			// store id field
			if f.id {
				if tbl.idField != nil {
					return nil, errors.New(
						fmt.Sprintf("dbhelper: attempt to define several fields with 'id' option in structure type '%v'", t))
				}

				tbl.idField = f
			}

			// store created field
			if f.created {
				if tbl.createdField != nil {
					return nil, errors.New(
						fmt.Sprintf("dbhelper: attempt to define several fields with 'created' option in structure type '%v'", t))
				}

				tbl.createdField = f
			}

			// store modified field
			if f.modified {
				if tbl.modifiedField != nil {
					return nil, errors.New(
						fmt.Sprintf("dbhelper: attempt to define several fields with 'modified' option in structure type '%v'", t))
				}

				tbl.modifiedField = f
			}
		}
	}

	// check that structure has fields
	if tbl.numField == 0 {
		return nil, errors.New(fmt.Sprintf("dbhelper: structure type '%v' has no exported fields", t))
	}

	// table must have an id field
	if tbl.idField == nil {
		return nil, errors.New(fmt.Sprintf("dbhelper: structure type '%v' has no field with option 'id'", t))
	}

	// prepare standart queries
	err := tbl.prepareStandardQueries()
	if err != nil {
		return nil, err
	}

	return tbl, nil
}

// Returns a slice of fields including embedded structures fields.
func (tbl *dbTable) parseField(field reflect.StructField) ([]*dbField, error) {
	// slice that will contain all fields
	fields := make([]*dbField, 0, 1)

	// check if field is anonymous
	if field.Anonymous {
		// check if field is embedded struct
		fieldType := field.Type
		if fieldType.Kind() != reflect.Struct {
			return nil, errors.New(fmt.Sprintf(
				"dbhelper: anonymous field of structure type'%v' has unsupported type '%v'. Only embedded structures are supported",
				tbl.structType, field.Type))
		}

		// number of fields in embedded structure
		num := fieldType.NumField()

		for i := 0; i < num; i++ {
			// parse field of embedded structure
			subFields, err := tbl.parseField(fieldType.Field(i))
			if err != nil {
				return nil, err
			}

			// append indexes of sub-fields
			for _, f := range subFields {
				l := len(f.index) + 1
				newIndex := make([]int, 1, l)
				newIndex[0] = field.Index[0]
				f.index = append(newIndex, f.index...)
			}

			// append fields from embedded structure
			fields = append(fields, subFields...)
		}
	} else {
		// check that field is exported
		if field.PkgPath != "" {
			return fields, nil
		}

		// check that field has supported type
		if !checkFieldType(field.Type) {
			return nil, errors.New(fmt.Sprintf("dbhelper: field '%s' of structure type'%v' has unsupported type '%v'",
				field.Name, tbl.structType, field.Type))
		}

		// get field db tag
		column := field.Tag.Get("db")
		if column == "" {
			// if db tag is empty, use field name as column name
			column = field.Name
		}

		// create new dbField structure
		f := &dbField{
			index:  field.Index,
			column: column,
		}

		// parse field options
		dbopt := field.Tag.Get("dbopt")
		if dbopt != "" {
			// remove spaces
			dbopt = strings.Replace(dbopt, " ", "", -1)

			// split flags
			opts := strings.Split(dbopt, ",")
			for _, opt := range opts {
				switch opt {
				case "auto":
					f.auto = true
				case "id":
					f.id = true
				case "created":
					f.created = true
				case "modified":
					f.modified = true
				case "skip":
					continue
				default:
					return nil, errors.New(fmt.Sprintf("dbhelper: unknown option '%s' for field '%s' in structure type '%v'",
						opt, field.Name, tbl.structType))
				}
			}
		}

		// append new field to slice
		fields = append(fields, f)
	}

	return fields, nil
}

// Returns fields that can be inserted and named placeholders
func (tbl *dbTable) getInsertFields() ([]string, []string) {
	fields := make([]string, 0, tbl.numField)
	holders := make([]string, 0, tbl.numField)

	for col, f := range tbl.fields {
		if f.auto {
			continue
		}

		fields = append(fields, col)
		holders = append(holders, getNamedPlaceholder(col))
	}

	return fields, holders
}

// Returns fields that can be updated and named placeholders
func (tbl *dbTable) getUpdateFields() ([]string, []string) {
	fields := make([]string, 0, tbl.numField)
	holders := make([]string, 0, tbl.numField)

	for col, f := range tbl.fields {
		if f.id || f.auto || f.created {
			continue
		}

		fields = append(fields, col)
		holders = append(holders, getNamedPlaceholder(col))
	}

	return fields, holders
}

func getNamedPlaceholder(name string) string {
	return fmt.Sprintf(":%s", name)
}

func (tbl *dbTable) prepareStandardQueries() error {
	// error
	var err error

	// insert fields and placeholders
	fields, ph := tbl.getInsertFields()

	// insert query postfix
	insertPostfix := ""
	if dbt, ok := tbl.dbHelper.dbType.(hasInsertPostfix); ok {
		insertPostfix = dbt.insertPostfix(tbl)
	}

	// insert SQL query
	insertQuery := fmt.Sprintf("INSERT INTO %s(%s) VALUES(%s) %s",
		tbl.name, strings.Join(fields, ", "), strings.Join(ph, ", "), insertPostfix)

	// prepare insert query
	tbl.insertQuery, err = tbl.dbHelper.Prepare(insertQuery)
	if err != nil {
		return err
	}

	// update fields and placeholders
	fields, ph = tbl.getUpdateFields()

	// number of non-auto fields
	num := len(fields)

	// prepare field assignments
	updateFields := make([]string, num, num)
	for i, f := range fields {
		updateFields[i] = fmt.Sprintf("%s = %s", f, ph[i])
	}

	// update SQL query
	updateQuery := fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s",
		tbl.name, strings.Join(updateFields, ", "), tbl.idField.column, getNamedPlaceholder(tbl.idField.column))

	// prepare udpate query
	tbl.updateQuery, err = tbl.dbHelper.Prepare(updateQuery)
	if err != nil {
		return err
	}

	// delete SQL query
	deleteQuery := fmt.Sprintf("DELETE FROM %s WHERE %s = %s",
		tbl.name, tbl.idField.column, getNamedPlaceholder(tbl.idField.column))

	// prepare udpate query
	tbl.deleteQuery, err = tbl.dbHelper.Prepare(deleteQuery)
	if err != nil {
		return err
	}

	return nil
}
