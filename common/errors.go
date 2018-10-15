// Copyright 2018 The eth-indexer Authors
// This file is part of the eth-indexer library.
//
// The eth-indexer library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The eth-indexer library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the eth-indexer library. If not, see <http://www.gnu.org/licenses/>.

package common

import (
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"
)

const (
	ErrCodeDuplicateKey = 1062
)

var (
	// ErrWrongSigner is returned if it's a wrong signer
	ErrWrongSigner = errors.New("wrong signer")
	// ErrInconsistentStates is returned if the number of blocks, dumps or receipts are different
	ErrInconsistentStates = errors.New("inconsistent states")
	// ErrInvalidTD is returned when a block has invalid TD
	ErrInvalidTD = errors.New("invalid TD")
	// ErrInvalidReceiptLog is returned if it's a invalid receipt log
	ErrInvalidReceiptLog = errors.New("invalid receipt log")
	// ErrHasPrevBalance is returned if an account has a previous balance when it's a new subscription
	ErrHasPrevBalance = errors.New("missing previous balance")
	// ErrMissingPrevBalance is returned if an account is missing previous balance when it's an old subscription
	ErrMissingPrevBalance = errors.New("missing previous balance")
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

// NotFoundError checks whether it's a not found error
func NotFoundError(err error) bool {
	return err == sql.ErrNoRows
}
