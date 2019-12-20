package dynsql

import (
	"database/sql/driver"
)

type Stmt struct {
	s    driver.Stmt
	args []driver.Value
}

func newStmt(s driver.Stmt, args []driver.Value) driver.Stmt {
	return &Stmt{s, args}
}

func (s *Stmt) overwriteArgs(argsNew []driver.Value) []driver.Value {
	argsOrg := s.args
	for i, a := range argsNew {
		if i < len(argsOrg) {
			argsOrg[i] = a
		} else {
			argsOrg = append(argsOrg, a)
		}
	}
	return argsOrg
}

func (s *Stmt) Close() (err error) {
	return s.s.Close()
}

func (s *Stmt) NumInput() int {
	return len(s.args)
}

func (s *Stmt) Exec(args []driver.Value) (driver.Result, error) {
	args = s.overwriteArgs(args)
	return s.s.Exec(args)
}

func (s *Stmt) Query(args []driver.Value) (driver.Rows, error) {
	args = s.overwriteArgs(args)
	return s.s.Query(args)
}
