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
	"math/big"

	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jinzhu/gorm"
	"github.com/maichain/eth-indexer/common"
	"github.com/maichain/eth-indexer/model"
	"github.com/maichain/eth-indexer/store/account"
	header "github.com/maichain/eth-indexer/store/block_header"
	"github.com/maichain/eth-indexer/store/transaction"
	receipt "github.com/maichain/eth-indexer/store/transaction_receipt"
)

//go:generate mockery -name Manager

// Manager is a wrapper interface to insert block, receipt and states quickly
type Manager interface {
	// InsertTd writes the total difficulty for a block
	InsertTd(block *types.Block, td *big.Int) error
	// InsertBlock inserts blocks and receipts in db if the block doesn't exist
	InsertBlock(block *types.Block, receipts []*types.Receipt) error
	// UpdateState updates states for the given blocks
	UpdateState(block *types.Block, accounts map[string]state.DumpDirtyAccount) error
	// DeleteDataFromBlock deletes all data from this block and higher
	DeleteDataFromBlock(blockNumber int64) error
	// LatestHeader returns a latest header from db
	LatestHeader() (*model.Header, error)
	// GetHeaderByNumber returns the header of the given block number
	GetHeaderByNumber(number int64) (*model.Header, error)
	// GetTd returns the TD of the given block hash
	GetTd(hash []byte) (*model.TotalDifficulty, error)
	// LatestStateBlock returns a latest state block from db
	LatestStateBlock() (*model.StateBlock, error)
}

type manager struct {
	db        *gorm.DB
	erc20List map[string]struct{}
}

// NewManager news a store manager to insert block, receipts and states.
func NewManager(db *gorm.DB) (Manager, error) {
	list, err := account.NewWithDB(db).ListERC20()
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	erc20List := make(map[string]struct{})
	for _, e := range list {
		erc20List[common.BytesToHex(e.Address)] = struct{}{}
	}
	return &manager{
		db:        db,
		erc20List: erc20List,
	}, nil
}

func (m *manager) InsertTd(block *types.Block, td *big.Int) error {
	headerStore := header.NewWithDB(m.db)
	return headerStore.InsertTd(common.TotalDifficulty(block, td))
}

func (m *manager) InsertBlock(block *types.Block, receipts []*types.Receipt) (err error) {
	dbtx := m.db.Begin()
	headerStore := header.NewWithDB(dbtx)
	txStore := transaction.NewWithDB(dbtx)
	receiptStore := receipt.NewWithDB(dbtx)

	defer func() {
		err = finalizeTransaction(dbtx, err)
	}()

	err = headerStore.Insert(common.Header(block))
	if err != nil {
		return err
	}

	for _, t := range block.Transactions() {
		tx, err := common.Transaction(block, t)
		if err != nil {
			return err
		}
		err = txStore.Insert(tx)
		if err != nil {
			return err
		}
	}

	for _, r := range receipts {
		err = receiptStore.Insert(common.Receipt(block, r))
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) LatestHeader() (*model.Header, error) {
	hs := header.NewWithDB(m.db)
	return hs.FindLatestBlock()
}

func (m *manager) GetHeaderByNumber(number int64) (*model.Header, error) {
	hs := header.NewWithDB(m.db)
	return hs.FindBlockByNumber(number)
}

func (m *manager) GetTd(hash []byte) (*model.TotalDifficulty, error) {
	return header.NewWithDB(m.db).FindTd(hash)
}

func (m *manager) UpdateState(block *types.Block, accounts map[string]state.DumpDirtyAccount) (err error) {
	dbtx := m.db.Begin()
	accountStore := account.NewWithDB(dbtx)
	defer func() {
		err = finalizeTransaction(dbtx, err)
	}()

	// Insert state block
	err = accountStore.InsertStateBlock(&model.StateBlock{
		Number: block.Number().Int64(),
	})
	if err != nil {
		return
	}

	// Insert modified accounts
	for addr, account := range accounts {
		err = insertAccount(accountStore, block.Number().Int64(), addr, account)
		if err != nil {
			return
		}

		// If it's in our erc20 list, update it's storage
		if _, ok := m.erc20List[addr]; len(account.Storage) > 0 && ok {
			for key, value := range account.Storage {
				s := &model.ERC20Storage{
					BlockNumber: block.Number().Int64(),
					Address:     common.HexToBytes(addr),
					Key:         common.HexToBytes(key),
					Value:       common.HexToBytes(value),
				}
				err = accountStore.InsertERC20Storage(s)
				if err != nil {
					return
				}
			}
		}
	}
	return
}

func (m *manager) DeleteDataFromBlock(blockNumber int64) (err error) {
	dbTx := m.db.Begin()
	accountStore := account.NewWithDB(dbTx)
	headerStore := header.NewWithDB(dbTx)
	txStore := transaction.NewWithDB(dbTx)
	receiptStore := receipt.NewWithDB(dbTx)

	defer func(dbTx *gorm.DB) {
		if err != nil {
			dbTx.Rollback()
			return
		}
		err = dbTx.Commit().Error
	}(dbTx)

	err = headerStore.DeleteFromBlock(blockNumber)
	if err != nil {
		return
	}
	err = txStore.DeleteFromBlock(blockNumber)
	if err != nil {
		return
	}
	err = receiptStore.DeleteFromBlock(blockNumber)
	if err != nil {
		return
	}
	err = accountStore.DeleteAccounts(blockNumber)
	if err != nil {
		return
	}
	err = accountStore.DeleteStateBlocks(blockNumber)
	if err != nil {
		return
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
		if common.DuplicateError(err) {
			err = nil
		}
		return err
	}
	return dbtx.Commit().Error
}

func insertAccount(accountStore account.Store, blockNumber int64, addr string, account state.DumpDirtyAccount) error {
	return accountStore.InsertAccount(&model.Account{
		BlockNumber: blockNumber,
		Address:     common.HexToBytes(addr),
		Balance:     account.Balance,
		Nonce:       int64(account.Nonce),
	})
}
