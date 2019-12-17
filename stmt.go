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
		o := i + len(InheritedFields)
		if o < len(argsOrg) {
			argsOrg[o] = a
		} else {
			argsOrg = append(argsOrg, a)
		}
	}
	return argsOrg
}

func (s *Stmt) Close() error {
	return s.s.Close()
}

func (s *Stmt) NumInput() int {
	return s.s.NumInput()
}

func (s *Stmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.s.Exec(s.overwriteArgs(args))
}

func (s *Stmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.s.Query(s.overwriteArgs(args))
}
