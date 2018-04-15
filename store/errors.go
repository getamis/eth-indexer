package store

import (
	"errors"

	"github.com/go-sql-driver/mysql"
)

const (
	ErrCodeDuplicateKey = 1062
)

var (
	// ErrWrongSigner returns if it's a wrong signer
	ErrWrongSigner = errors.New("wrong signer")
)

// DuplicateError checks whether it's a duplicate key error
func DuplicateError(err error) bool {
	if err == nil {
		return false
	}

	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		return mysqlErr.Number == ErrCodeDuplicateKey
	}
	return false
}
