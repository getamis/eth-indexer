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
	sbStore "github.com/getamis/eth-indexer/store/subscription"
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

	// Subscriptions store
	FindTotalBalance(blockNumber int64, token common.Address, group int64) (result *model.TotalBalance, err error)
}

type accountStore = accStore.Store
type blockHeaderStore = bhStore.Store
type transactionStore = txStore.Store
type subscriptionsStore = sbStore.Store

type serviceManager struct {
	accountStore
	blockHeaderStore
	transactionStore
	subscriptionsStore
}

// NewServiceManager news a service manager to serve data for RPC services.
func NewServiceManager(db *gorm.DB) ServiceManager {
	return &serviceManager{
		accountStore:       accStore.NewWithDB(db),
		blockHeaderStore:   bhStore.NewWithDB(db),
		transactionStore:   txStore.NewWithDB(db),
		subscriptionsStore: sbStore.NewWithDB(db),
	}
}
