Go Database Helper
========

This is a simple Go database/sql helper package. It is inspired by gorp, but uses prepared statements. Queries for insert, update and delete are prepared automatically when new table is added. Other statements can be prepared using dbhelper.Prepare(). It supports automatic update of:
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

Also **dbopt:"skip"** tag is supported and means that field will be skipped and nod mapped to database table.

Usage
========

```go
// create connection to database, check error
db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
    address, port, dbname, username, password))

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

See tests for examples.
