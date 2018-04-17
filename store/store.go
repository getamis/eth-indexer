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
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/service/pb"
	"github.com/maichain/eth-indexer/store/account"
	header "github.com/maichain/eth-indexer/store/block_header"
	"github.com/maichain/eth-indexer/store/model"
	"github.com/maichain/eth-indexer/store/transaction"
	receipt "github.com/maichain/eth-indexer/store/transaction_receipt"
)

//go:generate mockery -name Manager

// Manager is a wrapper interface to insert block, receipt and states quickly
type Manager interface {
	// InsertBlock inserts blocks and receipts in db if the block doesn't exist
	InsertBlock(block *types.Block, receipts []*types.Receipt) error
	// UpdateState updates states for the given blocks
	UpdateState(block *types.Block, dump *state.Dump) error
	// LatestHeader returns a latest header from db
	LatestHeader() (*pb.BlockHeader, error)
	// LatestStateBlock returns a latest state block from db
	LatestStateBlock() (*model.StateBlock, error)
}

type manager struct {
	db *gorm.DB
}

// NewManager news a store manager to insert block, receipts and states.
func NewManager(db *gorm.DB) Manager {
	return &manager{db: db}
}

func (m *manager) InsertBlock(block *types.Block, receipts []*types.Receipt) (err error) {
	dbtx := m.db.Begin()
	headerStore := header.NewWithDB(dbtx)
	txStore := transaction.NewWithDB(dbtx)
	receiptStore := receipt.NewWithDB(dbtx)

	defer func() {
		err = finalizeTransaction(dbtx, err)
	}()

	// TODO: how to ensure all data are inserted?
	err = headerStore.Insert(Header(block))
	if err != nil {
		return err
	}

	for _, t := range block.Transactions() {
		tx, err := Transaction(block, t)
		if err != nil {
			return err
		}
		err = txStore.Insert(tx)
		if err != nil {
			return err
		}
	}

	for _, r := range receipts {
		err = receiptStore.Insert(Receipt(r))
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) LatestHeader() (*pb.BlockHeader, error) {
	hs := header.NewWithDB(m.db)
	return hs.Last()
}

func (m *manager) UpdateState(block *types.Block, dump *state.Dump) (err error) {
	// Ensure the state root is the same
	if HashHex(block.Root()) != dump.Root {
		return ErrInconsistentRoot
	}

	dbtx := m.db.Begin()
	accountStore := account.NewWithDB(dbtx)
	defer func() {
		err = finalizeTransaction(dbtx, err)
	}()

	// Insert state block
	err = accountStore.InsertStateBlock(model.StateBlock{
		Number: block.Number().Int64(),
	})
	if err != nil {
		return
	}

	// Insert modified accounts
	for addr, account := range dump.Accounts {
		isContract := account.Code != ""

		if isContract {
			err = insertContract(accountStore, block.Number().Int64(), addr, account)
			if err != nil {
				return
			}
		} else {
			err = insertAccount(accountStore, block.Number().Int64(), addr, account)
			if err != nil {
				return
			}
		}
	}
	return
}

func (m *manager) LatestStateBlock() (*model.StateBlock, error) {
	return account.NewWithDB(m.db).LastStateBlock()
}

// finalizeTransaction finalizes the db transaction and ignores duplicate key error
func finalizeTransaction(dbtx *gorm.DB, err error) error {
	if err != nil {
		dbtx.Rollback()
		// If it's a duplicate key error, ignore it
		if DuplicateError(err) {
			err = nil
		}
		return err
	}
	return dbtx.Commit().Error
}

func insertContract(accountStore account.Store, blockNumber int64, addr string, account state.DumpAccount) error {
	// Insert contract code
	err := accountStore.InsertContractCode(model.ContractCode{
		Address: common.HexToAddress(addr).Bytes(),
		Hash:    common.HexToHash(account.CodeHash).Bytes(),
		Code:    account.Code,
	})
	// Ignore duplicate error
	if err != nil && !DuplicateError(err) {
		return err
	}

	var storage []byte
	storage, err = json.Marshal(account.Storage)
	if err != nil {
		return err
	}
	// Insert contract state
	return accountStore.InsertContract(model.Contract{
		BlockNumber: blockNumber,
		Address:     common.HexToAddress(addr).Bytes(),
		Balance:     account.Balance,
		Nonce:       int64(account.Nonce),
		Root:        common.HexToHash(account.Root).Bytes(),
		Storage:     storage,
	})
}

func insertAccount(accountStore account.Store, blockNumber int64, addr string, account state.DumpAccount) error {
	return accountStore.InsertAccount(model.Account{
		BlockNumber: blockNumber,
		Address:     common.HexToAddress(addr).Bytes(),
		Balance:     account.Balance,
		Nonce:       int64(account.Nonce),
	})
}
