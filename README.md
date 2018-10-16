# eth-indexer

eth-indexer is an Ethereum blockchain indexer project to crawl blocks, transactions & state difference per block/address into MySQL database.

[![travis](https://travis-ci.com/getamis/eth-indexer.svg?branch=develop)](https://travis-ci.com/getamis/eth-indexer)
[![codecov](https://codecov.io/gh/getamis/eth-indexer/branch/develop/graph/badge.svg)](https://codecov.io/gh/getamis/eth-indexer)
[![Go Report Card](https://goreportcard.com/badge/github.com/getamis/eth-indexer)](https://goreportcard.com/report/github.com/getamis/eth-indexer)

## Getting Started

There are 3 main components in the project:
1. geth: modified geth to get state difference per block/address
2. idx-database: MySQL to store all indexed data
3. indexer: indexer to crawl from geth then push to database

### Prerequisites

* docker
* docker-compose

### Before Building

Before building, please make sure environment variables `MYSQL_DATA_PATH` and `GETH_DATA_PATH` are setup properly, which are used to mount local data folder to MySQL and Geth containers for data persistence.
One way to set this up is to have a `.env` file in the same folder of the `docker-compose.yml`

Example `.env` file:

```
MYSQL_DATA_PATH=~/indexer-data/mysql
GETH_DATA_PATH=~/indexer-data/geth
```

### Configs and Flags

_eth_indexer_ supports two kinds of input:
1. static config YAML files
2. dynamic flags through command line

You can either define your `configs/config.yml` or pass flags (e.g., `indexer --eth.port 1234`) from command line to start _eth_indexer_.
If you use both settings, _eth_indexer_ will load `configs/config.yaml` as default and overwrite the corresponding values with specified flags from command line.

### Build

```shell
$ git clone git@github.com:getamis/eth-indexer.git
$ cd eth-indexer
$ # Set MYSQL_DATA_PATH and GETH_DATA_PATH environment variables
$ docker-compose build
```

### Usage

We use docker-compose for testing and developing. `MYSQL_DATA_PATH` & `GETH_DATA_PATH` environment variables are necessary, create them out of eth-indexer directory to store database and geth data.

first time to run indexer you need to create the database schema

```shell
$ mkdir -p ~/indexer-data/mysql ~/indexer-data/geth
# Create database sechema
MYSQL_DATA_PATH="$HOME/indexer-data/mysql" docker-compose up idx-database idx-migration
# press Ctrl + C when see `eth-indexer_idx-migration_1 exited with code 0`
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

```

ERC20 is similar, and you can see [the test case for ERC20](store/balance_erc20_test.go) to know how to use it.

## Contributing

There are several ways to contribute to this project:

1. **Find bug**: create an issue in our Github issue tracker.
2. **Fix a bug**: check our issue tracker, leave comments and send a pull request to us to fix a bug.
3. **Make new feature**: leave your idea in the issue tracker and discuss with us then send a pull request!

## License

This project is licensed under the LGPL 3 - see the [LICENSE](LICENSE) file for details
