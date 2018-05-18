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

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/sirius/log"
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
	// FindERC20 finds the erc20 code
	FindERC20(address gethCommon.Address) (*model.ERC20, error)
	// InsertERC20 inserts the erc20 code
	InsertERC20(code *model.ERC20) error
	// InsertTd writes the total difficulty for a block
	InsertTd(block *types.Block, td *big.Int) error
	// LatestHeader returns a latest header from db
	LatestHeader() (*model.Header, error)
	// GetHeaderByNumber returns the header of the given block number
	GetHeaderByNumber(number int64) (*model.Header, error)
	// GetTd returns the TD of the given block hash
	GetTd(hash []byte) (*model.TotalDifficulty, error)
	// UpdateBlock updates all block data. `delete` indicates whether deletes all data before update.
	// If `delete` is false, ignore duplicate key error.
	UpdateBlocks(blocks []*types.Block, receipts [][]*types.Receipt, dumps []*state.DirtyDump, delete bool) error
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

func (m *manager) UpdateBlocks(blocks []*types.Block, receipts [][]*types.Receipt, dumps []*state.DirtyDump, delete bool) (err error) {
	size := len(blocks)
	if size != len(receipts) || size != len(dumps) {
		log.Error("Inconsistent states", "blocks", size, "receipts", len(receipts), "dumps", len(dumps))
		return common.ErrInconsistentStates
	}

	from := int64(blocks[0].NumberU64())
	to := int64(blocks[size-1].NumberU64())
	if (to - from + 1) != int64(size) {
		log.Error("Inconsistent size and range", "size", size, "range", to-from+1)
		return common.ErrInconsistentStates
	}

	dbTx := m.db.Begin()
	defer func() {
		if err != nil {
			dbTx.Rollback()
			return
		}
		err = dbTx.Commit().Error
	}()

	if delete {
		// Delete all blocks, recipients and states within this range
		err = m.delete(dbTx, from, to)
		if err != nil {
			return err
		}
	}

	// Start to insert blocks and states
	for i := 0; i < size; i++ {
		err = m.insertBlock(dbTx, blocks[i], receipts[i])
		if err != nil && (delete || !common.DuplicateError(err)) {
			return
		}

		err = m.updateState(dbTx, blocks[i], dumps[i], !delete)
		if err != nil {
			return
		}
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
func (m *manager) updateState(dbTx *gorm.DB, block *types.Block, dump *state.DirtyDump, ignoreDuplicateError bool) error {
	accountStore := account.NewWithDB(dbTx)
	blockNumber := block.Number().Int64()
	// Insert modified accounts
	modifiedERC20s := make(map[string]struct{})
	for addr, account := range dump.Accounts {
		err := insertAccount(accountStore, blockNumber, addr, account)
		if err != nil && (!ignoreDuplicateError || !common.DuplicateError(err)) {
			return err
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
					return err
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
		err := accountStore.InsertERC20Storage(s)
		if err != nil && (!ignoreDuplicateError || !common.DuplicateError(err)) {
			return err
		}
	}
	return nil
}

// delete deletes block and state data inside a DB transaction
func (m *manager) delete(dbTx *gorm.DB, from, to int64) (err error) {
	headerStore := header.NewWithDB(dbTx)
	txStore := transaction.NewWithDB(dbTx)
	receiptStore := receipt.NewWithDB(dbTx)
	accountStore := account.NewWithDB(dbTx)
	err = headerStore.Delete(from, to)
	if err != nil {
		return
	}
	err = txStore.Delete(from, to)
	if err != nil {
		return
	}
	err = receiptStore.Delete(from, to)
	if err != nil {
		return
	}
	err = accountStore.DeleteAccounts(from, to)
	if err != nil {
		return
	}

	for hexAddr := range m.erc20List {
		err = accountStore.DeleteERC20Storage(gethCommon.HexToAddress(hexAddr), from, to)
		if err != nil {
			return
		}
	}
	return
}

func (m *manager) InsertERC20(code *model.ERC20) error {
	accountStore := account.NewWithDB(m.db)
	return accountStore.InsertERC20(code)
}

func (m *manager) FindERC20(address gethCommon.Address) (*model.ERC20, error) {
	accountStore := account.NewWithDB(m.db)
	return accountStore.FindERC20(address)
}

func insertAccount(accountStore account.Store, blockNumber int64, addr string, account state.DirtyDumpAccount) error {
	return accountStore.InsertAccount(&model.Account{
		BlockNumber: blockNumber,
		Address:     common.HexToBytes(addr),
		Balance:     account.Balance,
		Nonce:       int64(account.Nonce),
	})
}
