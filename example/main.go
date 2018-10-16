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

package main

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/account"
	"github.com/getamis/eth-indexer/store/sqldb"
	"github.com/getamis/sirius/database"
	"github.com/getamis/sirius/database/mysql"
)

func main() {
	db, _ := sqldb.New("mysql",
		database.DriverOption(
			mysql.Database("ethdb"),
			mysql.Connector(mysql.DefaultProtocol, "127.0.0.1", "3306"),
			mysql.UserInfo("root", "my-secret-pw"),
		),
	)
	addr := common.HexToAddress("0x756f45e3fa69347a9a973a725e3c98bc4db0b5a0")
	store := account.NewWithDB(db)

	account, err := store.FindAccount(context.Background(), model.ETHAddress, addr)
	if err != nil {
		fmt.Printf("Failed to find account: %v\n", err)
	} else {
		fmt.Printf("Find account, block_number: %v, balance: %v, \n", account.Balance, account.BlockNumber)
	}
}
