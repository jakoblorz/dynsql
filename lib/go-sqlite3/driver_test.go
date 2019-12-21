package sqlite3

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"testing"

	"github.com/jakoblorz/dynsql"

	sqlite3 "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

var (
	testJSON_s string
	testJSON_b []byte
	testJSON_m = map[string]interface{}{
		"name": "test_user",
		"age":  6,
	}
)

func init() {
	var err error
	testJSON_b, err = json.Marshal(testJSON_m)
	if err != nil {
		log.Fatal(err)
	}
	testJSON_s = string(testJSON_b)
}

func wrapRawConn(c driver.Conn) dynsql.ExecerQueryerPreparer {
	return &rawSQLConnWrapper{c}
}

type rawSQLConnWrapper struct {
	c driver.Conn
}

func (r *rawSQLConnWrapper) Prepare(query string) (driver.Stmt, error) {
	return r.c.Prepare(query)
}

func (r *rawSQLConnWrapper) Exec(query string, args []driver.Value) (driver.Result, error) {
	stmt, err := r.c.Prepare(query)
	if err != nil {
		return nil, err
	}
	return stmt.Exec(args)
}

func (r *rawSQLConnWrapper) Query(query string, args []driver.Value) (driver.Rows, error) {
	stmt, err := r.c.Prepare(query)
	if err != nil {
		return nil, err
	}
	return stmt.Query(args)
}

func wrapOnConnExtractor(baseDriver driver.Driver, sql dynsql.SQLDialect) *connExtractor {
	return &connExtractor{
		Driver: *dynsql.WrapDriver(baseDriver, sql),
	}
}

type connExtractor struct {
	dynsql.Driver
	conn driver.Conn
}

func (c *connExtractor) getLatestConn() driver.Conn {
	return c.conn
}

func (c *connExtractor) Open(name string) (driver.Conn, error) {
	var err error
	c.conn, err = c.Driver.Open(name)
	if err != nil {
		return nil, err
	}
	return c.conn, nil
}

func Test(t *testing.T) {
	d := wrapOnConnExtractor(&sqlite3.SQLiteDriver{}, SQLite3Dialect(0))
	sql.Register("dyn-test-sqlite3", d)

	db, err := sql.Open("dyn-test-sqlite3", "")
	if !assert.NoError(t, err, "expected no error opening transient in-memory database") {
		t.FailNow()
	}
	defer db.Close()

	tableName := "test"
	log.Printf("%s\n", testJSON_s)
	_, err = db.Exec(fmt.Sprintf("INSERT INTO %s JSON %s;", tableName, testJSON_s))
	if !assert.NoError(t, err, "expected no error inserting custom JSON payload: %+v", testJSON_m) {
		t.FailNow()
	}

	// r, err := db.Query("SELECT name FROM sqlite_master;")
	// if !assert.NoError(t, err) {
	// 	t.FailNow()
	// }
	// var dest interface{}
	// r.Next()
	// for err = r.Scan(&dest); err == nil && r.Next(); err = r.Scan(&dest) {
	// 	log.Printf("%+v\n", dest)
	// }
	// r.Close()
	// if !assert.NoError(t, err) {
	// 	t.FailNow()
	// }

	dialect := SQLite3Dialect(0)
	tables, err := dialect.GetAllTableNames(wrapRawConn(d.getLatestConn()))
	if !assert.NoError(t, err, "expected no error when listing all tables") {
		t.FailNow()
	}
	if !assert.Contains(t, tables, tableName, "expected table list to contain table %s", tableName) {
		t.FailNow()
	}
	columns, err := dialect.GetAllTableColumns(tableName, wrapRawConn(d.getLatestConn()))
	if !assert.NoError(t, err, "expected no error when listing all columns of table %s", tableName) {
		t.FailNow()
	}
	for k := range testJSON_m {
		if !assert.Contains(t, columns, k) {
			t.FailNow()
		}
		if k == "age" && !assert.Equal(t, "NUMBER", columns[k]) {
			t.FailNow()
		}
		if k == "name" && !assert.Equal(t, "TEXT", columns[k]) {
			t.FailNow()
		}
	}
}
