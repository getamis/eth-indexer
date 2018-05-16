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
	"bytes"
	"math/big"

	gethCommon "github.com/ethereum/go-ethereum/common"
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
	// InsertERC20 inserts the erc20 code
	InsertERC20(code *model.ERC20) error
	// InsertTd writes the total difficulty for a block
	InsertTd(block *types.Block, td *big.Int) error
	// DeleteDataFromBlock deletes all data from this block and higher
	DeleteStateFromBlock(blockNumber int64) error
	// LatestHeader returns a latest header from db
	LatestHeader() (*model.Header, error)
	// GetHeaderByNumber returns the header of the given block number
	GetHeaderByNumber(number int64) (*model.Header, error)
	// GetTd returns the TD of the given block hash
	GetTd(hash []byte) (*model.TotalDifficulty, error)
	// UpdateBlock updates all block data if the block doesn't exist
	UpdateBlock(block *types.Block, receipts []*types.Receipt, dump *state.DirtyDump) error
	// ForceInsertBlock inserts all block data even if some data already exist
	ForceInsertBlock(block *types.Block, receipts []*types.Receipt, dump *state.DirtyDump) (err error)
}

type manager struct {
	db        *gorm.DB
	erc20List map[string]model.ERC20
}

// NewManager news a store manager to insert block, receipts and states.
func NewManager(db *gorm.DB) (Manager, error) {
	list, err := account.NewWithDB(db).ListERC20()
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	erc20List := make(map[string]model.ERC20)
	for _, e := range list {
		erc20List[common.BytesToHex(e.Address)] = e
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

func (m *manager) UpdateBlock(block *types.Block, receipts []*types.Receipt, dump *state.DirtyDump) (err error) {
	headerStore := header.NewWithDB(m.db)
	blockNumber := block.Number().Int64()
	// Best effort to check if we already have the data
	hdr, err := headerStore.FindBlockByNumber(blockNumber)
	if err == nil && bytes.Equal(hdr.Hash, block.Hash().Bytes()) {
		return
	}
	err = nil

	dbTx := m.db.Begin()
	defer func() {
		err = finalizeTransaction(dbTx, err)
	}()

	err = m.deleteBlock(dbTx, blockNumber)
	if err != nil {
		return
	}

	err = m.insertBlock(dbTx, block, receipts)
	if err != nil {
		return
	}

	// No need to update states
	if dump == nil {
		return
	}
	err = m.updateState(dbTx, block, dump, true)
	if err != nil {
		return
	}
	return
}

func (m *manager) ForceInsertBlock(block *types.Block, receipts []*types.Receipt, dump *state.DirtyDump) (err error) {
	dbTx := m.db.Begin()
	defer func() {
		if err != nil {
			dbTx.Rollback()
			return
		}
		err = dbTx.Commit().Error
	}()

	err = m.insertBlock(dbTx, block, receipts)
	if err != nil && !common.DuplicateError(err) {
		return
	}

	// No need to update states
	if dump == nil {
		return
	}
	err = m.updateState(dbTx, block, dump, false)
	if err != nil {
		return
	}
	return
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

func (m *manager) DeleteStateFromBlock(blockNumber int64) (err error) {
	dbTx := m.db.Begin()
	defer func(dbTx *gorm.DB) {
		if err != nil {
			dbTx.Rollback()
			return
		}
		err = dbTx.Commit().Error
	}(dbTx)

	accountStore := account.NewWithDB(dbTx)
	err = accountStore.DeleteAccounts(blockNumber)
	if err != nil {
		return
	}

	for hexAddr := range m.erc20List {
		err = accountStore.DeleteERC20Storage(gethCommon.HexToAddress(hexAddr), blockNumber)
		if err != nil {
			return
		}
	}
	return
}

// insertBlock inserts block inside a DB transaction
func (m *manager) insertBlock(dbTx *gorm.DB, block *types.Block, receipts []*types.Receipt) (err error) {
	headerStore := header.NewWithDB(dbTx)
	txStore := transaction.NewWithDB(dbTx)
	receiptStore := receipt.NewWithDB(dbTx)

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

// updateState updates states inside a DB transaction
func (m *manager) updateState(dbTx *gorm.DB, block *types.Block, dump *state.DirtyDump, ignoreDuplicateError bool) (err error) {
	accountStore := account.NewWithDB(dbTx)

	blockNumber := block.Number().Int64()
	// Insert modified accounts
	modifiedERC20s := make(map[string]struct{})
	for addr, account := range dump.Accounts {
		err = insertAccount(accountStore, blockNumber, addr, account)
		if err != nil && (!ignoreDuplicateError || !common.DuplicateError(err)) {
			return
		}

		// If it's in our erc20 list, update it's storage
		if _, ok := m.erc20List[addr]; len(account.Storage) > 0 && ok {
			modifiedERC20s[addr] = struct{}{}
			for key, value := range account.Storage {
				s := &model.ERC20Storage{
					BlockNumber: blockNumber,
					Address:     common.HexToBytes(addr),
					Key:         common.HexToBytes(key),
					Value:       common.HexToBytes(value),
				}
				err = accountStore.InsertERC20Storage(s)
				if err != nil && (!ignoreDuplicateError || !common.DuplicateError(err)) {
					return
				}
			}
		}
	}

	// Update non-modified ERC20s
	for addr, erc20 := range m.erc20List {
		// This erc20 contract is not deployed yet
		if blockNumber < erc20.BlockNumber {
			continue
		}

		// This erc20 contract is modified
		if _, ok := modifiedERC20s[addr]; ok {
			continue
		}

		s := &model.ERC20Storage{
			BlockNumber: blockNumber,
			Address:     common.HexToBytes(addr),
		}
		err = accountStore.InsertERC20Storage(s)
		if err != nil && (!ignoreDuplicateError || !common.DuplicateError(err)) {
			return
		}
	}
	return
}

// deleteBlock deletes block data inside a DB transaction
func (m *manager) deleteBlock(dbTx *gorm.DB, blockNumber int64) (err error) {
	headerStore := header.NewWithDB(dbTx)
	txStore := transaction.NewWithDB(dbTx)
	receiptStore := receipt.NewWithDB(dbTx)

	err = headerStore.Delete(blockNumber)
	if err != nil {
		return
	}
	err = txStore.Delete(blockNumber)
	if err != nil {
		return
	}
	err = receiptStore.Delete(blockNumber)
	if err != nil {
		return
	}
	return
}

func (m *manager) InsertERC20(code *model.ERC20) error {
	accountStore := account.NewWithDB(m.db)
	return accountStore.InsertERC20(code)
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

func insertAccount(accountStore account.Store, blockNumber int64, addr string, account state.DirtyDumpAccount) error {
	return accountStore.InsertAccount(&model.Account{
		BlockNumber: blockNumber,
		Address:     common.HexToBytes(addr),
		Balance:     account.Balance,
		Nonce:       int64(account.Nonce),
	})
}
