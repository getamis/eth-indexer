// Copyright 2017 AMIS Technologies
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gorm

import (
	"fmt"
	"time"

	"github.com/jinzhu/gorm"

	"github.com/getamis/sirius/database"
	"github.com/getamis/sirius/database/mysql"
)

const (
	// retryDelay is the delay time for each retrying
	retryDelay = 1 * time.Second
	// retryTimeout is the timeout for retry process
	retryTimeout = 10 * time.Second
)

// New creates a GORM database wrapper
func New(driver string, opts ...database.Option) (db *gorm.DB, err error) {
	o := &database.Options{
		RetryDelay:   retryDelay,
		RetryTimeout: retryTimeout,
	}

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

func connectToDatabase(driver, connectionString string, o *database.Options) (*gorm.DB, error) {
	var gormDB *gorm.DB
	var lastErr error

	getDB := func() (*gorm.DB, error) {
		db, err := gorm.Open(driver, connectionString)
		if err != nil {
			return nil, err
		}
		if err := db.DB().Ping(); err != nil {
			return nil, err
		}

		if o.MaxIdleConns > 0 {
			db.DB().SetMaxIdleConns(o.MaxIdleConns)
		}

		if o.MaxOpenConns > 0 {
			db.DB().SetMaxOpenConns(o.MaxOpenConns)
		}

		if o.ConnMaxLifetime > 0 {
			db.DB().SetConnMaxLifetime(o.ConnMaxLifetime)
		}

		db.LogMode(o.Logging)
		db.SetLogger(&logger{
			logger: o.Logger,
		})
		if o.TableName != "" {
			db = db.Table(o.TableName)
		}

		return db, nil
	}

	gormDB, lastErr = getDB()
	if lastErr == nil {
		return gormDB, nil
	}

	// retry
	timer := time.NewTimer(o.RetryTimeout)
	defer timer.Stop()

	ticker := time.NewTicker(o.RetryDelay)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			if o.Logger != nil {
				o.Logger.Error("Connecting to database timeout", "err", lastErr)
			}
			return nil, lastErr
		case <-ticker.C:
			gormDB, lastErr = getDB()
			if o.Logger != nil {
				o.Logger.Warn("Failed to connect to database, retry...", "err", lastErr)
			}
			if lastErr == nil {
				return gormDB, nil
			}
		}
	}
}
