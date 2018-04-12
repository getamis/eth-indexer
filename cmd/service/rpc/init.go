// Copyright 2018 AMIS Technologies
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

package rpc

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	ethMySQL "github.com/ethereum/go-ethereum/ethdb/mysql"
	"github.com/getamis/sirius/database"
	gormFactory "github.com/getamis/sirius/database/gorm"
	"github.com/getamis/sirius/database/mysql"
	"github.com/jinzhu/gorm"
)

const (
	defaultDialTimeout = 5 * time.Second
)

func MustNewDatabase() *gorm.DB {
	db, err := gormFactory.New(dbDriver,
		database.DriverOption(
			mysql.Database(dbName),
			mysql.Connector(mysql.DefaultProtocol, dbHost, fmt.Sprintf("%d", dbPort)),
			mysql.UserInfo(dbUser, dbPassword),
		),
	)
	if err != nil {
		panic(err)
	}

	return db
}

func MustNewEthDatabase() ethdb.Database {
	db, err := ethMySQL.NewDatabase("chaindata", &ethMySQL.Config{
		Protocol:             mysql.DefaultProtocol,
		Address:              dbHost,
		Port:                 strconv.Itoa(dbPort),
		User:                 dbUser,
		Password:             dbPassword,
		Database:             dbName,
		AllowNativePasswords: true,
	})
	if err != nil {
		panic(err)
	}
	return db
}
