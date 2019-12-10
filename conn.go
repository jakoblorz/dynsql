package dynsql

import "regexp"

var (
	InsertJSONStatementHeads []string
	insertJSONStatement = "Insert Json Into"
)

func init() {
	InsertJSONStatementHeads = []string{
		insertJSONStatement,
		strings.ToUpper(insertJSONStatement),
		strings.ToLower(insertJSONStatement),
	}
}

type Conn struct {
	d *Driver

	c driver.Conn
}

func (c *Conn) Prepare(query string) (driver.Stmt, error) {
	for _, head := range InsertJSONStatementHeads {
		if len(query) < len(head) {
			continue
		}
		if query[:len(head)] != head {
			continue
		}


		break
	}
}
func (c *Conn) Close() error
func (c *Conn) Begin() (driver.Tx, error)