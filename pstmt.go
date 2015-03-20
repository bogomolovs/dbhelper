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
func (pstmt *Pstmt) getValues(params interface{}) ([]interface{}, error) {
	// number of parameters
	num := len(pstmt.params)

	// there are no parameters
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

	// get value of params
	paramsValue := reflect.ValueOf(params)

	// get type of params
	paramsType := paramsValue.Type()

	if paramsType.Kind() == reflect.Map {
		// fill values in correct order
		for i, p := range pstmt.params {
			// value
			v := paramsValue.MapIndex(reflect.ValueOf(p))
			if !v.IsValid() {
				return nil, errors.New(fmt.Sprintf("dbhelper: value for parameter '%s' is missing", p))
			}

			values[i] = v.Interface()
		}
	} else {
		if num > 1 {
			return nil, errors.New("dbhelper: query has more than one parameter, params must be a map[string]interface{}")
		}

		if !checkFieldType(paramsType) {
			return nil, errors.New(fmt.Sprintf("dbhelper: wrong parameter type '%v'", paramsType))
		}

		values[0] = paramsValue.Interface()
	}

	return values, nil
}

func (pstmt *Pstmt) exec(params interface{}) (sql.Result, error) {
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
		return nil, wrapError(err)
	}

	return res, nil
}

// Executes prepared statement with provided parameter values.
// If query has only one parameter, params can be the value of that parameter.
// If query has more than one parameter, params must be a map[string]interface{}.
// Returns number of affected rows or -1 if this number cannot be obtained.
func (pstmt *Pstmt) Exec(params interface{}) (int64, error) {
	// execute query
	res, err := pstmt.exec(params)
	if err != nil {
		return 0, err
	}

	// get number of affected rows
	num, err := res.RowsAffected()
	if err != nil {
		return -1, nil
	}

	return num, nil
}

// Executes prepared query with provided parameter values. Returns number of processed rows.
// If i is a pointer to slice of pointers - all rows are mapped.
// If i is a pointer to structure - only the first matched row is mapped.
// If i is a pointer to another supported data type - corresponding column value
// of the first matched row is mapped.
// If query has only one parameter, params can be the value of that parameter.
// If query has more than one parameter, params must be a map[string]interface{}.
func (pstmt *Pstmt) Query(i interface{}, params interface{}) (int64, error) {
	if i == nil {
		return 0, errorNil
	}

	var err error
	returnSlice := false
	returnStruct := false

	// get pointer to slice value
	slicePtrValue := reflect.ValueOf(i)
	slicePtrType := slicePtrValue.Type()

	if slicePtrType.Kind() != reflect.Ptr {
		return 0, errors.New("dbhelper: pointer expected")
	}

	// get slice value
	sliceValue := slicePtrValue.Elem()
	if !sliceValue.IsValid() {
		return 0, errors.New("dbhelper: cannot use pointer to nil")
	}

	// get slice type
	sliceType := sliceValue.Type()
	if sliceType.Kind() == reflect.Ptr {
		return 0, errors.New("dbhelper: cannot use pointer to pointer")
	}

	if sliceType.Kind() == reflect.Interface {
		return 0, errors.New("dbhelper: wrong type of i")
	}

	// get return pointer type
	var returnPtrType reflect.Type
	if sliceType.Kind() == reflect.Slice {
		// return slice of pointers to structs
		returnSlice = true
		returnPtrType = sliceType.Elem()

		if returnPtrType.Kind() != reflect.Ptr {
			return 0, errors.New("dbhelper: pointer to a slice of pointers expected")
		}
	} else {
		// return pointer
		returnPtrType = slicePtrType
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
			return 0, err
		}
	}

	// get parameter values for query
	values, err := pstmt.getValues(params)
	if err != nil {
		return 0, err
	}

	// perform query
	var rows *sql.Rows
	if values != nil {
		rows, err = pstmt.stmt.Query(values...)
	} else {
		rows, err = pstmt.stmt.Query()
	}

	if err != nil {
		return 0, wrapError(err)
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
		return 0, wrapError(err)
	}

	// read rows data to structures
	num := int64(0)
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
			return 0, wrapError(err)
		}

		num++

		if returnSlice {
			// append pointer to slice
			sliceValue.Set(reflect.Append(sliceValue, returnPtrValue))
		} else {
			break
		}
	}

	return num, nil
}
