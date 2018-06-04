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

package store

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jinzhu/gorm"
	"github.com/shopspring/decimal"

	"github.com/getamis/eth-indexer/model"
	accStore "github.com/getamis/eth-indexer/store/account"
	bhStore "github.com/getamis/eth-indexer/store/block_header"
	txStore "github.com/getamis/eth-indexer/store/transaction"
)

//go:generate mockery -name ServiceManager

// ServiceManager is a wrapper interface that serves data for RPC services.
type ServiceManager interface {
	// Block header store
	FindBlockByNumber(blockNumber int64) (result *model.Header, err error)
	FindBlockByHash(hash []byte) (result *model.Header, err error)
	FindLatestBlock() (result *model.Header, err error)

	// Transaction store
	FindTransaction(hash []byte) (result *model.Transaction, err error)
	FindTransactionsByBlockHash(blockHash []byte) (result []*model.Transaction, err error)

	// GetBalance returns the amount of wei for the given address in the state of the
	// given block number. If blockNr < 0, the given block is the latest block.
	// Noted that the return block number may be different from the input one because
	// we don't have state in the input one.
	GetBalance(ctx context.Context, address common.Address, blockNr int64) (balance *big.Int, blockNumber *big.Int, err error)

	// GetERC20Balance returns the amount of ERC20 token for the given address in the state of the
	// given block number. If blockNr < 0, the given block is the latest block.
	// Noted that the return block number may be different from the input one because
	// we don't have state in the input one.
	GetERC20Balance(ctx context.Context, contractAddress, address common.Address, blockNr int64) (*decimal.Decimal, *big.Int, error)
}

type accountStore = accStore.Store
type blockHeaderStore = bhStore.Store
type transactionStore = txStore.Store

type serviceManager struct {
	accountStore
	blockHeaderStore
	transactionStore
}

// NewServiceManager news a service manager to serve data for RPC services.
func NewServiceManager(db *gorm.DB) ServiceManager {
	return &serviceManager{
		accountStore:     accStore.NewWithDB(db),
		blockHeaderStore: bhStore.NewWithDB(db),
		transactionStore: txStore.NewWithDB(db),
	}
}
