package dynsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"regexp"
)

var (
	InsertJSONStatementHeads = []string{"INSERT INTO", "Insert Into", "insert into"}
	InsertJSONStatementRegex = regexp.MustCompile(`^(INSERT INTO|Insert Into|insert into)\s*(?P<TABLE>[\w\d]+)\s*(JSON|Json|json)\s*(?P<JSON>.*)\s*(\;*)$`)
)

type lockMode int

const (
	missLock lockMode = iota
	readLock
	writeLock
)

func obtainMatches(search []string, query string, r *regexp.Regexp) []string {
	values := r.FindStringSubmatch(query)
	keys := r.SubexpNames()
	matches := map[string]string{}
	for i, key := range keys {
		matches[key] = values[i]
	}

	data := []string{}
	for _, s := range search {
		data = append(data, matches[s])
	}

	return data
}

func compareRequiredKeys(req []string, exists map[string]string) []string {
	missing := []string{}
	for _, r := range req {
		_, ok := exists[r]
		if !ok {
			missing = append(missing, r)
		}
	}
	return missing
}

func newConn(d *Driver, dc driver.Conn) (*Conn, error) {
	c := &Conn{
		d: d,
		c: dc,
	}

	if execer, ok := dc.(driver.Execer); ok {
		c.execer = execer
	}
	if execerCtx, ok := dc.(driver.ExecerContext); ok {
		c.execerCtx = execerCtx
	}
	if queryer, ok := dc.(driver.Queryer); ok {
		c.queryer = queryer
	}
	if queryerCtx, ok := dc.(driver.QueryerContext); ok {
		c.queryerCtx = queryerCtx
	}

	c.d.mu.Lock()
	defer c.d.mu.Unlock()

	if c.d.schemas == nil {
		err := c.pollDatabaseData(false)
		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

type Conn struct {
	d *Driver

	c driver.Conn

	execer    driver.Execer
	execerCtx driver.ExecerContext

	queryer    driver.Queryer
	queryerCtx driver.QueryerContext
}

func (c *Conn) toExecerQueryerPreparer() ExecerQueryerPreparer {
	return &connExecerQueryerPreparer{c}
}

func (c *Conn) doExec(query string, stmt driver.Stmt, args []driver.Value) (driver.Result, error) {
	// log.Printf("DoExec: %s with %+v\n", query, args)
	var err error
	if stmt == nil {
		stmt, err = c.c.Prepare(query)
		if err != nil {
			return nil, err
		}
	}
	return stmt.Exec(args)
}

func (c *Conn) doQuery(query string, args []driver.Value) (driver.Rows, error) {
	// log.Printf("DoQuery: %s with %+v\n", query, args)
	stmt, err := c.c.Prepare(query)
	if err != nil {
		return nil, err
	}
	return stmt.Query(args)
}

func (c *Conn) isExecJSONInsertOnTable(query string) (string, string, bool) {
	for _, head := range InsertJSONStatementHeads {
		if len(query) < len(head) {
			continue
		}
		if query[:len(head)] != head {
			continue
		}

		matches := obtainMatches([]string{"TABLE", "JSON"}, query, InsertJSONStatementRegex)
		tableName, jsonPayload := matches[0], matches[1]
		if jsonPayload[len(jsonPayload)-1] == ';' {
			jsonPayload = jsonPayload[:len(jsonPayload)-1]
		}
		return tableName, jsonPayload, true
	}
	return "", "", false
}

func (c *Conn) pollDatabaseData(lockDriver bool) error {
	if lockDriver {
		c.d.mu.Lock()
		defer c.d.mu.Unlock()
	}

	tableNames, err := c.d.SQL.GetAllTableNames(c.toExecerQueryerPreparer())
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	schemas := map[string]map[string]string{}
	if err != sql.ErrNoRows {
		for _, name := range tableNames {
			schemas[name] = map[string]string{}
		}
		for tableName := range schemas {
			schemas[tableName], err = c.d.SQL.GetAllTableColumns(tableName, c.toExecerQueryerPreparer())
			if err != nil {
				return err
			}
		}
	}
	c.d.schemas = schemas

	return nil
}

func (c *Conn) prepareDatabase(lockDriverMode lockMode, tableName string, requiredKeys []string) (err error) {
	if lockDriverMode == missLock {
		c.d.mu.RLock()
		defer c.d.mu.RUnlock()
	}

	existingKeys, ok := c.d.schemas[tableName]
	missingKeys := []string{}
	if ok {
		missingKeys = compareRequiredKeys(requiredKeys, existingKeys)
	}
TEST_SLOW_PATH:
	if !ok || len(missingKeys) > 0 {

		// Slow Path: Switch Lock Type, then test again
		if lockDriverMode != writeLock {
			c.d.mu.RUnlock()
			c.d.mu.Lock()
		}
		existingKeys, ok = c.d.schemas[tableName]
		if ok {
			missingKeys = compareRequiredKeys(requiredKeys, existingKeys)
		}
		if ok && len(missingKeys) == 0 {

			// somehow during lock switch there was an alteration
			// no update to table schema required. Switch Lock Type
			// and proceed with Fast Path
			if lockDriverMode != writeLock {
				c.d.mu.Unlock()
				c.d.mu.RLock()
			}
			goto TEST_SLOW_PATH
		}

		if lockDriverMode != writeLock {
			defer c.d.mu.RLock()
			defer c.d.mu.Unlock()
		}

		var tx driver.Tx
		tx, err = c.Begin()
		if err != nil {
			return
		}

		if !ok {
			// Create Table
			err = c.d.SQL.CreateNewTable(tableName, requiredKeys, c.toExecerQueryerPreparer())
			if err != nil {
				return
			}
		} else {
			// Add missing column
			for _, k := range missingKeys {
				err = c.d.SQL.AddColumnToTable(tableName, k, c.toExecerQueryerPreparer())
				if err != nil {
					return
				}
			}
		}
		err = tx.Commit()
		if err != nil {
			return
		}

		existingKeys, err = c.d.SQL.GetAllTableColumns(tableName, c.toExecerQueryerPreparer())
		if err != nil {
			return
		}
		c.d.schemas[tableName] = existingKeys
	}

	return
}

func (c *Conn) prepare(query, tableName, jsonPayload string, isExecJSONStatement bool) (driver.Stmt, error) {
	if isExecJSONStatement {
		data := map[string]interface{}{}
		err := json.Unmarshal([]byte(jsonPayload), &data)
		if err != nil {
			return nil, err
		}

		values := []driver.Value{}
		requiredKeys := []string{}

		i := len(values)
		for k, v := range data {
			requiredKeys = append(requiredKeys, k)
			values = append(values, v)
			i++
		}

		c.d.mu.RLock()
		defer c.d.mu.RUnlock()
		err = c.prepareDatabase(readLock, tableName, requiredKeys)
		if err != nil {
			return nil, err
		}

		stmt, err := c.d.SQL.InsertValuesPrepare(tableName, requiredKeys, c.toExecerQueryerPreparer())
		if err != nil {
			return nil, err
		}

		return newStmt(stmt, values), err
	}
	return c.c.Prepare(query)
}

func (c *Conn) Exec(query string, args []driver.Value) (driver.Result, error) {
	tableName, jsonPayload, isExecJSONStatement := c.isExecJSONInsertOnTable(query)
	var err error
	var stmt driver.Stmt
	if isExecJSONStatement {
		stmt, err = c.prepare(query, tableName, jsonPayload, isExecJSONStatement)
		if err != nil {
			return nil, err
		}
	}

	if c.execer != nil && stmt == nil {
		return c.execer.Exec(query, args)
	}

	return c.doExec(query, stmt, args)
}

func (c *Conn) Query(query string, args []driver.Value) (driver.Rows, error) {
	if c.queryer != nil {
		return c.queryer.Query(query, args)
	}
	return c.doQuery(query, args)
}

func (c *Conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	return c.Prepare(query)
}

func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	tableName, jsonPayload, isExecJSONStatement := c.isExecJSONInsertOnTable(query)
	return c.prepare(query, tableName, jsonPayload, isExecJSONStatement)
}

func (c *Conn) Close() error {
	return c.c.Close()
}

func (c *Conn) Begin() (driver.Tx, error) {
	return c.c.Begin()
}

type connExecerQueryerPreparer struct {
	c *Conn
}

func (c *connExecerQueryerPreparer) Prepare(query string) (driver.Stmt, error) {
	// log.Printf("Prepare: %+v\n", query)
	return c.c.c.Prepare(query)
}

func (c *connExecerQueryerPreparer) Exec(query string, args []driver.Value) (driver.Result, error) {
	if c.c.execer != nil {
		// log.Printf("execer.Exec: %s with %+v\n", query, args)
		return c.c.execer.Exec(query, args)
	}
	return c.c.doExec(query, nil, args)
}

func (c *connExecerQueryerPreparer) Query(query string, args []driver.Value) (driver.Rows, error) {
	if c.c.queryer != nil {
		// log.Printf("queryer.Query: %s with %+v\n", query, args)
		return c.c.queryer.Query(query, args)
	}
	return c.c.doQuery(query, args)
}
