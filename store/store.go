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
	"math/big"

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/account"
	header "github.com/getamis/eth-indexer/store/block_header"
	"github.com/getamis/eth-indexer/store/transaction"
	receipt "github.com/getamis/eth-indexer/store/transaction_receipt"
	"github.com/getamis/sirius/log"
	"github.com/jinzhu/gorm"
)

// UpdateMode defines the mode to update blocks
type UpdateMode = int

const (
	// ModeReOrg represents update blocks by reorg
	// Stop if any errors occur.
	ModeReOrg UpdateMode = iota
	// ModeSync represents update blocks by ethereum sync
	// Stop if any errors occur, but return nil error if it's a duplicate error
	ModeSync
	// ModeForceSync represents update erc20 storage data forcibly
	// Update all erc20 storage data even if duplicate errors occur.
	ModeForceSync
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
	// UpdateBlock updates all block data. 'delete' indicates whether deletes all data before update.
	UpdateBlocks(blocks []*types.Block, receipts [][]*types.Receipt, dumps []*state.DirtyDump, mode UpdateMode) error
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

func (m *manager) UpdateBlocks(blocks []*types.Block, receipts [][]*types.Receipt, dumps []*state.DirtyDump, mode UpdateMode) (err error) {
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
		// In ModeSync, return nil error if it's a duplicate error
		if (mode == ModeSync) && common.DuplicateError(err) {
			err = nil
		}

		if err != nil {
			dbTx.Rollback()
			return
		}
		err = dbTx.Commit().Error
	}()

	// In ModeReOrg, delete all blocks, recipients and states within this range before insertions
	if mode == ModeReOrg {
		err = m.delete(dbTx, from, to)
		if err != nil {
			return err
		}
	}

	// In ModeForceSync, ignore the duplicate error and continue to process it.
	ignoreDupErr := (mode == ModeForceSync)
	// Start to insert blocks and states
	for i := 0; i < size; i++ {
		err = m.insertBlock(dbTx, blocks[i], receipts[i], dumps[i])
		if ignoreDupErr && common.DuplicateError(err) {
			err = nil
		}

		if err != nil {
			return
		}
		err = m.updateERC20(dbTx, blocks[i], dumps[i], ignoreDupErr)
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

// insertBlock inserts block, and accounts inside a DB transaction
func (m *manager) insertBlock(dbTx *gorm.DB, block *types.Block, receipts []*types.Receipt, dump *state.DirtyDump) (err error) {
	headerStore := header.NewWithDB(dbTx)
	txStore := transaction.NewWithDB(dbTx)
	receiptStore := receipt.NewWithDB(dbTx)
	accountStore := account.NewWithDB(dbTx)

	// Insert blocks, txs and receipts
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

	// Insert accounts
	blockNumber := block.Number().Int64()
	for addr, account := range dump.Accounts {
		err := insertAccount(accountStore, blockNumber, addr, account)
		if err != nil {
			return err
		}
	}
	return nil
}

// updateERC20 updates erc20 storages. If 'ignoreDuplicateError' is true, ignore duplicate key error, and continue to process
func (m *manager) updateERC20(dbTx *gorm.DB, block *types.Block, dump *state.DirtyDump, ignoreDuplicateError bool) error {
	// Ensure accounts is not nil
	if dump.Accounts == nil {
		dump.Accounts = make(map[string]state.DirtyDumpAccount)
	}

	// There is no erc20 storage updates
	accountStore := account.NewWithDB(dbTx)
	blockNumber := block.Number().Int64()

	for addr, erc20 := range m.erc20List {
		// This erc20 contract is not deployed yet
		if blockNumber < erc20.BlockNumber {
			continue
		}

		account, ok := dump.Accounts[addr]
		// Insert a null record if it's NOT in modified accounts
		if !ok || len(account.Storage) == 0 {
			s := &model.ERC20Storage{
				BlockNumber: blockNumber,
				Address:     common.HexToBytes(addr),
			}
			err := accountStore.InsertERC20Storage(s)
			if err != nil && !(ignoreDuplicateError && common.DuplicateError(err)) {
				return err
			}
			continue
		}

		// Update storage record if it's in modified accounts
		for key, value := range account.Storage {
			s := &model.ERC20Storage{
				BlockNumber: blockNumber,
				Address:     common.HexToBytes(addr),
				Key:         common.HexToBytes(key),
				Value:       common.HexToBytes(value),
			}
			err := accountStore.InsertERC20Storage(s)
			if err != nil && !(ignoreDuplicateError && common.DuplicateError(err)) {
				return err
			}
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
