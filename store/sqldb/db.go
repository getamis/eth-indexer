// Copyright © 2018 AMIS Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sqldb

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/getamis/sirius/database"
	"github.com/getamis/sirius/database/mysql"
	"github.com/jmoiron/sqlx"
)

const (
	NullDateTime = "1000-01-01 00:00:00"
)

var (
	NullTime, _ = time.Parse("2006-01-02 15:04:05", NullDateTime)

	ErrInvalidPage  = errors.New("invalid page")
	ErrInvalidLimit = errors.New("invalid limit")
)

type QueryParameters struct {
	Page  uint64
	Limit uint64
}

// DbOrTx defines the minimal set of sqlx database we use. Used when a database store
// needs either a database or a DB transaction.
type DbOrTx interface {
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

const (
	// retryDelay is the delay time for each retrying
	retryDelay = 1 * time.Second
	// retryTimeout is the timeout for retry process
	retryTimeout = 10 * time.Second
)

// New creates a sql.DB object with connectivity.
func New(driver string, opts ...database.Option) (db *sqlx.DB, err error) {
	o := &database.Options{}

	for _, opt := range opts {
		opt(o)
	}

	if o.RetryDelay == 0 {
		o.RetryDelay = retryDelay
	}

	if o.RetryTimeout == 0 {
		o.RetryTimeout = retryTimeout
	}

	var connectionString string
	switch driver {
	case "mysql":
		connectionString, err = mysql.ToConnectionString(o.DriverOptions...)
	default:
		return nil, fmt.Errorf("unsupported driver '%s'", driver)
	}

	if err != nil {
		return nil, err
	}

	return connectToDatabase(driver, connectionString, o)
}

func SimpleConnect(driver, connectionString string) (sqlDB *sqlx.DB, err error) {
	db, err := sqlx.Connect(driver, connectionString)
	if err != nil {
		return nil, err
	}
	// call Unsafe() to omit error for missing columns in struct like ID
	return db.Unsafe(), nil
}

func connectToDatabase(driver, connectionString string, o *database.Options) (sqlDB *sqlx.DB, err error) {
	getDB := func() (*sqlx.DB, error) {
		db, err := SimpleConnect(driver, connectionString)
		if err != nil {
			return nil, err
		}
		if err := db.Ping(); err != nil {
			return nil, err
		}

		if o.MaxIdleConns > 0 {
			db.SetMaxIdleConns(o.MaxIdleConns)
		}

		if o.MaxOpenConns > 0 {
			db.SetMaxOpenConns(o.MaxOpenConns)
		}

		if o.ConnMaxLifetime > 0 {
			db.SetConnMaxLifetime(o.ConnMaxLifetime)
		}
		return db, nil
	}

	sqlDB, err = getDB()
	if err == nil {
		return
	}

	// retry
	timer := time.NewTimer(o.RetryTimeout)
	defer timer.Stop()

	ticker := time.NewTicker(o.RetryDelay)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			return nil, err
		case <-ticker.C:
			sqlDB, err = getDB()
			if err == nil {
				return
			}
		}
	}
}

func ToTimeStr(now time.Time) string {
	return now.Format("2006-01-02 15:04:05")
}

func Hex(data []byte) string {
	return hex.EncodeToString(data)
}

func InClauseForBytes(arr [][]byte) string {
	var buf bytes.Buffer
	last := len(arr) - 1
	for i, h := range arr {
		buf.WriteString(fmt.Sprintf("X'%s'", Hex(h)))
		if i < last {
			buf.WriteString(fmt.Sprintf(", "))
		}
	}
	return buf.String()
}
