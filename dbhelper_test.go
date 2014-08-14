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
	_ "github.com/lib/pq"
	"testing"
	// "time"
)

type testEmbedded struct {
	T string `db:"text"`
}

type testType struct {
	Id int64 `db:"id" dbopt:"id,auto"`
	B  bool  `db:"b"`
	C  int64 `db:"c" dbopt:"created"`
	M  int64 `db:"m" dbopt:"modified"`
	testEmbedded
}

func initDb() (*sql.DB, error) {
	return sql.Open("postgres", fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		"localhost", 5432, "test", "test", "test"))
}

func TestQuery(t *testing.T) {
	db, err := initDb()
	if err != nil {
		t.Error(err)
		return
	}

	dbh := New(db, Postgresql{})
	err = dbh.AddTable(testType{}, "test")
	if err != nil {
		t.Error(err)
		return
	}

	// insert
	t1 := &testType{testEmbedded: testEmbedded{T: "test1"}, B: true}
	err = dbh.Insert(t1)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(t1.Id)

	// time.Sleep(1 * time.Second)

	t2 := &testType{testEmbedded: testEmbedded{T: "test2"}, B: false}
	err = dbh.Insert(t2)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(t2.Id)

	// time.Sleep(2 * time.Second)

	// update
	t1.T = "another text"
	t1.B = false
	_, err = dbh.Update(t1)
	if err != nil {
		t.Error(err)
		return
	}

	// select
	q, err := dbh.Prepare("SELECT * FROM test")
	if err != nil {
		t.Error(err)
		return
	}

	var res []*testType
	_, err = q.Query(&res, nil)
	if err != nil {
		t.Error(err)
		return
	}

	for _, r := range res {
		fmt.Println(*r)
	}

	// select one record
	var res1 testType
	_, err = q.Query(&res1, nil)
	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(res1)

	// select one field
	queryString, err := dbh.Prepare("SELECT text FROM test WHERE id = :id")
	if err != nil {
		t.Error(err)
		return
	}

	var s string
	num, err := queryString.Query(&s, map[string]interface{}{
		"id": t1.Id,
	})

	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(num)
	fmt.Println(s)

	// or simplier
	var s2 string
	num, err = queryString.Query(&s2, t1.Id)

	if err != nil {
		t.Error(err)
		return
	}

	fmt.Println(num)
	fmt.Println(s2)

	// delete
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
