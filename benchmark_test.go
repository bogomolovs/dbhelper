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
	"github.com/coopernurse/gorp"
	_ "github.com/lib/pq"
	"testing"
)

func BenchmarkPreparedQueries(b *testing.B) {
	db, err := initDb()
	if err != nil {
		b.Error(err)
		return
	}

	defer db.Close()

	queryInsert, err := db.Prepare("INSERT INTO test (text, b, c, m) VALUES ($1, $2, $3, $4) RETURNING id")
	if err != nil {
		b.Error(err)
		return
	}

	queryUpdate, err := db.Prepare("UPDATE test SET text = $1, b = $2, m = $3")
	if err != nil {
		b.Error(err)
		return
	}

	queryDelete, err := db.Prepare("DELETE FROM test WHERE id = $1")
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// insert
		t1 := &testType{testEmbedded: testEmbedded{T: "test1"}, B: true}
		var id int64
		err = queryInsert.QueryRow(t1.T, t1.B, t1.C, t1.M).Scan(&id)
		if err != nil {
			b.Error(err)
			return
		}

		t1.Id = id

		// update
		t1.T = "new text"
		t1.B = false
		_, err = queryUpdate.Exec(t1.T, t1.B, t1.M)
		if err != nil {
			b.Error(err)
			return
		}

		// delete
		_, err = queryDelete.Exec(t1.Id)
		if err != nil {
			b.Error(err)
			return
		}
	}
}

func BenchmarkDbHelper(b *testing.B) {
	db, err := initDb()
	if err != nil {
		b.Error(err)
		return
	}

	defer db.Close()

	dbh := New(db, Postgresql{})
	err = dbh.AddTable(testType{}, "test")
	if err != nil {
		b.Error(err)
		return
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// insert
		t1 := &testType{testEmbedded: testEmbedded{T: "test1"}, B: true}
		err = dbh.Insert(t1)
		if err != nil {
			b.Error(err)
			return
		}

		// update
		t1.T = "new text"
		t1.B = false
		_, err = dbh.Update(t1)
		if err != nil {
			b.Error(err)
			return
		}

		// delete
		_, err = dbh.Delete(t1)
		if err != nil {
			b.Error(err)
			return
		}
	}
}

func BenchmarkGorp(b *testing.B) {
	db, err := initDb()
	if err != nil {
		b.Error(err)
		return
	}

	defer db.Close()

	dbmap := &gorp.DbMap{Db: db, Dialect: gorp.PostgresDialect{}}
	dbmap.AddTableWithName(testType{}, "test").SetKeys(true, "Id")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// insert
		t1 := &testType{testEmbedded: testEmbedded{T: "test1"}, B: true}
		err = dbmap.Insert(t1)
		if err != nil {
			b.Error(err)
			return
		}

		// update
		t1.T = "new text"
		t1.B = false
		_, err = dbmap.Update(t1)
		if err != nil {
			b.Error(err)
			return
		}

		// delete
		_, err = dbmap.Delete(t1)
		if err != nil {
			b.Error(err)
			return
		}
	}
}
