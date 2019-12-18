package util

import (
	"database/sql/driver"
)

func IterateDriverRows(rows driver.Rows) (<-chan []driver.Value, <-chan error) {
	valCh := make(chan []driver.Value)
	errCh := make(chan error)
	go func() {
		iterRowSlice := make([]driver.Value, len(rows.Columns()))
		err := rows.Next(iterRowSlice)
		for ; err == nil; err = rows.Next(iterRowSlice) {
			valCh <- iterRowSlice
		}
		close(valCh)
		errCh <- err
		close(errCh)
	}()
	return valCh, errCh
}
