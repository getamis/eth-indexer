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

package indexer

import (
	"fmt"

	"github.com/getamis/eth-indexer/client"
	"github.com/getamis/sirius/database"
	gormFactory "github.com/getamis/sirius/database/gorm"
	"github.com/getamis/sirius/database/mysql"
	"github.com/jinzhu/gorm"
)

func NewEthConn(url string) (client.EthClient, error) {
	return client.NewClient(url)
}

func NewDatabase() (*gorm.DB, error) {
	return gormFactory.New(dbDriver,
		database.DriverOption(
			mysql.Database(dbName),
			mysql.Connector(mysql.DefaultProtocol, dbHost, fmt.Sprintf("%d", dbPort)),
			mysql.UserInfo(dbUser, dbPassword),
		),
	)
}
