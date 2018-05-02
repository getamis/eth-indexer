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
	"github.com/ethereum/go-ethereum/common"
	"github.com/jinzhu/gorm"

	"github.com/maichain/eth-indexer/model"
	accStore "github.com/maichain/eth-indexer/store/account"
	bhStore "github.com/maichain/eth-indexer/store/block_header"
	txStore "github.com/maichain/eth-indexer/store/transaction"
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

	// Account store
	LastStateBlock() (result *model.StateBlock, err error)
	FindAccount(address common.Address, blockNr ...int64) (result *model.Account, err error)
	FindContract(address common.Address, blockNr ...int64) (result *model.Contract, err error)
	FindContractCode(address common.Address) (result *model.ContractCode, err error)
	FindStateBlock(blockNr int64) (result *model.StateBlock, err error)
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
