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
	"fmt"
	"testing"

	_ "github.com/lib/pq"
	// "time"
)

type testEmbeddedStruct struct {
	// data field
	Text string `db:"text"`
}

type testStruct struct {
	// structure must have a field with dbopt: "id"
	// this field will be automatically updated on record insertion
	Id int64 `db:"id" dbopt:"id,auto"`

	// data field
	Bool bool `db:"b"`

	// this field will be automatically updated on record insertion
	Created int64 `db:"c" dbopt:"created"`

	// this field will be automatically updated on record insertion
	// and modification
	Modified int64 `db:"m" dbopt:"modified"`

	// embedded structures are supported
	testEmbeddedStruct
}

func initDb() (*sql.DB, error) {
	return sql.Open("postgres", fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		"localhost", 5432, "test", "test", "test"))
}

func TestQuery(t *testing.T) {
	// create connection to DB
	db, err := initDb()
	if err != nil {
		t.Error(err)
		return
	}

	// create DbHelper
	dbh := New(db, Postgresql{})
	err = dbh.AddTable(testStruct{}, "test")
	if err != nil {
		t.Error(err)
		return
	}

	// insert
	t1 := &testStruct{}
	t1.Text = "text 1"
	t1.Bool = true

	err = dbh.Insert(t1)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Printf("First record id: %d\n\n", t1.Id)

	// time.Sleep(1 * time.Second)

	t2 := &testStruct{}
	t2.Text = "text 2"
	t2.Bool = false

	err = dbh.Insert(t2)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Printf("Second record id: %d\n\n", t2.Id)

	// time.Sleep(2 * time.Second)

	// update
	t1.Text = "another text"
	t1.Bool = false

	_, err = dbh.Update(t1)
	if err != nil {
		t.Error(err)
		return
	}

	// select all records
	queryAllRecords, err := dbh.Prepare("SELECT * FROM test")
	if err != nil {
		t.Error(err)
		return
	}

	var allRecords []*testStruct
	_, err = queryAllRecords.Query(&allRecords, nil)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println("Select all records:")
	for _, r := range allRecords {
		fmt.Println(*r)
	}

	fmt.Println()

	// or simplier
	var allRecords2 []*testStruct
	_, err = dbh.SelectAll(&allRecords2)

	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println("Select all records (simplier):")
	for _, r := range allRecords {
		fmt.Println(*r)
	}

	fmt.Println()

	// select first record
	var firstRecord testStruct
	_, err = queryAllRecords.Query(&firstRecord, nil)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Printf("Select first record:\n%v\n\n", firstRecord)

	// select one record with specific id
	queryRecordById, err := dbh.Prepare("SELECT * FROM test WHERE id = :id")
	if err != nil {
		t.Error(err)
		return
	}

	var record testStruct
	_, err = queryRecordById.Query(&record, map[string]interface{}{
		"id": t2.Id,
	})
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Printf("Select one record by id:\n%v\n\n", record)

	// or simplier
	var record2 testStruct
	_, err = queryRecordById.Query(&record2, t2.Id)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Printf("Select one record by id (simplier):\n%v\n\n", record2)

	// or even simplier
	var record3 testStruct
	_, err = dbh.SelectById(&record3, t2.Id)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Printf("Select one record by id (even simplier):\n%v\n\n", record3)

	// select one record with specific field value
	// on the first selection by field, query is prepared
	// and stored, so next time selection by the same
	// field will be performed using already prepared query
	var record4 testStruct
	_, err = dbh.SelectBy(&record4, "text", t1.Text)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Printf("Select one record by field:\n%v\n\n", record4)

	// select one field of record with specific id
	queryString, err := dbh.Prepare("SELECT text FROM test WHERE id = :id")
	if err != nil {
		t.Error(err)
		return
	}

	var str string
	num, err := queryString.Query(&str, map[string]interface{}{
		"id": t1.Id,
	})
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println("Select one field:")
	fmt.Printf("Number of fields processed: %d\n", num)
	fmt.Printf("Field value: %s\n\n", str)

	// or simplier
	var str2 string
	num, err = queryString.Query(&str2, t1.Id)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println("Select one field (simplier):")
	fmt.Printf("Number of fields processed: %d\n", num)
	fmt.Printf("Field value: %s\n\n", str2)

	// delete records
	_, err = dbh.Delete(t1)
	if err != nil {
		t.Error(err)
		return
	}

	_, err = dbh.Delete(t2)
	if err != nil {
		t.Error(err)
		return
	}
}
