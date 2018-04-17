// Copyright 2018 AMIS Technologies
// This file is part of eapi.
//
// The eapi is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The eapi is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with eapi. If not, see <http://www.gnu.org/licenses/>.

package indexer

import (
	"fmt"

	"github.com/getamis/sirius/database"
	gormFactory "github.com/getamis/sirius/database/gorm"
	"github.com/getamis/sirius/database/mysql"
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/service/indexer"
)

func NewEthConn(url string) (indexer.EthClient, error) {
	return indexer.NewClient(url)
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
