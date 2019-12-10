package dynsql

import (
	"database/sql/driver"
	"regexp"
)

var (
	InsertJSONStatementHeads = []string{"INSERT INTO", "Insert Into", "insert into"}
	InsertJSONStatementRegex = regexp.MustCompile(`^(INSERT INTO|Insert Into|insert into)\s*(?P<TABLE>[\w\d]+)\s*(JSON|Json|json)\s*(?P<JSON>.*)\s*(\;*)$`)
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

type Conn struct {
	d *Driver
	c driver.Conn
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

func (c *Conn) pollDatabaseData(lockDriver bool) error {
	if lockDriver {
		c.d.mu.Lock()
		defer c.d.mu.Unlock()
	}

	getAllTableNamesStmt, err := c.c.Prepare(c.d.queries.GetAllTableNames)
	if err != nil {
		return err
	}

	getAllTableNamesRows, err := getAllTableNamesStmt.Query(nil)
	if err != nil {
		return err
	}
	defer getAllTableNamesRows.Close()

	iterRowSlice, schemas := []driver.Value{}, map[string][]string{}
	for err = getAllTableNamesRows.Next(iterRowSlice); err == nil; err = getAllTableNamesRows.Next(iterRowSlice) {
		if name, ok := iterRowSlice[0].(string); ok {
			schemas[name] = make([]string, 0)
			continue
		}

		return ErrFailedToParseString
	}
	for table := range schemas {
		schemas[table], err = c.pollColumnDefinitions(table)
		if err != nil {
			return err
		}
	}
	c.d.schemas = schemas

	return nil
}

func (c *Conn) pollColumnDefinitions(tableName string) ([]string, error) {
	getAllColumnsFromTableNameStmt, err := c.c.Prepare(c.d.queries.GetAllColumnsFromTableName)
	if err != nil {
		return nil, err
	}
	defer getAllColumnsFromTableNameStmt.Close()

	getAllColumnsFromTableNameRows, err := getAllColumnsFromTableNameStmt.Query([]driver.Value{tableName})
	if err != nil {
		return nil, err
	}

	iterRowSlice, columnDefinitions := []driver.Value{}, []string{}
	for err = getAllColumnsFromTableNameRows.Next(iterRowSlice); err == nil; err = getAllColumnsFromTableNameRows.Next(iterRowSlice) {
		name, ok := iterRowSlice[0].(string)
		if !ok {
			return nil, ErrFailedToParseString
		}

		columnDefinitions = append(columnDefinitions, name)
	}

	return columnDefinitions, nil
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

		break
	}
}

func (c *Conn) Close() error
func (c *Conn) Begin() (driver.Tx, error)
