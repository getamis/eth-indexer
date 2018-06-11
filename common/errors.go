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
	"errors"

	"github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

const (
	ErrCodeDuplicateKey = 1062
)

var (
	// ErrWrongSigner returns if it's a wrong signer
	ErrWrongSigner = errors.New("wrong signer")
	// ErrInconsistentRoot returns if the block and dump states have different root
	ErrInconsistentRoot = errors.New("inconsistent root")
	// ErrInconsistentStates returns if the number of blocks, dumps or recipents are different
	ErrInconsistentStates = errors.New("inconsistent states")
	// ErrInvalidTD is returned when a block has invalid TD
	ErrInvalidTD = errors.New("invalid TD")
	// ErrInvalidReceiptLog returns if it's a invalid receipt log
	ErrInvalidReceiptLog = errors.New("invalid receipt log")
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
	return err == gorm.ErrRecordNotFound
}
