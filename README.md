Go Database Helper
========

This is a simple Go database helper package. It is inspired by `gorp`, but uses prepared statements. It helps to interact with sql.DB by generating, preparing and executing queries. It marshals Go structs to and from databases and uses database/sql.
Queries for insert, update and delete are prepared automatically, when new table is added. Other statements can be prepared using dbhelper.Prepare(). It supports automatic update of:

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

Also `dbopt:"skip"` tag is supported and means that field will be skipped and nod mapped to database table. if `db` tag is not set - field name will be used instead.

Usage
========

```go
type someStructType struct {
  // structure must have a field with dbopt: "id"
  // this field will be automatically updated on record insertion
  Id int64 `db:"id" dbopt:"id,auto"`

  // data field
  SomeField string `db:"some_field"`

  // this field will be automatically updated on record insertion
  Created int64 `db:"created" dbopt:"created"`
  
  // this field will be automatically updated on record insertion
  // and modification
  Modified int64 `db:"modified" dbopt:"modified"`
}
```

```go
// create connection to database, check error
db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
    address, port, dbname, username, password))
defer db.Close()

// create database helper
dbh := dbhelper.New(db, dbhelper.Posgresql{})

// map type to table, check error
err = dbh.AddTable(someStructType{}, "table_name")

// insert new record, id, modified (if present) and created (if present)
// fields are automatically updated, check error
var s *someStructType
s = newStruct()
err = dbh.Insert(s)

// update record, modified field (if present) is automatically updated
s.SomeField = "new_value"
err = dbh.Update(s)

// custom select query to get all records, check errors
q1, err := dbh.Prepare("SELECT * FROM table_name")

var a []*someStructType
err = q.Query(&a, nil)

// custom select query to get record with id = 3, check errors
q2, err := dbh.Prepare("SELECT * FROM table_name WHERE id = :id")

var r someStructType
err = q.Query(&r, map[string]interface{}{
  "id": 3,
})

// custom select query to get one field of record with id = 3, check errors
q2, err := dbh.Prepare("SELECT some_field FROM table_name WHERE id = :id")

var str string
err = q.Query(&str, map[string]interface{}{
  "id": 3,
})

// delete record
err = dbh.Delete(s)
```

See tests for examples. Embedded structures should be supported as weel, but I have not tested this.

Performance
========

The main motivation to do this was performance. Prepared queries should work faster and I tried to add as small overhead as possible for the convenience of named placeholders and mapping results to structure fields.

Some benchmark results (average of 5 runs):

```
BenchmarkPreparedQueries      2000      851048 ns/op
BenchmarkDbHelper             2000      914452 ns/op
BenchmarkGorp                 1000     1409280 ns/op
```

Not sure how reliable these results are, but one can see that the overhead is quite small. The comparison to `gorp` here is not really fare, because it does not use prepared queries. However, this project was inspired by it and would make no sense if it was slower.
