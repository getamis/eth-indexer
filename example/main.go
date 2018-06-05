package main

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/getamis/eth-indexer/store"
	"github.com/getamis/sirius/database"
	gormFactory "github.com/getamis/sirius/database/gorm"
	"github.com/getamis/sirius/database/mysql"
)

func main() {
	db, _ := gormFactory.New("mysql",
		database.DriverOption(
			mysql.Database("eth-db"),
			mysql.Connector(mysql.DefaultProtocol, "127.0.0.1", "3306"),
			mysql.UserInfo("root", "my-secret-pw"),
		),
	)
	addr := common.HexToAddress("0x756f45e3fa69347a9a973a725e3c98bc4db0b5a0")
	manager := store.NewServiceManager(db)
	balance, blockNumber, _ := manager.GetBalance(context.Background(), addr, -1)
	fmt.Println(balance, blockNumber)
}
