package dynsql

import (
	"database/sql/driver"
	"log"
	"sync"
)

type Stmt struct {
	s            driver.Stmt
	fixedArgsLen int
	args         []driver.Value
	once         *sync.Once
	tx           driver.Tx
}

func newStmt(s driver.Stmt, args []driver.Value, tx driver.Tx, fixedArgsLen int) driver.Stmt {
	return &Stmt{s, fixedArgsLen, args, &sync.Once{}, tx}
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

func (s *Stmt) Close() (err error) {
	s.once.Do(func() {
		err = s.tx.Commit()
	})
	if err != nil {
		log.Printf("%+v\n", err)
	}
	return s.s.Close()
}

func (s *Stmt) NumInput() int {
	return len(s.args) - s.fixedArgsLen
}

func (s *Stmt) Exec(args []driver.Value) (driver.Result, error) {
	args = s.overwriteArgs(args)
	return s.s.Exec(args)
}

func (s *Stmt) Query(args []driver.Value) (driver.Rows, error) {
	args = s.overwriteArgs(args)
	return s.s.Query(args)
}
