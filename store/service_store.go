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

	"github.com/ethereum/go-ethereum/common"
	"github.com/jinzhu/gorm"
	"math/big"

	"github.com/maichain/eth-indexer/model"
	"github.com/maichain/eth-indexer/service/account"
	bhStore "github.com/maichain/eth-indexer/store/block_header"
	txStore "github.com/maichain/eth-indexer/store/transaction"
)

//go:generate mockery -name ServiceManager

// ServiceManager is a wrapper interface that serves data for RPC services.
type ServiceManager interface {
	FindBlockByNumber(blockNumber int64) (result *model.Header, err error)
	FindBlockByHash(hash []byte) (result *model.Header, err error)
	FindLatestBlock() (result *model.Header, err error)
	FindTransaction(hash []byte) (result *model.Transaction, err error)
	FindTransactionsByBlockHash(blockHash []byte) (result []*model.Transaction, err error)
	GetBalance(ctx context.Context, address common.Address, blockNr int64) (balance *big.Int, blockNumber *big.Int, err error)
	GetERC20Balance(ctx context.Context, contractAddress, address common.Address, blockNr int64) (balance *big.Int, blockNumber *big.Int, err error)
}

type serviceManager struct {
	accountAPI account.API
	bhStore    bhStore.Store
	txStore    txStore.Store
}

// NewServiceManager news a service manager to serve data for RPC services.
func NewServiceManager(db *gorm.DB) ServiceManager {
	return &serviceManager{
		accountAPI: account.NewAPIWithWithDB(db),
		bhStore:    bhStore.NewWithDB(db),
		txStore:    txStore.NewWithDB(db),
	}
}

func (s *serviceManager) FindBlockByNumber(blockNumber int64) (result *model.Header, err error) {
	return s.bhStore.FindBlockByNumber(blockNumber)
}

func (s *serviceManager) FindBlockByHash(hash []byte) (result *model.Header, err error) {
	return s.bhStore.FindBlockByHash(hash)
}

func (s *serviceManager) FindLatestBlock() (result *model.Header, err error) {
	return s.bhStore.Last()
}

func (s *serviceManager) FindTransaction(hash []byte) (result *model.Transaction, err error) {
	return s.txStore.FindTransaction(hash)
}

func (s *serviceManager) FindTransactionsByBlockHash(blockHash []byte) (result []*model.Transaction, err error) {
	return s.txStore.FindTransactionsByBlockHash(blockHash)
}

func (s *serviceManager) GetBalance(ctx context.Context, address common.Address, blockNr int64) (balance *big.Int, blockNumber *big.Int, err error) {
	return s.accountAPI.GetBalance(ctx, address, blockNr)
}

func (s *serviceManager) GetERC20Balance(ctx context.Context, contractAddress, address common.Address, blockNr int64) (balance *big.Int, blockNumber *big.Int, err error) {
	return s.accountAPI.GetERC20Balance(ctx, contractAddress, address, blockNr)
}
