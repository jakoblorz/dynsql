package dynsql

import (
	"database/sql/driver"
)

type Stmt struct {
	s driver.Stmt
}

func createStmtFromDriverStmt(s driver.Stmt) driver.Stmt {
	return &Stmt{s}
}

func (s *Stmt) Close() error                                    {}
func (s *Stmt) NumInput() int                                   {}
func (s *Stmt) Exec(args []driver.Value) (driver.Result, error) {}
func (s *Stmt) Query(args []driver.Value) (driver.Rows, error)  {}
