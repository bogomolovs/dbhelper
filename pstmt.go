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
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

// Contains prepared statement ready for execution.
type Pstmt struct {
	dbHelper *DbHelper
	params   []string
	stmt     *sql.Stmt
}

// Returns a list of values for query parameters
func (pstmt *Pstmt) getValues(params map[string]interface{}) ([]interface{}, error) {
	// number of parameters
	num := len(pstmt.params)

	if params == nil {
		// if params = nil
		if num == 0 {
			// OK if query has no parameters
			return nil, nil
		} else {
			// error if query has parameters
			return nil, errors.New("dbhelper: values for all parameters are missing")
		}
	}

	// slice containing values
	values := make([]interface{}, num, num)

	// fill values in correct order
	for i, p := range pstmt.params {
		v, ok := params[p]
		if !ok {
			return nil, errors.New(fmt.Sprintf("dbhelper: value for parameter '%s' is missing", p))
		}

		values[i] = v
	}

	return values, nil
}

// Executes prepared statement with provided parameter values.
func (pstmt *Pstmt) Exec(params map[string]interface{}) (sql.Result, error) {
	// get parameter values for query
	values, err := pstmt.getValues(params)
	if err != nil {
		return nil, err
	}

	// execute query
	var res sql.Result
	if values != nil {
		res, err = pstmt.stmt.Exec(values...)
	} else {
		res, err = pstmt.stmt.Exec()
	}

	if err != nil {
		return nil, wrapError(nil)
	}

	return res, nil
}

// Executes prepared query with provided parameter values.
func (pstmt *Pstmt) Query(i interface{}, params map[string]interface{}) (bool, error) {
	var err error
	returnSlice := false
	returnStruct := false

	// get pointer to slice value
	slicePtrValue := reflect.ValueOf(i)
	slicePtrType := slicePtrValue.Type()

	if slicePtrType.Kind() != reflect.Ptr {
		return false, errors.New("dbhelper: pointer expected")
	}

	// get slice value
	sliceValue := slicePtrValue.Elem()

	// get slice type
	sliceType := sliceValue.Type()

	// get return pointer type
	var returnPtrType reflect.Type
	if sliceType.Kind() == reflect.Slice {
		// return slice of pointers to structs
		returnSlice = true
		returnPtrType = sliceType.Elem()
	} else {
		// return pointer
		returnPtrType = slicePtrType
	}

	if returnPtrType.Kind() != reflect.Ptr {
		return false, errors.New("dbhelper: pointer to a slice of pointers to structures expected")
	}

	// get return type
	returnType := returnPtrType.Elem()
	if returnType.Kind() == reflect.Struct {
		returnStruct = true
	}

	// get table
	var tbl *dbTable
	if returnStruct {
		tbl, err = pstmt.dbHelper.getTable(returnType)
		if err != nil {
			return false, err
		}
	}

	// get parameter values for query
	values, err := pstmt.getValues(params)
	if err != nil {
		return false, err
	}

	// perform query
	var rows *sql.Rows
	if values != nil {
		rows, err = pstmt.stmt.Query(values...)
	} else {
		rows, err = pstmt.stmt.Query()
	}

	if err != nil {
		return false, wrapError(err)
	}

	// close rows on exit
	defer rows.Close()

	// create slice
	if returnSlice {
		sliceValue.Set(reflect.MakeSlice(sliceType, 0, 10))
	}

	// get column names
	columns, err := rows.Columns()
	if err != nil {
		return false, wrapError(err)
	}

	// read rows data to structures
	dataRead := false
	for rows.Next() {
		// create new structure and get a pointer to it
		var returnPtrValue reflect.Value
		if returnSlice {
			returnPtrValue = reflect.New(returnType)
		} else {
			returnPtrValue = slicePtrValue
		}

		// get new structure value
		returnValue := returnPtrValue.Elem()

		if returnStruct {
			// slice containing pointers to corresponding fields of the structure
			fields := make([]interface{}, tbl.numField, tbl.numField)

			// fill slice with pointers
			for i, col := range columns {
				// get field in new structure
				v := returnValue.FieldByIndex(tbl.fields[col].index)

				// append pointer to field to slice
				fields[i] = v.Addr().Interface()
			}

			// scan row and assign values to struct fields
			err = rows.Scan(fields...)
		} else {
			// scan row and assign return value
			err = rows.Scan(returnValue.Addr().Interface())
		}

		// check scan error
		if err != nil {
			return false, wrapError(err)
		}

		dataRead = true

		if returnSlice {
			// append pointer to new structure to slice
			sliceValue.Set(reflect.Append(sliceValue, returnPtrValue))
		} else {
			break
		}
	}

	return dataRead, nil
}
