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
	"regexp"
	"strings"
	"time"
)

var (
	paramRegexp *regexp.Regexp
	errorNil    = errors.New("dbhelper: cannot use nil to define type")
)

func init() {
	paramRegexp = regexp.MustCompile(`:[^,\s)]*`)
}

func typeOf(i interface{}) (reflect.Type, error) {
	if i == nil {
		return nil, errorNil
	}

	t := reflect.TypeOf(i)
	if t.Kind() != reflect.Ptr {
		return t, nil
	}

	return t.Elem(), nil
}

func wrapError(err error) error {
	return errors.New(fmt.Sprintf("dbhelper: %v", err))
}

func checkFieldType(t reflect.Type) bool {
	kind := t.Kind()
	return kind == reflect.String ||
		kind == reflect.Int ||
		kind == reflect.Int8 ||
		kind == reflect.Int16 ||
		kind == reflect.Int32 ||
		kind == reflect.Int64 ||
		kind == reflect.Float32 ||
		kind == reflect.Float64 ||
		kind == reflect.Bool
}

// DbHelper contains all data about database and tables.
type DbHelper struct {
	// Pointer to underlying sql.DB.
	Db *sql.DB

	sqlDialect SqlDialect
	tables     map[reflect.Type]*dbTable
}

// New returns new DbHelper.
func New(db *sql.DB, sqlDialect SqlDialect) *DbHelper {
	return &DbHelper{
		Db:         db,
		sqlDialect: sqlDialect,
		tables:     make(map[reflect.Type]*dbTable),
	}
}

// AddTable adds a connection between type of i and table name.
// There is no difference what to use, type or pointer to type.
func (dbh *DbHelper) AddTable(i interface{}, name string) error {
	t, err := typeOf(i)
	if err != nil {
		return err
	}

	tbl, ok := dbh.tables[t]
	if ok {
		return errors.New(fmt.Sprintf("dbhelper: type '%v' already has assigned table name '%s'", t, tbl.name))
	}

	if name == "" {
		return errors.New("dbhelper: table name cannot be an empty string")
	}

	tbl, err = dbh.newDbTable(t, name)
	if err != nil {
		return err
	}

	dbh.tables[t] = tbl

	return nil
}

// RemoveTable removes a connection between type of i and table name assigned to it.
// Returns true if connection was removed and false if there were no connection or if i is nil.
func (dbh *DbHelper) RemoveTable(i interface{}) bool {
	if i == nil {
		return false
	}

	t, err := typeOf(i)
	if err != nil {
		return false
	}

	_, ok := dbh.tables[t]
	if ok {
		delete(dbh.tables, t)
		return true
	}

	return false
}

func (dbh *DbHelper) getTable(t reflect.Type) (*dbTable, error) {
	tbl, ok := dbh.tables[t]
	if !ok {
		return nil, errors.New(fmt.Sprintf("dbhelper: type '%v' has no assigned table", t))
	}

	return tbl, nil
}

func (dbh *DbHelper) getPlaceholders(n int) []string {
	a := make([]string, n, n)
	ph := dbh.sqlDialect.placeholder()
	for i := 1; i < n; i++ {
		a[i] = ph.next()
	}

	return a
}

// Prepares SQL query. Prepared query can be executed with different parameter values.
func (dbh *DbHelper) Prepare(query string) (*Pstmt, error) {
	ph := dbh.sqlDialect.placeholder()
	params := paramRegexp.FindAllString(query, -1)
	for i, p := range params {
		if len(p) < 2 {
			return nil, errors.New(fmt.Sprintf("dbhelper: wrong parameter placeholder: '%s'", p))
		}

		// replaced named parameter with placeholder
		query = strings.Replace(query, p, ph.next(), 1)

		// store named parameter
		params[i] = p[1:]
	}

	// prepare query
	stmt, err := dbh.Db.Prepare(query)
	if err != nil {
		return nil, wrapError(err)
	}

	pstmp := &Pstmt{
		dbHelper: dbh,
		params:   params,
		stmt:     stmt,
	}

	return pstmp, nil
}

// Prepares standard select SQL query by value of one column.
// Prepared query can be executed with different parameter values.
func (dbh *DbHelper) PrepareSelect(i interface{}, column string) (*Pstmt, error) {
	// get type
	t, err := typeOf(i)
	if err != nil {
		return nil, err
	}

	// get table
	tbl, err := dbh.getTable(t)
	if err != nil {
		return nil, err
	}

	// check column name
	_, ok := tbl.fields[column]
	if !ok {
		return nil, errors.New(fmt.Sprintf("dbhelper: structure type '%v' has no field assigned to column '%s' of table '%s'",
			t, column, tbl.name))
	}

	// select query
	query := fmt.Sprintf("SELECT * FORM %s WHERE %s = :%s", tbl.name, column, column)

	// prepare query
	return dbh.Prepare(query)
}

// Prepares parameters for standard query.
func (dbh *DbHelper) prepareParams(i interface{}) (tbl *dbTable, params map[string]interface{}, v reflect.Value, err error) {
	// get structure type
	t, err := typeOf(i)
	if err != nil {
		return
	}

	// get table
	tbl, err = dbh.getTable(t)
	if err != nil {
		return
	}

	// get value of structure to insert
	v = reflect.ValueOf(i)
	if v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// get parameter values
	l := len(tbl.insertQuery.params)
	params = make(map[string]interface{}, l)
	for _, f := range tbl.fields {
		params[f.column] = v.FieldByIndex(f.index).Interface()
	}

	return
}

// Inserts new record to databse. Field with option 'id' is automatically updated.
func (dbh *DbHelper) Insert(i interface{}) error {
	// get current timestamp
	time := time.Now().UTC().Unix()

	// prepare parameters
	tbl, params, v, err := dbh.prepareParams(i)
	if err != nil {
		return err
	}

	// set created time
	if tbl.createdField != nil {
		params[tbl.createdField.column] = time
	}

	// set modified time
	if tbl.modifiedField != nil {
		params[tbl.modifiedField.column] = time
	}

	var id int64
	if sqld, ok := dbh.sqlDialect.(hasCustomInsert); ok {
		// custom insert
		id, err = sqld.insert(tbl, params)
		if err != nil {
			return err
		}
	} else {
		// standart insert
		res, err := tbl.insertQuery.exec(params)
		if err != nil {
			return err
		}

		// get last inserted id
		id, err = res.LastInsertId()
		if err != nil {
			return nil
		}
	}

	// udpate id field in structure
	v.FieldByIndex(tbl.idField.index).SetInt(id)

	// update created field in structure
	if tbl.createdField != nil {
		v.FieldByIndex(tbl.createdField.index).SetInt(time)
	}

	// update modified field in structure
	if tbl.modifiedField != nil {
		v.FieldByIndex(tbl.modifiedField.index).SetInt(time)
	}

	return nil
}

// Updates record(s) in database and returns number of affected rows.
// Field with option 'id' is used to define the record in database.
// This means that field with option 'id' cannot be updated.
func (dbh *DbHelper) Update(i interface{}) (int64, error) {
	// get current timestamp
	time := time.Now().UTC().Unix()

	// prepare parameters
	tbl, params, v, err := dbh.prepareParams(i)
	if err != nil {
		return 0, err
	}

	// set modified time
	if tbl.modifiedField != nil {
		params[tbl.modifiedField.column] = time
	}

	// standart update
	num, err := tbl.updateQuery.Exec(params)
	if err != nil {
		return 0, err
	}

	// update modified field in structure
	if tbl.modifiedField != nil {
		v.FieldByIndex(tbl.modifiedField.index).SetInt(time)
	}

	return num, nil
}

// Deletes record(s) in database and returns number of affected rows.
// Field with option 'id' is used to define the record in database.
func (dbh *DbHelper) Delete(i interface{}) (int64, error) {
	// prepare parameters
	tbl, params, _, err := dbh.prepareParams(i)
	if err != nil {
		return 0, err
	}

	// standart update
	num, err := tbl.deleteQuery.Exec(params)
	if err != nil {
		return 0, err
	}

	return num, nil
}
