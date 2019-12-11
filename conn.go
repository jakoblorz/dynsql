package dynsql

import (
	"database/sql/driver"
	"encoding/json"
	"regexp"
	"time"

	uuid "github.com/satori/go.uuid"
)

var (
	InsertJSONStatementHeads = []string{"INSERT INTO", "Insert Into", "insert into"}
	InsertJSONStatementRegex = regexp.MustCompile(`^(INSERT INTO|Insert Into|insert into)\s*(?P<TABLE>[\w\d]+)\s*(JSON|Json|json)\s*(?P<JSON>.*)\s*(\;*)$`)

	InheritedIDField         = "_id"
	InheritedInsertedAtField = "_inserted_at"
	InheritedUpdatedAtField  = "_updated_at"
	InheritedFields          = []string{
		InheritedIDField,
		InheritedInsertedAtField,
		InheritedUpdatedAtField,
	}
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
	c := &Conn{d, dc}

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
}

func (c *Conn) toExecerQueryerPreparer() ExecerQueryerPreparer {
	return &connExecerQueryerPreparer{c}
}

func (c *Conn) DoExec(query string, args ...driver.Value) (driver.Result, error) {
	stmt, err := c.c.Prepare(query)
	if err != nil {
		return nil, err
	}
	return stmt.Exec(args)
}

func (c *Conn) DoQuery(query string, args ...driver.Value) (driver.Rows, error) {
	stmt, err := c.c.Prepare(query)
	if err != nil {
		return nil, err
	}
	return stmt.Query(args)
}

func (c *Conn) pollDatabaseData(lockDriver bool) error {
	if lockDriver {
		c.d.mu.Lock()
		defer c.d.mu.Unlock()
	}

	tableNames, err := c.d.SQL.GetAllTableNames(c.toExecerQueryerPreparer())
	if err != nil {
		return err
	}

	schemas := map[string]map[string]string{}
	for _, name := range tableNames {
		schemas[name] = map[string]string{}
	}
	for tableName := range schemas {
		schemas[tableName], err = c.d.SQL.GetAllTableColumns(tableName, c.toExecerQueryerPreparer())
		if err != nil {
			return err
		}
	}
	c.d.schemas = schemas

	return nil
}

func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	for _, head := range InsertJSONStatementHeads {
		if len(query) < len(head) {
			continue
		}
		if query[:len(head)] != head {
			continue
		}

		matches := obtainMatches([]string{"TABLE", "JSON"}, query, InsertJSONStatementRegex)
		tableName, jsonPayload := matches[0], matches[1]

		data := map[string]interface{}{}
		err := json.Unmarshal([]byte(jsonPayload), &data)
		if err != nil {
			return nil, err
		}

		id, _ := uuid.NewV4()
		values := []driver.Value{id.String(), time.Now(), time.Now()}
		requiredKeys := InheritedFields

		i := len(values) + 1
		for k, v := range data {
			requiredKeys = append(requiredKeys, k)
			values = append(values, v)
			i++
		}

		// Fast Path
		c.d.mu.RLock()
		existingKeys, ok := c.d.schemas[tableName]
		missingKeys := []string{}
		if ok {
			missingKeys = compareRequiredKeys(requiredKeys, existingKeys)
		}
	TEST_SLOW_PATH:
		if !ok || len(missingKeys) > 0 {

			// Slow Path: Switch Lock Type, then test again
			c.d.mu.RUnlock()
			c.d.mu.Lock()
			existingKeys, ok = c.d.schemas[tableName]
			if ok {
				missingKeys = compareRequiredKeys(requiredKeys, existingKeys)
			}
			if ok && len(missingKeys) == 0 {
				// somehow during lock switch there was an alteration
				// no update to table schema required. Switch Lock Type
				// and proceed with Fast Path
				c.d.mu.Unlock()
				c.d.mu.RLock()
				goto TEST_SLOW_PATH
			}
			defer c.d.mu.Unlock()

			if !ok {
				// Create Table
				err := c.d.SQL.CreateNewTable(tableName, requiredKeys, c.toExecerQueryerPreparer())
				if err != nil {
					return nil, err
				}
			} else {
				// Add missing column
				for _, k := range missingKeys {
					err = c.d.SQL.AddColumnToTable(tableName, k, c.toExecerQueryerPreparer())
					if err != nil {
						return nil, err
					}
				}
			}

			existingKeys, err = c.d.SQL.GetAllTableColumns(tableName, c.toExecerQueryerPreparer())
			if err != nil {
				return nil, err
			}
			c.d.schemas[tableName] = existingKeys
		}

		return c.d.SQL.InsertValuesPrepare(tableName, requiredKeys, c.toExecerQueryerPreparer())
	}

	return c.c.Prepare(query)
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
	return c.c.c.Prepare(query)
}

func (c *connExecerQueryerPreparer) Exec(query string, args []driver.Value) (driver.Result, error) {
	return c.c.DoExec(query, args...)
}

func (c *connExecerQueryerPreparer) Query(query string, args []driver.Value) (driver.Rows, error) {
	return c.c.DoQuery(query, args...)
}
