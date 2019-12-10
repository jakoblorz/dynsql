package sqlite3

import ( 
	"database/sql"
	"github.com/jakoblorz/dynsql"
	sqlite3 "github.com/mattn/go-sqlite3"
)

func translateSQLType(sqlType string) dynsql.AbstractColumnType {

}

func init() {
	sql.Register("dyn-sqlite3", dynsql.CreateDriver(&sqlite3.SQLiteDriver{}, dynsql.DriverQuerySet{
		GetAllTableNames: ``,
		GetAllColumnsFromTableName: ``,

		TranslateSQLType: translateSQLType,
	}))
}
