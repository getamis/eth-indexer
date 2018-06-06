# eth-indexer

eth-indexer is an Ethereum blockchain indexer project to crawl blocks, transactions & state difference per block/address into MySQL database.

## Getting Started

There are few docker images in the project:
1. geth: modified geth to get state difference per block/address
2. ws-database: MySQL to store all indexed data
3. ws-indexer: indexer to crawl from geth then push to database

### Prerequisites

* docker
* docker-compose

### Build

```shell
$ git clone git@github.com:getamis/eth-indexer.git
$ cd eth-indexer
$ docker-compose build
```

### Usage

We use docker-compose for testing and developing. MYSQL_DATA_PATH & GETH_DATA_PATH environment variables are necessary, create them out of eth-indexer directory to store database and geth data.

first time to run indexer you need to create the database schema

```shell
$ mkdir -p ~/indexer-data/mysql ~/indexer-data/geth
# Create database sechema
MYSQL_DATA_PATH="$HOME/indexer-data/mysql" docker-compose up ws-database ws-migration
# press Ctrl + C when see `eth-indexer_ws-migration_1 exited with code 0`
```

then use `docker-compose up` with environment variables to start indexer:

```shell
$ MYSQL_DATA_PATH="$HOME/indexer-data/mysql" GETH_DATA_PATH="$HOME/indexer-data/geth" docker-compose up
```

wait few minutes, then you can see indexing messages from indexer:

```
Inserted TD for block                    number=0       TD=17179869184 hash=0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3
```

### Example

Once there are some data in MySQL, you can query specific data from it, e.g., you can get data from `block_headers` and `transactions` table. Balance is slightly different, and you can take a look at [example](example) folder to see how to query them.

```go
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
```

ERC20 is similar, and you can see [the test case for ERC20](store/balance_erc20_test.go) to know how to use it.

## Contributing

There are several ways to contribute to this project:

1. **Find bug**: create an issue in our Github issue tracker.
2. **Fix a bug**: check our issue tracker, leave comments and send a pull request to us to fix a bug.
3. **Make new feature**: leave your idea in the issue tracker and discuss with us then send a pull request!

## License

This project is licensed under the LGPL 3 - see the [LICENSE](LICENSE) file for details
