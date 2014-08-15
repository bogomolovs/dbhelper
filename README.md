Go Database Helper
========

This is a simple Go database helper package. It is inspired by `gorp`, but uses prepared statements. It helps to interact with sql.DB by generating, preparing and executing queries. It marshals Go structs to and from databases and uses database/sql.
Queries for insert, update, delete and select by id are prepared automatically, when new table is added. Queries to select by one column are prepared automatically when dbhelper.SelectBy() is called first time for corresponding column. Other statements can be prepared using dbhelper.Prepare(). Following structure fields (and columns) are set automatically:

* record id (after inserting)
* created time (after inserting)
* modified time (after inserting and updating)

Supported dialects
========

It was tested only with Postgresql, but should also support MySQL and Sqlite.

Structure tags
========

Structure tags are supported:

```go
type testType struct {
  // db column name is 'id', it is auto-incremented and it stores record id
  Id   int64  `db:"id" dbopt:"id,auto"`
  
  // db column name is 'text'
  Text string `db:"text"`
  
  // db column name is 'c', this field will be automatically set with the time
  // of record creation. Time is stored as UNIX timestamp (UTC timezone)
  C    int64  `db:"c" dbopt:"created"`
  
  // db column name is 'm', this field will be automatically updated with the time
  // of last record update. Time is stored as UNIX timestamp (UTC timezone)
  M    int64  `db:"m" dbopt:"modified"`  
}
```

Also `dbopt:"skip"` tag is supported and means that field will be skipped and not mapped to database table. if `db` tag is not set - field name will be used instead.

Usage
========

```go
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
```

```go
// error checks are omitted to make this listing shorter
// see tests for complete examples

// create connection to database, check error
db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
    address, port, dbname, username, password))
defer db.Close()

// create DbHelper
dbh := New(db, Postgresql{})
err = dbh.AddTable(testStruct{}, "test")

// insert
t1 := &testStruct{}
t1.Text = "text 1"
t1.Bool = true

err = dbh.Insert(t1)

t2 := &testStruct{}
t2.Text = "text 2"
t2.Bool = false

err = dbh.Insert(t2)

// update
t1.Text = "another text"
t1.Bool = false

_, err = dbh.Update(t1)

// select all records
var allRecords []*testStruct

queryAllRecords, err := dbh.Prepare("SELECT * FROM test")
_, err = queryAllRecords.Query(&allRecords, nil)

// select first record
var firstRecord testStruct
_, err = queryAllRecords.Query(&firstRecord, nil)

// select one record with specific id
var record testStruct

queryRecordById, err := dbh.Prepare("SELECT * FROM test WHERE id = :id")
_, err = queryRecordById.Query(&record, map[string]interface{}{
  "id": t2.Id,
})

// or simplier
var record2 testStruct
_, err = queryRecordById.Query(&record2, t2.Id)

// or even simplier
var record3 testStruct
_, err = dbh.SelectById(&record3, t2.Id)

// select one record with specific field value
// on the first selection by field, query is prepared
// and stored, so next time selection by the same
// field will be performed using already prepared query
var record4 testStruct
_, err = dbh.SelectBy(&record4, "text", t1.Text)

// select one field of record with specific id
var str string
queryString, err := dbh.Prepare("SELECT text FROM test WHERE id = :id")
_, err := queryString.Query(&str, map[string]interface{}{
  "id": t1.Id,
})

// or simplier
var str2 string
_, err = queryString.Query(&str2, t1.Id)

// delete records
_, err = dbh.Delete(t1)
_, err = dbh.Delete(t2)
```

Benchmarks
========

The main motivation to do this was performance. Prepared queries should work faster and I tried to add as small overhead as possible for the convenience of named placeholders and mapping results to structure fields.

Some benchmark results (average of 5 runs):

```
go test -bench .
BenchmarkPreparedQueries      2000      851048 ns/op
BenchmarkDbHelper             2000      914452 ns/op (overhead - 7.45%)
BenchmarkGorp                 1000     1409280 ns/op (overhead - 65.6%)

go test -bench . -benchtime 10s
BenchmarkPreparedQueries      2000      932642 ns/op
BenchmarkDbHelper             2000     1011751 ns/op (overhead - 8.48%)
BenchmarkGorp                 1000     1723938 ns/op (overhead - 84.8%)
```

Not sure how reliable these results are, but one can see that the overhead is quite small. The comparison to `gorp` here is not really fare, because it does not use prepared queries. However, this project was inspired by it and would make no sense if it was slower. Ten times smaller overhead makes sense, at least for my needs.