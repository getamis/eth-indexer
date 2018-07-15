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

	gethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/getamis/eth-indexer/client"
	"github.com/getamis/eth-indexer/common"
	"github.com/getamis/eth-indexer/model"
	"github.com/getamis/eth-indexer/store/account"
	"github.com/getamis/eth-indexer/store/block_header"
	"github.com/getamis/eth-indexer/store/subscription"
	"github.com/getamis/eth-indexer/store/transaction"
	"github.com/getamis/eth-indexer/store/transaction_receipt"
	"github.com/getamis/eth-indexer/store/uncle_header"
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
)

//go:generate mockery -name Manager

// Manager is a wrapper interface to insert block, receipt and states quickly
type Manager interface {
	// Init the store manager to load the erc20 list
	Init(balancer client.Balancer) error
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
	// UpdateBlocks updates all block data. 'delete' indicates whether deletes all data before update.
	UpdateBlocks(ctx context.Context, blocks []*types.Block, receipts [][]*types.Receipt, dumps []*state.DirtyDump, events [][]*types.TransferLog, mode UpdateMode) error
}

type manager struct {
	db        *gorm.DB
	tokenList map[gethCommon.Address]struct{}
	balancer  client.Balancer
}

// NewManager news a store manager to insert block, receipts and states.
func NewManager(db *gorm.DB) Manager {
	return &manager{
		db: db,
	}
}

func (m *manager) Init(balancer client.Balancer) error {
	list, err := account.NewWithDB(m.db).ListERC20()
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	tokenList := make(map[gethCommon.Address]struct{}, len(list))
	tokenList[model.ETHAddress] = struct{}{}
	for _, e := range list {
		tokenList[gethCommon.BytesToAddress(e.Address)] = struct{}{}
	}
	m.tokenList = tokenList

	// Init balance of function
	m.balancer = balancer
	return nil
}

func (m *manager) InsertTd(block *types.Block, td *big.Int) error {
	return block_header.NewWithDB(m.db).InsertTd(common.TotalDifficulty(block, td))
}

func (m *manager) UpdateBlocks(ctx context.Context, blocks []*types.Block, receipts [][]*types.Receipt, dumps []*state.DirtyDump, events [][]*types.TransferLog, mode UpdateMode) (err error) {
	size := len(blocks)
	if size != len(receipts) || size != len(dumps) || size != len(events) {
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

	// Start to insert blocks and states
	for i := 0; i < size; i++ {
		err = m.insertBlock(ctx, dbTx, blocks[i], receipts[i], dumps[i], events[i])
		if common.DuplicateError(err) {
			err = nil
		}

		if err != nil {
			return
		}
	}
	return
}

func (m *manager) LatestHeader() (*model.Header, error) {
	return block_header.NewWithDB(m.db).FindLatestBlock()
}

func (m *manager) GetHeaderByNumber(number int64) (*model.Header, error) {
	return block_header.NewWithDB(m.db).FindBlockByNumber(number)
}

func (m *manager) GetTd(hash []byte) (*model.TotalDifficulty, error) {
	return block_header.NewWithDB(m.db).FindTd(hash)
}

// insertBlock inserts block, and accounts inside a DB transaction
func (m *manager) insertBlock(ctx context.Context, dbTx *gorm.DB, block *types.Block, receipts []*types.Receipt, dump *state.DirtyDump, ethEvents []*types.TransferLog) (err error) {
	headerStore := block_header.NewWithDB(dbTx)
	txStore := transaction.NewWithDB(dbTx)
	receiptStore := transaction_receipt.NewWithDB(dbTx)
	accountStore := account.NewWithDB(dbTx)
	subsStore := subscription.NewWithDB(dbTx)
	uncleStore := uncle_header.NewWithDB(dbTx)
	blockNumber := block.Number().Int64()
	txPrices := make(map[gethCommon.Hash]*big.Int)
	totalTxsFee := new(big.Int)

	// Insert txs
	var txs []*model.Transaction
	for _, t := range block.Transactions() {
		tx, err := common.Transaction(block, t)
		if err != nil {
			return err
		}
		txs = append(txs, tx)
		err = txStore.Insert(tx)
		if err != nil {
			return err
		}
		txPrices[t.Hash()] = t.GasPrice()
	}

	var events []*model.Transfer
	// Add eth transfer events
	for _, e := range ethEvents {
		events = append(events, common.EthTransferEvent(block, e))
	}

	// Insert tx receipts
	for _, receipt := range receipts {
		r, err := common.Receipt(block, receipt)
		if err != nil {
			return err
		}

		err = receiptStore.Insert(r)
		if err != nil {
			return err
		}

		// Collect erc20 events
		es, err := m.erc20Events(blockNumber, receipt.TxHash, r.Logs)
		if err != nil {
			return err
		}
		events = append(events, es...)
		totalTxsFee.Add(totalTxsFee, new(big.Int).Mul(txPrices[receipt.TxHash], new(big.Int).SetUint64(receipt.GasUsed)))
	}

	// Insert blocks
	minerBaseReward, uncleInclusionReward, unclesReward, unclesHash := common.AccumulateRewards(block.Header(), block.Uncles())
	h, err := common.Header(block).AddReward(totalTxsFee, minerBaseReward, uncleInclusionReward, unclesReward, unclesHash)
	if err != nil {
		return err
	}
	err = headerStore.Insert(h)
	if err != nil {
		return err
	}

	// Insert uncles
	for i, u := range block.Uncles() {
		err = uncleStore.Insert(common.UncleHeader(block, u, i).AddReward(unclesReward[i]))
		if err != nil {
			return err
		}
	}

	// Insert erc20 balance & events
	coinbases := make([]gethCommon.Address, 1+len(block.Uncles()))
	for _, u := range block.Uncles() {
		coinbases = append(coinbases, u.Coinbase)
	}
	coinbases = append(coinbases, block.Coinbase())
	err = newTransferProcessor(blockNumber, m.tokenList, receipts, txs, subsStore, accountStore, m.balancer).process(ctx, events, coinbases)
	if err != nil {
		return err
	}

	return nil
}

// delete deletes block and state data inside a DB transaction
func (m *manager) delete(dbTx *gorm.DB, from, to int64) (err error) {
	headerStore := block_header.NewWithDB(dbTx)
	uncleStore := uncle_header.NewWithDB(dbTx)
	txStore := transaction.NewWithDB(dbTx)
	receiptStore := transaction_receipt.NewWithDB(dbTx)
	accountStore := account.NewWithDB(dbTx)
	subscriptionStore := subscription.NewWithDB(dbTx)

	err = headerStore.Delete(from, to)
	if err != nil {
		return
	}
	err = uncleStore.DeleteByBlockNumber(from, to)
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

	err = subscriptionStore.Reset(from, to)
	if err != nil {
		return
	}

	for addr := range m.tokenList {
		// Delete erc20 balances
		err = accountStore.DeleteAccounts(addr, from, to)
		if err != nil {
			return
		}

		// Delete erc20 events
		err = accountStore.DeleteTransfer(addr, from, to)
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
